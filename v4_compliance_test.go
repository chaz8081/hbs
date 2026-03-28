package handlebars

import (
	"testing"
)

// Tests for bugs found during v4 review

func TestRemoveAllHelpers_ReregistersBuiltins(t *testing.T) {
	RemoveAllHelpers()

	// Built-in helpers should still work after RemoveAllHelpers
	tests := []struct {
		name     string
		input    string
		data     interface{}
		expected string
	}{
		{"if", `{{#if x}}yes{{/if}}`, map[string]bool{"x": true}, "yes"},
		{"unless", `{{#unless x}}yes{{/unless}}`, map[string]bool{"x": false}, "yes"},
		{"each", `{{#each items}}{{this}}{{/each}}`, map[string][]string{"items": {"a", "b"}}, "ab"},
		{"with", `{{#with user}}{{name}}{{/with}}`, map[string]interface{}{"user": map[string]string{"name": "Bob"}}, "Bob"},
		{"lookup", `{{lookup data "x"}}`, map[string]interface{}{"data": map[string]string{"x": "found"}}, "found"},
	}

	for _, tt := range tests {
		tpl := MustParse(tt.input)
		result := tpl.MustExec(tt.data)
		if result != tt.expected {
			t.Errorf("%s: expected %q, got %q", tt.name, tt.expected, result)
		}
	}
}

func TestRemoveAllDecorators_ReregistersInline(t *testing.T) {
	RemoveAllDecorators()

	tpl := MustParse(`{{#*inline "greeting"}}Hello{{/inline}}{{> greeting}}`)
	result := tpl.MustExec(nil)
	if result != "Hello" {
		t.Errorf("expected 'Hello', got %q", result)
	}
}

func TestIncludeZero_Float64(t *testing.T) {
	tpl := MustParse(`{{#if val includeZero=true}}shown{{else}}hidden{{/if}}`)
	result := tpl.MustExec(map[string]interface{}{"val": float64(0)})
	if result != "shown" {
		t.Errorf("expected 'shown' for float64(0) with includeZero, got %q", result)
	}
}

func TestIncludeZero_Int32(t *testing.T) {
	tpl := MustParse(`{{#if val includeZero=true}}shown{{else}}hidden{{/if}}`)
	result := tpl.MustExec(map[string]interface{}{"val": int32(0)})
	if result != "shown" {
		t.Errorf("expected 'shown' for int32(0) with includeZero, got %q", result)
	}
}

func TestIncludeZero_Uint(t *testing.T) {
	tpl := MustParse(`{{#if val includeZero=true}}shown{{else}}hidden{{/if}}`)
	result := tpl.MustExec(map[string]interface{}{"val": uint(0)})
	if result != "shown" {
		t.Errorf("expected 'shown' for uint(0) with includeZero, got %q", result)
	}
}

func TestIncludeZero_NonZeroFloat(t *testing.T) {
	tpl := MustParse(`{{#if val includeZero=true}}shown{{else}}hidden{{/if}}`)
	result := tpl.MustExec(map[string]interface{}{"val": float64(3.14)})
	if result != "shown" {
		t.Errorf("expected 'shown' for float64(3.14) with includeZero, got %q", result)
	}
}

func TestRawBlock_WithWhitespaceContent(t *testing.T) {
	tpl := MustParse("{{{{raw}}}} {{{{/raw}}}}")
	result := tpl.MustExec(nil)
	if result != " " {
		t.Errorf("expected single space for raw block with whitespace, got %q", result)
	}
}

func TestInlinePartial_NestedDefinition(t *testing.T) {
	// Inline partial defined inside another inline partial
	tpl := MustParse(`{{#*inline "outer"}}[{{#*inline "inner"}}INNER{{/inline}}{{> inner}}]{{/inline}}{{> outer}}`)
	result := tpl.MustExec(nil)
	if result != "[INNER]" {
		t.Errorf("expected '[INNER]', got %q", result)
	}
}

func TestBlockParams_WithHelper(t *testing.T) {
	tpl := MustParse(`{{#with user as |u|}}{{u.name}}{{/with}}`)
	data := map[string]interface{}{
		"user": map[string]string{"name": "Alice"},
	}
	result := tpl.MustExec(data)
	if result != "Alice" {
		t.Errorf("expected 'Alice', got %q", result)
	}
}

func TestBlockParams_Shadowing(t *testing.T) {
	tpl := MustParse(`{{#each outer as |item|}}{{#each inner as |item|}}{{item}}{{/each}}{{/each}}`)
	data := map[string]interface{}{
		"outer": []string{"X"},
		"inner": []string{"A", "B"},
	}
	result := tpl.MustExec(data)
	if result != "AB" {
		t.Errorf("expected 'AB' (inner shadows outer), got %q", result)
	}
}

func TestStrictMode_WithSubexpressions(t *testing.T) {
	tpl := MustParse(`{{#if (lookup data "key")}}yes{{/if}}`)
	tpl.SetStrict(true)
	tpl.RegisterHelper("lookup", func(obj interface{}, field string, options *Options) interface{} {
		return options.Eval(obj, field)
	})

	result := tpl.MustExec(map[string]interface{}{
		"data": map[string]string{"key": "value"},
	})
	if result != "yes" {
		t.Errorf("expected 'yes', got %q", result)
	}
}

func TestStrictMode_MissingVarErrors(t *testing.T) {
	tpl := MustParse(`{{missing}}`)
	tpl.SetStrict(true)
	_, err := tpl.Exec(map[string]string{"other": "value"})
	if err == nil {
		t.Error("expected error for missing variable in strict mode")
	}
}

func TestDynamicPartial_WithSubexpression(t *testing.T) {
	tpl := MustParse(`{{> (whichPartial)}}`)
	tpl.RegisterHelper("whichPartial", func() string {
		return "greeting"
	})
	tpl.RegisterPartial("greeting", "Hello!")

	result := tpl.MustExec(nil)
	if result != "Hello!" {
		t.Errorf("expected 'Hello!', got %q", result)
	}
}

func TestPartialBlock_NestedPartials(t *testing.T) {
	tpl := MustParse(`{{#> outer}}inner content{{/outer}}`)
	tpl.RegisterPartial("outer", "<outer>{{#> middle}}{{> @partial-block}}{{/middle}}</outer>")
	tpl.RegisterPartial("middle", "<middle>{{> @partial-block}}</middle>")

	result := tpl.MustExec(nil)
	if result != "<outer><middle>inner content</middle></outer>" {
		t.Errorf("expected nested partial blocks, got %q", result)
	}
}

func TestDecorator_MultipleInSequence(t *testing.T) {
	tpl := MustParse(`{{#*inline "a"}}A{{/inline}}{{#*inline "b"}}B{{/inline}}{{> a}}{{> b}}`)
	result := tpl.MustExec(nil)
	if result != "AB" {
		t.Errorf("expected 'AB', got %q", result)
	}
}

func TestElseIf_WithSubexpression(t *testing.T) {
	tpl := MustParse(`{{#if a}}A{{else if (lookup data "b")}}B{{else}}C{{/if}}`)
	tpl.RegisterHelper("lookup", func(obj interface{}, field string, options *Options) interface{} {
		return options.Eval(obj, field)
	})
	data := map[string]interface{}{
		"a":    false,
		"data": map[string]interface{}{"b": true},
	}
	result := tpl.MustExec(data)
	if result != "B" {
		t.Errorf("expected 'B', got %q", result)
	}
}

func TestEach_ElseBlock(t *testing.T) {
	tpl := MustParse(`{{#each items}}{{this}}{{else}}empty{{/each}}`)
	result := tpl.MustExec(map[string]interface{}{"items": []string{}})
	if result != "empty" {
		t.Errorf("expected 'empty', got %q", result)
	}
}
