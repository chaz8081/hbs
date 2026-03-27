package handlebars

import (
	"testing"
)

// Edge cases that are more likely to trigger panics

type nilEdgeContainer struct {
	Items *[]string
}

type nilEdgeNested struct {
	Data map[string]interface{}
}

func TestNilSafety_NilPointerToSlice_Each(t *testing.T) {
	// Typed nil pointer to slice in #each
	tpl := MustParse("{{#each container.items}}{{this}}{{else}}none{{/each}}")
	ctx := map[string]interface{}{
		"container": nilEdgeContainer{Items: nil},
	}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "none" {
		t.Errorf("expected 'none', got %q", result)
	}
}

func TestNilSafety_NilPointerContext(t *testing.T) {
	// Rendering with a typed nil pointer as context
	tpl := MustParse("hello {{name}}")
	ctx := (*nilTestUser)(nil)
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "hello " {
		t.Errorf("expected 'hello ', got %q", result)
	}
}

func TestNilSafety_DoublePointerNil(t *testing.T) {
	// Double pointer where outer is non-nil but inner is nil
	var inner *nilTestUser
	outer := &inner
	tpl := MustParse("{{name}}")
	result := mustRenderNoPanic(t, tpl, outer)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestNilSafety_NilMapInStruct(t *testing.T) {
	// Struct with nil map field
	tpl := MustParse("{{nested.data.key}}")
	ctx := map[string]interface{}{
		"nested": nilEdgeNested{Data: nil},
	}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestNilSafety_WithNilPointerContext(t *testing.T) {
	// #with receives a typed nil pointer and should use else
	tpl := MustParse("{{#with user}}{{name}}{{else}}no user{{/with}}")
	ctx := map[string]interface{}{
		"user": (*nilTestUser)(nil),
	}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "no user" {
		t.Errorf("expected 'no user', got %q", result)
	}
}

func TestNilSafety_EachOnNilPointerToSlice(t *testing.T) {
	// #each on a value that resolves to typed nil slice pointer
	var items *[]string
	tpl := MustParse("{{#each items}}{{this}}{{else}}empty{{/each}}")
	ctx := map[string]interface{}{"items": items}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "empty" {
		t.Errorf("expected 'empty', got %q", result)
	}
}

func TestNilSafety_NestedBlocksWithNil(t *testing.T) {
	// Nested blocks where inner path is nil
	tpl := MustParse("{{#if user}}{{#if user.address}}has addr{{else}}no addr{{/if}}{{else}}no user{{/if}}")
	ctx := map[string]interface{}{
		"user": nilTestUser{Name: "Alice", Address: nil},
	}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "no addr" {
		t.Errorf("expected 'no addr', got %q", result)
	}
}

func TestNilSafety_LookupOnNil(t *testing.T) {
	// lookup helper on nil value
	tpl := MustParse("{{lookup foo 0}}")
	ctx := map[string]interface{}{"foo": nil}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestNilSafety_NilInEachIteration(t *testing.T) {
	// Array containing nil elements
	tpl := MustParse("{{#each items}}[{{this.name}}]{{/each}}")
	ctx := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "a"},
			nil,
			map[string]interface{}{"name": "c"},
		},
	}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "[a][][c]" {
		t.Errorf("expected '[a][][c]', got %q", result)
	}
}

func TestNilSafety_PartialWithNilContext(t *testing.T) {
	// Partial invoked with nil context
	tpl := MustParse("{{> myPartial user}}")
	tpl.RegisterPartial("myPartial", "{{name}}")
	ctx := map[string]interface{}{"user": nil}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}
