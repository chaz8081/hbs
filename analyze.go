package handlebars

import (
	"strings"

	"github.com/chaz8081/handlebars-go/v4/ast"
)

// TemplateVar represents a variable referenced in a Handlebars template.
type TemplateVar struct {
	Path       string   // Dot-separated field path, e.g. "billing.plan"
	Required   bool     // True if not inside any conditional block
	Conditions []string // Condition paths that must be truthy for this variable to be needed
	Location   ast.Loc  // Position in template source
	Source     string   // Which template/partial, e.g. "body", "partial:header"
}

// builtInBlockHelpers are helpers that affect control flow / conditionality.
var builtInBlockHelpers = map[string]bool{
	"if":     true,
	"unless": true,
	"with":   true,
	"each":   true,
}

// ExtractVariables walks the template's AST and returns all referenced variables
// as a flat list. The helpers parameter provides known helper names for disambiguation:
// if a bare {{foo}} matches a known helper, it's excluded from the variable list.
// Pass nil if no helper disambiguation is needed.
func ExtractVariables(tpl *Template, helpers map[string]bool) []TemplateVar {
	program := tpl.AST()
	if program == nil {
		return nil
	}

	e := &varExtractor{
		helpers: helpers,
		seen:    make(map[string]*TemplateVar),
		source:  "body",
	}

	e.walkProgram(program)

	// Convert map to slice, preserving discovery order
	result := make([]TemplateVar, 0, len(e.order))
	for _, path := range e.order {
		result = append(result, *e.seen[path])
	}
	return result
}

// varExtractor walks the AST to collect template variables.
type varExtractor struct {
	helpers    map[string]bool
	conditions []string // stack of condition paths
	seen       map[string]*TemplateVar
	order      []string // insertion order for deterministic output
	source     string
}

// addVar records a variable, deduplicating by path.
// If the variable was already seen, it upgrades Required to true if the new
// occurrence is unconditional.
func (e *varExtractor) addVar(path string, loc ast.Loc) {
	if path == "" || path == "this" || path == "." {
		return
	}

	// Skip @data variables
	if strings.HasPrefix(path, "@") {
		return
	}

	required := len(e.conditions) == 0
	conditions := make([]string, len(e.conditions))
	copy(conditions, e.conditions)

	if existing, ok := e.seen[path]; ok {
		// Upgrade to required if this occurrence is unconditional
		if required {
			existing.Required = true
			existing.Conditions = nil
		}
		return
	}

	v := &TemplateVar{
		Path:       path,
		Required:   required,
		Conditions: conditions,
		Location:   loc,
		Source:     e.source,
	}
	e.seen[path] = v
	e.order = append(e.order, path)
}

// isHelper checks if a name is a known helper (built-in or user-provided).
func (e *varExtractor) isHelper(name string) bool {
	if builtInBlockHelpers[name] {
		return true
	}
	// Also check built-in non-block helpers
	switch name {
	case "lookup", "log", "equal":
		return true
	}
	if e.helpers != nil {
		return e.helpers[name]
	}
	return false
}

func (e *varExtractor) walkProgram(node *ast.Program) {
	for _, n := range node.Body {
		e.walkNode(n)
	}
}

func (e *varExtractor) walkNode(node ast.Node) {
	switch n := node.(type) {
	case *ast.MustacheStatement:
		e.walkMustache(n)
	case *ast.BlockStatement:
		e.walkBlock(n)
	case *ast.PartialStatement:
		e.walkPartial(n)
	case *ast.ContentStatement:
		// no variables in content
	case *ast.CommentStatement:
		// no variables in comments
	}
}

func (e *varExtractor) walkMustache(node *ast.MustacheStatement) {
	e.walkExpression(node.Expression)
}

func (e *varExtractor) walkExpression(node *ast.Expression) {
	hasParams := len(node.Params) > 0
	hasHash := node.Hash != nil

	if path, ok := node.Path.(*ast.PathExpression); ok {
		helperName := node.HelperName()

		if hasParams || hasHash {
			// Unambiguously a helper call — extract variables from params and hash only
			e.walkExpressionParams(node)
		} else if helperName != "" && e.isHelper(helperName) {
			// Zero-arg helper — skip (not a variable)
		} else {
			// Variable reference
			e.addVar(path.Original, path.Location())
		}
	} else {
		// Path is a literal (string, number, boolean) — no variable
		// But still check params/hash
		e.walkExpressionParams(node)
	}
}

func (e *varExtractor) walkExpressionParams(node *ast.Expression) {
	for _, param := range node.Params {
		e.walkParamNode(param)
	}
	if node.Hash != nil {
		e.walkHash(node.Hash)
	}
}

func (e *varExtractor) walkParamNode(node ast.Node) {
	switch n := node.(type) {
	case *ast.PathExpression:
		if !n.Data { // skip @data variables
			e.addVar(n.Original, n.Location())
		}
	case *ast.SubExpression:
		// Subexpression — walk the inner expression
		e.walkExpression(n.Expression)
	case *ast.StringLiteral, *ast.BooleanLiteral, *ast.NumberLiteral:
		// Literals — no variables
	}
}

func (e *varExtractor) walkHash(node *ast.Hash) {
	for _, pair := range node.Pairs {
		e.walkParamNode(pair.Val)
	}
}

func (e *varExtractor) walkBlock(node *ast.BlockStatement) {
	helperName := node.Expression.HelperName()

	switch helperName {
	case "if", "unless":
		e.walkConditionalBlock(node)
	case "each":
		e.walkEachBlock(node)
	case "with":
		e.walkWithBlock(node)
	default:
		// Custom block helper or expression-based block
		e.walkExpression(node.Expression)
		if node.Program != nil {
			e.walkProgram(node.Program)
		}
		if node.Inverse != nil {
			e.walkProgram(node.Inverse)
		}
	}
}

func (e *varExtractor) walkConditionalBlock(node *ast.BlockStatement) {
	// The condition itself is a required variable (or a helper call with params)
	var condPath string

	if len(node.Expression.Params) > 0 {
		// {{#if condition}} — condition is the first param
		if path, ok := node.Expression.Params[0].(*ast.PathExpression); ok {
			condPath = path.Original
			if !path.Data {
				e.addVar(condPath, path.Location())
			}
		}
		// Also walk any additional params (e.g., subexpressions in condition)
		for _, param := range node.Expression.Params {
			if _, ok := param.(*ast.SubExpression); ok {
				e.walkParamNode(param)
			}
		}
	}

	// Walk the main block with condition pushed
	if node.Program != nil {
		if condPath != "" {
			e.conditions = append(e.conditions, condPath)
		}
		e.walkProgram(node.Program)
		if condPath != "" {
			e.conditions = e.conditions[:len(e.conditions)-1]
		}
	}

	// Walk the inverse/else block with condition pushed
	// (else variables are also conditional — they only matter when the condition is checked)
	if node.Inverse != nil {
		if condPath != "" {
			e.conditions = append(e.conditions, condPath)
		}
		e.walkProgram(node.Inverse)
		if condPath != "" {
			e.conditions = e.conditions[:len(e.conditions)-1]
		}
	}
}

func (e *varExtractor) walkEachBlock(node *ast.BlockStatement) {
	// The iteration target is a required variable
	if len(node.Expression.Params) > 0 {
		if path, ok := node.Expression.Params[0].(*ast.PathExpression); ok {
			if !path.Data {
				e.addVar(path.Original, path.Location())
			}
		}
		// Walk subexpression params
		for _, param := range node.Expression.Params {
			if _, ok := param.(*ast.SubExpression); ok {
				e.walkParamNode(param)
			}
		}
	}

	// Walk the body — variables inside #each refer to iteration context,
	// so we don't push conditions (items are iterated, not conditional)
	if node.Program != nil {
		e.walkProgram(node.Program)
	}
	if node.Inverse != nil {
		e.walkProgram(node.Inverse)
	}
}

func (e *varExtractor) walkWithBlock(node *ast.BlockStatement) {
	// The context target is a required variable
	if len(node.Expression.Params) > 0 {
		if path, ok := node.Expression.Params[0].(*ast.PathExpression); ok {
			if !path.Data {
				e.addVar(path.Original, path.Location())
			}
		}
	}

	// Walk body — variables inside #with refer to the shifted context
	if node.Program != nil {
		e.walkProgram(node.Program)
	}
	if node.Inverse != nil {
		e.walkProgram(node.Inverse)
	}
}

func (e *varExtractor) walkPartial(node *ast.PartialStatement) {
	// Extract variables from partial params
	for _, param := range node.Params {
		e.walkParamNode(param)
	}

	// Extract variables from partial hash params
	if node.Hash != nil {
		e.walkHash(node.Hash)
	}
}
