package handlebars

import (
	"fmt"
	"testing"
)

func TestHelperMissing_NotCalled_WhenHelperExists(t *testing.T) {
	RegisterHelper("existingHelper", func() string { return "found" })
	defer RemoveHelper("existingHelper")
	RegisterHelper("helperMissing", func(options *Options) string { return "MISSING" })
	defer RemoveHelper("helperMissing")

	tpl := MustParse("{{existingHelper}}")
	result := tpl.MustExec(nil)
	if result != "found" {
		t.Errorf("expected 'found', got %q", result)
	}
}

func TestHelperMissing_Called_WhenHelperNotFound(t *testing.T) {
	RegisterHelper("helperMissing", func(options *Options) string {
		return fmt.Sprintf("missing:%s", options.Name())
	})
	defer RemoveHelper("helperMissing")

	tpl := MustParse("{{unknownHelper arg1}}")
	result := tpl.MustExec(map[string]interface{}{"arg1": "val"})
	if result != "missing:unknownHelper" {
		t.Errorf("expected 'missing:unknownHelper', got %q", result)
	}
}

func TestHelperMissing_ReceivesParams(t *testing.T) {
	RegisterHelper("helperMissing", func(options *Options) string {
		return fmt.Sprintf("missing:%s(%d params)", options.Name(), len(options.Params()))
	})
	defer RemoveHelper("helperMissing")

	tpl := MustParse("{{unknownHelper a b c}}")
	result := tpl.MustExec(map[string]interface{}{"a": 1, "b": 2, "c": 3})
	if result != "missing:unknownHelper(3 params)" {
		t.Errorf("expected 'missing:unknownHelper(3 params)', got %q", result)
	}
}

func TestHelperMissing_DefaultBehavior(t *testing.T) {
	tpl := MustParse("{{name}}")
	result := tpl.MustExec(map[string]interface{}{"name": "Alice"})
	if result != "Alice" {
		t.Errorf("expected 'Alice', got %q", result)
	}
}

func TestBlockHelperMissing_Called(t *testing.T) {
	RegisterHelper("blockHelperMissing", func(context interface{}, options *Options) string {
		return fmt.Sprintf("block-missing:%s", options.Name())
	})
	defer RemoveHelper("blockHelperMissing")

	tpl := MustParse("{{#unknownBlock}}content{{/unknownBlock}}")
	result := tpl.MustExec(nil)
	if result != "block-missing:unknownBlock" {
		t.Errorf("expected 'block-missing:unknownBlock', got %q", result)
	}
}

func TestBlockHelperMissing_NotCalled_WhenExists(t *testing.T) {
	RegisterHelper("myBlock", func(options *Options) string { return options.Fn() })
	defer RemoveHelper("myBlock")
	called := false
	RegisterHelper("blockHelperMissing", func(context interface{}, options *Options) string {
		called = true
		return ""
	})
	defer RemoveHelper("blockHelperMissing")

	tpl := MustParse("{{#myBlock}}content{{/myBlock}}")
	result := tpl.MustExec(nil)
	if called {
		t.Error("blockHelperMissing should not be called when block helper exists")
	}
	if result != "content" {
		t.Errorf("expected 'content', got %q", result)
	}
}
