package handlebars

import (
	"testing"
)

func TestWhitespace_TildeElseIf_Basic(t *testing.T) {
	tpl := MustParse("{{~#if a~}}yes{{~else if b~}}maybe{{~else~}}no{{~/if~}}")

	// a=false, b=true → "maybe"
	result, err := tpl.Exec(map[string]interface{}{"a": false, "b": true})
	if err != nil {
		t.Fatal(err)
	}
	if result != "maybe" {
		t.Errorf("expected 'maybe', got %q", result)
	}
}

func TestWhitespace_TildeElseIf_FirstBranch(t *testing.T) {
	tpl := MustParse("{{~#if a~}}yes{{~else if b~}}maybe{{~else~}}no{{~/if~}}")

	// a=true → "yes"
	result, err := tpl.Exec(map[string]interface{}{"a": true, "b": false})
	if err != nil {
		t.Fatal(err)
	}
	if result != "yes" {
		t.Errorf("expected 'yes', got %q", result)
	}
}

func TestWhitespace_TildeElseIf_ElseBranch(t *testing.T) {
	tpl := MustParse("{{~#if a~}}yes{{~else if b~}}maybe{{~else~}}no{{~/if~}}")

	// a=false, b=false → "no"
	result, err := tpl.Exec(map[string]interface{}{"a": false, "b": false})
	if err != nil {
		t.Fatal(err)
	}
	if result != "no" {
		t.Errorf("expected 'no', got %q", result)
	}
}

func TestWhitespace_TildeElseIf_WithNewlines(t *testing.T) {
	tpl := MustParse("before\n{{~#if a~}}\nyes\n{{~else if b~}}\nmaybe\n{{~/if~}}\nafter")

	result, err := tpl.Exec(map[string]interface{}{"a": false, "b": true})
	if err != nil {
		t.Fatal(err)
	}
	if result != "beforemaybeafter" {
		t.Errorf("expected 'beforemaybeafter', got %q", result)
	}
}

func TestWhitespace_TildeElseIf_MultipleChain(t *testing.T) {
	tpl := MustParse("{{~#if a~}}A{{~else if b~}}B{{~else if c~}}C{{~else~}}D{{~/if~}}")

	result, err := tpl.Exec(map[string]interface{}{"a": false, "b": false, "c": true})
	if err != nil {
		t.Fatal(err)
	}
	if result != "C" {
		t.Errorf("expected 'C', got %q", result)
	}
}

func TestWhitespace_ElseIf_StandaloneNewlines(t *testing.T) {
	// Standalone else-if should strip surrounding newlines
	source := "start\n{{#if a}}\nyes\n{{else if b}}\nmaybe\n{{/if}}\nend"
	tpl := MustParse(source)

	result, err := tpl.Exec(map[string]interface{}{"a": false, "b": true})
	if err != nil {
		t.Fatal(err)
	}
	if result != "start\nmaybe\nend" {
		t.Errorf("expected 'start\\nmaybe\\nend', got %q", result)
	}
}

func TestWhitespace_ElseIf_NoTilde_Spacing(t *testing.T) {
	// Without tilde, spaces should be preserved
	tpl := MustParse("{{#if a}}yes{{else if b}}maybe{{else}}no{{/if}}")

	result, err := tpl.Exec(map[string]interface{}{"a": false, "b": true})
	if err != nil {
		t.Fatal(err)
	}
	if result != "maybe" {
		t.Errorf("expected 'maybe', got %q", result)
	}
}
