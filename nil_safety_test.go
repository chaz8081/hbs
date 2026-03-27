package handlebars

import (
	"testing"
)

// Test nil-safe evaluation: Handlebars.js treats nil/undefined path segments as falsy
// and renders empty string. Raymond/flowchartsman panics with nil pointer dereference.

type nilTestAddress struct {
	City  string
	State string
}

type nilTestUser struct {
	Name    string
	Address *nilTestAddress
}

type nilTestOuter struct {
	Inner *nilTestMiddle
}

type nilTestMiddle struct {
	Leaf *nilTestLeaf
}

type nilTestLeaf struct {
	Value string
}

func TestNilSafety_MapNilValue(t *testing.T) {
	// Accessing a field on a nil map value should return empty string, not panic
	tpl := MustParse("{{foo.bar}}")
	result := mustRenderNoPanic(t, tpl, map[string]interface{}{"foo": nil})
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestNilSafety_MapNilValueNested(t *testing.T) {
	// Deeply nested access through nil should return empty string
	tpl := MustParse("{{a.b.c.d}}")
	result := mustRenderNoPanic(t, tpl, map[string]interface{}{"a": nil})
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestNilSafety_StructNilPointerField(t *testing.T) {
	// Accessing field through nil struct pointer should return empty string
	tpl := MustParse("{{user.address.city}}")
	ctx := map[string]interface{}{
		"user": nilTestUser{Name: "Alice", Address: nil},
	}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestNilSafety_StructNilPointerFieldDeep(t *testing.T) {
	// Three levels deep: outer.inner.leaf.value where inner is nil
	tpl := MustParse("{{outer.inner.leaf.value}}")
	ctx := map[string]interface{}{
		"outer": nilTestOuter{Inner: nil},
	}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestNilSafety_IfBlockNilPath(t *testing.T) {
	// {{#if}} with nil path should evaluate to falsy, not panic
	tpl := MustParse("{{#if user.address.city}}has city{{else}}no city{{/if}}")
	ctx := map[string]interface{}{
		"user": nilTestUser{Name: "Alice", Address: nil},
	}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "no city" {
		t.Errorf("expected 'no city', got %q", result)
	}
}

func TestNilSafety_IfBlockNilMapValue(t *testing.T) {
	// {{#if}} on nil map value should be falsy
	tpl := MustParse("{{#if foo.bar}}yes{{else}}no{{/if}}")
	ctx := map[string]interface{}{"foo": nil}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "no" {
		t.Errorf("expected 'no', got %q", result)
	}
}

func TestNilSafety_WithBlockNilPath(t *testing.T) {
	// {{#with}} nil value should use else block
	tpl := MustParse("{{#with user.address}}{{city}}{{else}}no address{{/with}}")
	ctx := map[string]interface{}{
		"user": nilTestUser{Name: "Alice", Address: nil},
	}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "no address" {
		t.Errorf("expected 'no address', got %q", result)
	}
}

func TestNilSafety_EachNilSlice(t *testing.T) {
	// {{#each}} on nil slice should use else block
	tpl := MustParse("{{#each items}}{{name}}{{else}}none{{/each}}")
	ctx := map[string]interface{}{"items": nil}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "none" {
		t.Errorf("expected 'none', got %q", result)
	}
}

func TestNilSafety_EachNilSliceTyped(t *testing.T) {
	// {{#each}} on typed nil slice should use else block
	var items []nilTestUser
	tpl := MustParse("{{#each items}}{{name}}{{else}}none{{/each}}")
	ctx := map[string]interface{}{"items": items}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "none" {
		t.Errorf("expected 'none', got %q", result)
	}
}

func TestNilSafety_HelperParamNilPath(t *testing.T) {
	// Helper receiving nil path param should get empty string, not panic
	RegisterHelper("nilTestUpper", func(s string) string {
		return "got:" + s
	})
	defer RemoveHelper("nilTestUpper")

	tpl := MustParse("{{nilTestUpper foo.bar}}")
	ctx := map[string]interface{}{"foo": nil}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "got:" {
		t.Errorf("expected 'got:', got %q", result)
	}
}

func TestNilSafety_MissingIntermediateKey(t *testing.T) {
	// Accessing a.b.c where 'b' doesn't exist in the map at all
	tpl := MustParse("{{a.b.c}}")
	ctx := map[string]interface{}{
		"a": map[string]interface{}{"x": "y"},
	}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestNilSafety_TopLevelNilContext(t *testing.T) {
	// Rendering with nil context should not panic
	tpl := MustParse("hello {{name}}")
	result := mustRenderNoPanic(t, tpl, nil)
	if result != "hello " {
		t.Errorf("expected 'hello ', got %q", result)
	}
}

func TestNilSafety_TypedNilPointerInMap(t *testing.T) {
	// A typed nil pointer stored in interface{} map value
	tpl := MustParse("{{user.name}}")
	ctx := map[string]interface{}{
		"user": (*nilTestUser)(nil),
	}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestNilSafety_NilPointerToStruct_IfBlock(t *testing.T) {
	// Typed nil pointer in #if should be falsy
	tpl := MustParse("{{#if user}}yes{{else}}no{{/if}}")
	ctx := map[string]interface{}{
		"user": (*nilTestUser)(nil),
	}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "no" {
		t.Errorf("expected 'no', got %q", result)
	}
}

func TestNilSafety_NilInterfaceField(t *testing.T) {
	// Interface field that is nil
	tpl := MustParse("{{data.value}}")
	ctx := map[string]interface{}{
		"data": map[string]interface{}{
			"value": nil,
		},
	}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestNilSafety_SiblingAfterNil(t *testing.T) {
	// Accessing a valid sibling after a nil path should still work
	tpl := MustParse("{{user.address.city}}-{{user.name}}")
	ctx := map[string]interface{}{
		"user": nilTestUser{Name: "Alice", Address: nil},
	}
	result := mustRenderNoPanic(t, tpl, ctx)
	if result != "-Alice" {
		t.Errorf("expected '-Alice', got %q", result)
	}
}

// mustRenderNoPanic renders a template and fails the test if it panics
func mustRenderNoPanic(t *testing.T, tpl *Template, ctx interface{}) string {
	t.Helper()

	var result string
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("template rendering panicked: %v", r)
			}
		}()
		var err error
		result, err = tpl.Exec(ctx)
		if err != nil {
			t.Fatalf("template rendering error: %v", err)
		}
	}()

	return result
}
