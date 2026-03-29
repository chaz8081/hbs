package handlebars

import (
	"testing"

	"github.com/chaz8081/hbs/v4/ast"
	"github.com/chaz8081/hbs/v4/parser"
)

func TestTemplate_AST_ReturnsProgram(t *testing.T) {
	tpl := MustParse("Hello {{name}}")
	program := tpl.AST()
	if program == nil {
		t.Fatal("AST() returned nil")
	}
	if len(program.Body) != 2 {
		t.Errorf("expected 2 body nodes, got %d", len(program.Body))
	}
}

func TestTemplate_AST_MatchesParserOutput(t *testing.T) {
	source := "{{#if show}}yes{{else}}no{{/if}}"
	tpl := MustParse(source)
	program := tpl.AST()

	// Compare with direct parser output
	directProgram, err := parser.Parse(source)
	if err != nil {
		t.Fatal(err)
	}

	// Both should produce the same AST string representation
	if ast.Print(program) != ast.Print(directProgram) {
		t.Errorf("AST() output differs from parser.Parse() output")
	}
}

func TestTemplate_AST_BlockStatement(t *testing.T) {
	tpl := MustParse("{{#each items}}{{name}}{{/each}}")
	program := tpl.AST()

	if len(program.Body) != 1 {
		t.Fatalf("expected 1 body node, got %d", len(program.Body))
	}
	block, ok := program.Body[0].(*ast.BlockStatement)
	if !ok {
		t.Fatalf("expected BlockStatement, got %T", program.Body[0])
	}
	if block.Expression == nil {
		t.Fatal("block expression is nil")
	}
}

func TestNewTemplateFromAST_BasicRender(t *testing.T) {
	// Parse a source to get an AST
	program, err := parser.Parse("Hello {{name}}")
	if err != nil {
		t.Fatal(err)
	}

	// Create template from AST
	tpl := NewTemplateFromAST(program)
	if tpl == nil {
		t.Fatal("NewTemplateFromAST returned nil")
	}

	// Render it
	result, err := tpl.Exec(map[string]string{"name": "World"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", result)
	}
}

func TestNewTemplateFromAST_WithHelpers(t *testing.T) {
	program, err := parser.Parse("{{shout name}}")
	if err != nil {
		t.Fatal(err)
	}

	tpl := NewTemplateFromAST(program)
	tpl.RegisterHelper("shout", func(s string) string {
		return "[" + s + "]"
	})

	result, err := tpl.Exec(map[string]string{"name": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "[hello]" {
		t.Errorf("expected '[hello]', got %q", result)
	}
}

func TestNewTemplateFromAST_WithPartials(t *testing.T) {
	program, err := parser.Parse("before-{{> myPartial}}-after")
	if err != nil {
		t.Fatal(err)
	}

	tpl := NewTemplateFromAST(program)
	tpl.RegisterPartial("myPartial", "PARTIAL")

	result, err := tpl.Exec(nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "before-PARTIAL-after" {
		t.Errorf("expected 'before-PARTIAL-after', got %q", result)
	}
}

func TestNewTemplateFromAST_RoundTrip(t *testing.T) {
	// Parse → AST → NewTemplateFromAST → Exec should match Parse → Exec
	source := "{{#if active}}{{name}} is active{{else}}inactive{{/if}}"
	ctx := map[string]interface{}{"active": true, "name": "Alice"}

	// Original path
	tpl1 := MustParse(source)
	result1, err := tpl1.Exec(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Round-trip through AST
	program := tpl1.AST()
	tpl2 := NewTemplateFromAST(program)
	result2, err := tpl2.Exec(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if result1 != result2 {
		t.Errorf("round-trip mismatch: %q vs %q", result1, result2)
	}
}

func TestTemplate_AST_IsReadOnly(t *testing.T) {
	// Modifying the returned AST should not affect the template's behavior
	// (or if it does, that's a documented choice — this test documents current behavior)
	tpl := MustParse("Hello {{name}}")
	program := tpl.AST()

	// Verify original works
	result1, _ := tpl.Exec(map[string]string{"name": "World"})
	if result1 != "Hello World" {
		t.Fatalf("initial render failed: %q", result1)
	}

	// Note: AST() returns the same pointer (not a deep copy).
	// This test documents that the pointer is shared.
	if program != tpl.AST() {
		t.Error("AST() returned different pointer on second call")
	}
}
