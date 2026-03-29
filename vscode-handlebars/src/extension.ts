import * as vscode from 'vscode';
import * as cp from 'child_process';
import * as path from 'path';

const DIAGNOSTICS_SOURCE = 'hbs';
let diagnosticCollection: vscode.DiagnosticCollection;

// Built-in helpers and their documentation
const BUILTIN_HELPERS: Map<string, HelperDoc> = new Map([
    ['if', {
        snippet: '{{#if ${1:condition}}}\n\t$0\n{{/if}}',
        detail: 'Conditionally render a block',
        documentation: 'Renders the block if the argument is truthy. Supports `{{else}}` and chained `{{else if}}`.',
    }],
    ['unless', {
        snippet: '{{#unless ${1:condition}}}\n\t$0\n{{/unless}}',
        detail: 'Inverse conditional block',
        documentation: 'Renders the block if the argument is falsy. Inverse of `#if`.',
    }],
    ['each', {
        snippet: '{{#each ${1:array}}}\n\t$0\n{{/each}}',
        detail: 'Iterate over a list or object',
        documentation: 'Iterates over each item. Inside the block, `this` refers to the current element.\n\nData variables: `@index`, `@key`, `@first`, `@last`.\n\nSupports `{{else}}` for empty collections and `as |item index|` block params.',
    }],
    ['with', {
        snippet: '{{#with ${1:context}}}\n\t$0\n{{/with}}',
        detail: 'Change context for a block',
        documentation: 'Shifts the context for the block to the given object.\n\nSupports `{{else}}` for falsy/missing context and `as |alias|` block params.',
    }],
    ['lookup', {
        snippet: '{{lookup ${1:object} ${2:key}}}',
        detail: 'Dynamic property lookup',
        documentation: 'Looks up a property by name at runtime. Useful for dynamic keys.\n\nExample: `{{lookup person @key}}`\n\nPreserves the value type (does not convert to string).',
    }],
    ['log', {
        snippet: '{{log ${1:message}}}',
        detail: 'Log a value',
        documentation: 'Outputs a log message. Supports a `level` hash parameter.\n\nExample: `{{log "debug info" level="warn"}}`\n\nLevels: debug, info, warn, error.',
    }],
]);

interface HelperDoc {
    snippet: string;
    detail: string;
    documentation: string;
}

export function activate(context: vscode.ExtensionContext) {
    diagnosticCollection = vscode.languages.createDiagnosticCollection(DIAGNOSTICS_SOURCE);
    context.subscriptions.push(diagnosticCollection);

    // Completions for built-in helpers
    const completionProvider = vscode.languages.registerCompletionItemProvider(
        'handlebars',
        new HandlebarsCompletionProvider(),
        '{', '#'
    );
    context.subscriptions.push(completionProvider);

    // Hover documentation
    const hoverProvider = vscode.languages.registerHoverProvider(
        'handlebars',
        new HandlebarsHoverProvider()
    );
    context.subscriptions.push(hoverProvider);

    // Lint on save
    const onSave = vscode.workspace.onDidSaveTextDocument((doc) => {
        if (doc.languageId === 'handlebars') {
            const config = vscode.workspace.getConfiguration('handlebarsGo');
            if (config.get<boolean>('lintOnSave', true)) {
                lintDocument(doc);
            }
        }
    });
    context.subscriptions.push(onSave);

    // Lint on open
    const onOpen = vscode.workspace.onDidOpenTextDocument((doc) => {
        if (doc.languageId === 'handlebars') {
            const config = vscode.workspace.getConfiguration('handlebarsGo');
            if (config.get<boolean>('lintOnSave', true)) {
                lintDocument(doc);
            }
        }
    });
    context.subscriptions.push(onOpen);

    // Clear diagnostics when document is closed
    const onClose = vscode.workspace.onDidCloseTextDocument((doc) => {
        diagnosticCollection.delete(doc.uri);
    });
    context.subscriptions.push(onClose);

    // Lint already open handlebars files
    vscode.workspace.textDocuments.forEach((doc) => {
        if (doc.languageId === 'handlebars') {
            lintDocument(doc);
        }
    });
}

class HandlebarsCompletionProvider implements vscode.CompletionItemProvider {
    provideCompletionItems(
        document: vscode.TextDocument,
        position: vscode.Position
    ): vscode.CompletionItem[] {
        const lineText = document.lineAt(position).text;
        const textBefore = lineText.substring(0, position.character);

        // Check if we're inside a handlebars expression
        const inExpression = /\{\{[#^/]?\s*\w*$/.test(textBefore);
        if (!inExpression) {
            return [];
        }

        const items: vscode.CompletionItem[] = [];

        // Built-in helpers
        for (const [name, doc] of BUILTIN_HELPERS) {
            const item = new vscode.CompletionItem(name, vscode.CompletionItemKind.Function);
            item.detail = doc.detail;
            item.documentation = new vscode.MarkdownString(doc.documentation);
            item.insertText = new vscode.SnippetString(doc.snippet);
            items.push(item);
        }

        // User-configured custom helpers
        const config = vscode.workspace.getConfiguration('handlebarsGo');
        const customHelpers = config.get<string[]>('helpers', []);
        for (const helper of customHelpers) {
            const item = new vscode.CompletionItem(helper, vscode.CompletionItemKind.Function);
            item.detail = 'Custom helper';
            items.push(item);
        }

        // Data variables
        const dataVars = ['@index', '@key', '@first', '@last', '@root', '@level'];
        for (const dv of dataVars) {
            const item = new vscode.CompletionItem(dv, vscode.CompletionItemKind.Variable);
            item.detail = 'Data variable';
            items.push(item);
        }

        return items;
    }
}

class HandlebarsHoverProvider implements vscode.HoverProvider {
    provideHover(
        document: vscode.TextDocument,
        position: vscode.Position
    ): vscode.Hover | undefined {
        const range = document.getWordRangeAtPosition(position, /\b\w+\b/);
        if (!range) {
            return undefined;
        }

        const word = document.getText(range);
        const lineText = document.lineAt(position).text;
        const textBefore = lineText.substring(0, range.start.character);

        // Only show hover inside handlebars expressions
        const openCount = (textBefore.match(/\{\{/g) || []).length;
        const closeCount = (textBefore.match(/\}\}/g) || []).length;
        if (openCount <= closeCount) {
            return undefined;
        }

        const helper = BUILTIN_HELPERS.get(word);
        if (!helper) {
            return undefined;
        }

        const md = new vscode.MarkdownString();
        md.appendMarkdown(`**${word}** — ${helper.detail}\n\n`);
        md.appendMarkdown(helper.documentation);
        return new vscode.Hover(md, range);
    }
}

interface LintResult {
    file: string;
    errors: LintError[];
    valid: boolean;
}

interface LintError {
    type: string;
    message: string;
    path?: string;
}

function findLintBinary(): string | undefined {
    const config = vscode.workspace.getConfiguration('handlebarsGo');
    const configPath = config.get<string>('lintBinary', '');
    if (configPath) {
        return configPath;
    }

    // Try to find in PATH or Go bin
    try {
        const result = cp.execSync('which handlebars-lint 2>/dev/null || go env GOPATH', {
            encoding: 'utf-8',
            timeout: 5000,
        }).trim();

        if (result.endsWith('handlebars-lint')) {
            return result;
        }

        // Try GOPATH/bin
        const gopath = result.split('\n').pop() || '';
        const goBin = path.join(gopath, 'bin', 'handlebars-lint');
        try {
            cp.execSync(`test -x "${goBin}"`, { timeout: 2000 });
            return goBin;
        } catch {
            return undefined;
        }
    } catch {
        return undefined;
    }
}

function lintDocument(document: vscode.TextDocument) {
    const binary = findLintBinary();
    if (!binary) {
        return;
    }

    const args = ['--json'];

    const config = vscode.workspace.getConfiguration('handlebarsGo');
    const dataFile = config.get<string>('dataFile', '');
    if (dataFile) {
        args.push('--data', dataFile);
    }

    const helpers = config.get<string[]>('helpers', []);
    if (helpers.length > 0) {
        args.push('--helpers', helpers.join(','));
    }

    args.push(document.fileName);

    cp.exec(`"${binary}" ${args.join(' ')}`, { timeout: 10000 }, (err, stdout) => {
        const diagnostics: vscode.Diagnostic[] = [];

        if (stdout) {
            try {
                const results: LintResult[] = JSON.parse(stdout);
                for (const result of results) {
                    for (const lintErr of result.errors) {
                        const diagnostic = new vscode.Diagnostic(
                            new vscode.Range(0, 0, 0, 0), // Line info not available yet
                            lintErr.message,
                            lintErr.type === 'parse'
                                ? vscode.DiagnosticSeverity.Error
                                : vscode.DiagnosticSeverity.Warning
                        );
                        diagnostic.source = DIAGNOSTICS_SOURCE;
                        diagnostics.push(diagnostic);
                    }
                }
            } catch {
                // JSON parse failure — ignore
            }
        }

        diagnosticCollection.set(document.uri, diagnostics);
    });
}

export function deactivate() {
    if (diagnosticCollection) {
        diagnosticCollection.dispose();
    }
}
