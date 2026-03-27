package handlebars

import (
	"testing"
)

func TestPartialBlock_BasicWithPartialBlock(t *testing.T) {
	tpl := MustParse("{{#> layout}}content{{/layout}}")
	tpl.RegisterPartial("layout", "<div>{{> @partial-block}}</div>")

	result := tpl.MustExec(nil)
	if result != "<div>content</div>" {
		t.Errorf("expected '<div>content</div>', got %q", result)
	}
}

func TestPartialBlock_FallbackWhenPartialMissing(t *testing.T) {
	tpl := MustParse("{{#> nonexistent}}fallback content{{/nonexistent}}")

	result := tpl.MustExec(nil)
	if result != "fallback content" {
		t.Errorf("expected 'fallback content', got %q", result)
	}
}

func TestPartialBlock_WithContextData(t *testing.T) {
	tpl := MustParse("{{#> layout}}Hello {{name}}{{/layout}}")
	tpl.RegisterPartial("layout", "<main>{{> @partial-block}}</main>")

	result := tpl.MustExec(map[string]interface{}{"name": "World"})
	if result != "<main>Hello World</main>" {
		t.Errorf("expected '<main>Hello World</main>', got %q", result)
	}
}

func TestPartialBlock_ContentAroundPartialBlock(t *testing.T) {
	tpl := MustParse("{{#> page}}Page Body{{/page}}")
	tpl.RegisterPartial("page", "<header>Header</header>{{> @partial-block}}<footer>Footer</footer>")

	result := tpl.MustExec(nil)
	if result != "<header>Header</header>Page Body<footer>Footer</footer>" {
		t.Errorf("expected header+body+footer, got %q", result)
	}
}

func TestPartialBlock_PartialIgnoresBlockContent(t *testing.T) {
	tpl := MustParse("{{#> simple}}this is ignored{{/simple}}")
	tpl.RegisterPartial("simple", "just the partial")

	result := tpl.MustExec(nil)
	if result != "just the partial" {
		t.Errorf("expected 'just the partial', got %q", result)
	}
}

func TestPartialBlock_EmptyBlockContent(t *testing.T) {
	tpl := MustParse("{{#> layout}}{{/layout}}")
	tpl.RegisterPartial("layout", "before{{> @partial-block}}after")

	result := tpl.MustExec(nil)
	if result != "beforeafter" {
		t.Errorf("expected 'beforeafter', got %q", result)
	}
}
