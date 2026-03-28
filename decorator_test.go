package handlebars

import (
	"strings"
	"testing"
)

func TestDecorator_InlinePartial(t *testing.T) {
	tpl := MustParse(`{{#*inline "myPartial"}}hello{{/inline}}{{> myPartial}}`)
	result := tpl.MustExec(nil)
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestDecorator_CustomDecorator(t *testing.T) {
	RegisterDecorator("uppercase", func(options *DecoratorOptions) interface{} {
		// This decorator renders its block content and registers it as
		// an inline partial with the given name, uppercased.
		name := options.ParamStr(0)
		content := strings.ToUpper(options.Fn())
		options.RegisterInlinePartial(name, content)
		return ""
	})
	defer RemoveDecorator("uppercase")

	tpl := MustParse(`{{#*uppercase "shout"}}hello world{{/uppercase}}{{> shout}}`)
	result := tpl.MustExec(nil)
	if result != "HELLO WORLD" {
		t.Errorf("expected 'HELLO WORLD', got %q", result)
	}
}

func TestDecorator_TemplateLevel(t *testing.T) {
	tpl := MustParse(`{{#*wrap "greeting"}}hi{{/wrap}}{{> greeting}}`)
	tpl.RegisterDecorator("wrap", func(options *DecoratorOptions) interface{} {
		name := options.ParamStr(0)
		content := "[" + options.Fn() + "]"
		options.RegisterInlinePartial(name, content)
		return ""
	})
	result := tpl.MustExec(nil)
	if result != "[hi]" {
		t.Errorf("expected '[hi]', got %q", result)
	}
}

func TestDecorator_WithHashParams(t *testing.T) {
	RegisterDecorator("meta", func(options *DecoratorOptions) interface{} {
		prefix := options.HashStr("prefix")
		name := options.ParamStr(0)
		content := prefix + options.Fn()
		options.RegisterInlinePartial(name, content)
		return ""
	})
	defer RemoveDecorator("meta")

	tpl := MustParse(`{{#*meta "tagged" prefix=">> "}}content{{/meta}}{{> tagged}}`)
	result := tpl.MustExec(nil)
	if result != ">> content" {
		t.Errorf("expected '>> content', got %q", result)
	}
}

func TestDecorator_UnknownDecorator(t *testing.T) {
	_, err := MustParse(`{{#*unknown "foo"}}bar{{/unknown}}`).ExecWith(nil, nil)
	if err == nil {
		t.Error("expected error for unknown decorator, got nil")
	}
}
