package handlebars

import (
	"testing"
)

func TestRawBlock_Basic(t *testing.T) {
	tpl := MustParse("{{{{raw}}}}{{escaped}}{{{{/raw}}}}")
	result := tpl.MustExec(nil)
	if result != "{{escaped}}" {
		t.Errorf("expected '{{escaped}}', got %q", result)
	}
}

func TestRawBlock_WithContent(t *testing.T) {
	tpl := MustParse("before{{{{raw}}}}{{hello}} world{{{{/raw}}}}after")
	result := tpl.MustExec(nil)
	if result != "before{{hello}} worldafter" {
		t.Errorf("expected 'before{{hello}} worldafter', got %q", result)
	}
}

func TestRawBlock_PreservesHandlebarsExpressions(t *testing.T) {
	tpl := MustParse("{{{{raw}}}}{{#if true}}yes{{/if}}{{{{/raw}}}}")
	result := tpl.MustExec(nil)
	if result != "{{#if true}}yes{{/if}}" {
		t.Errorf("expected '{{#if true}}yes{{/if}}', got %q", result)
	}
}

func TestRawBlock_CustomName(t *testing.T) {
	RegisterHelper("myraw", func(options *Options) string {
		return options.Fn()
	})
	defer RemoveHelper("myraw")

	tpl := MustParse("{{{{myraw}}}}{{stuff}}{{{{/myraw}}}}")
	result := tpl.MustExec(nil)
	if result != "{{stuff}}" {
		t.Errorf("expected '{{stuff}}', got %q", result)
	}
}
