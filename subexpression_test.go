package handlebars

import (
	"fmt"
	"strings"
	"testing"
)

// Tests for full subexpression support.
//
// Subexpressions like (helper arg) should work everywhere:
// as helper parameters, hash values, in block helpers (#if, #each, #with, #unless), etc.
//
// These tests complement the existing coverage in internal/handlebarsjs/subexpressions_test.go
// which tests via the Test struct/launchTests infrastructure. These are direct Go tests
// that verify the public API.

// Test 1: Subexpression as helper parameter: {{foo (bar baz)}}
func TestSubexpressionAsHelperParam(t *testing.T) {
	tpl := MustParse(`{{foo (bar baz)}}`)
	tpl.RegisterHelper("foo", func(val string) string {
		return "foo:" + val
	})
	tpl.RegisterHelper("bar", func(val string) string {
		return "bar:" + val
	})

	result := tpl.MustExec(map[string]string{"baz": "hello"})
	expected := "foo:bar:hello"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// Test 2: Subexpression in hash parameter: {{foo key=(bar baz)}}
func TestSubexpressionInHashParam(t *testing.T) {
	tpl := MustParse(`{{foo key=(bar baz)}}`)
	tpl.RegisterHelper("foo", func(options *Options) string {
		return "foo:" + options.HashStr("key")
	})
	tpl.RegisterHelper("bar", func(val string) string {
		return "bar:" + val
	})

	result := tpl.MustExec(map[string]string{"baz": "world"})
	expected := "foo:bar:world"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// Test 3: Nested subexpressions: {{foo (bar (baz qux))}}
func TestSubexpressionNested(t *testing.T) {
	tpl := MustParse(`{{foo (bar (baz qux))}}`)
	tpl.RegisterHelper("foo", func(val string) string {
		return "foo:" + val
	})
	tpl.RegisterHelper("bar", func(val string) string {
		return "bar:" + val
	})
	tpl.RegisterHelper("baz", func(val string) string {
		return "baz:" + val
	})

	result := tpl.MustExec(map[string]string{"qux": "deep"})
	expected := "foo:bar:baz:deep"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// Test 4: Subexpression in #if condition: {{#if (isEqual a b)}}yes{{/if}}
func TestSubexpressionInIfCondition(t *testing.T) {
	// Case where condition is true
	tpl := MustParse(`{{#if (isEqual a b)}}yes{{else}}no{{/if}}`)
	tpl.RegisterHelper("isEqual", func(a, b string) bool {
		return a == b
	})

	result := tpl.MustExec(map[string]string{"a": "same", "b": "same"})
	if result != "yes" {
		t.Errorf("Expected 'yes' when a==b, got %q", result)
	}

	// Case where condition is false
	tpl2 := MustParse(`{{#if (isEqual a b)}}yes{{else}}no{{/if}}`)
	tpl2.RegisterHelper("isEqual", func(a, b string) bool {
		return a == b
	})

	result2 := tpl2.MustExec(map[string]string{"a": "foo", "b": "bar"})
	if result2 != "no" {
		t.Errorf("Expected 'no' when a!=b, got %q", result2)
	}
}

// Test 5: Subexpression in #each: {{#each (filter items "active")}}{{name}}{{/each}}
func TestSubexpressionInEach(t *testing.T) {
	tpl := MustParse(`{{#each (filter items "active")}}{{name}} {{/each}}`)
	tpl.RegisterHelper("filter", func(items interface{}, status string) interface{} {
		if arr, ok := items.([]interface{}); ok {
			var result []interface{}
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					if m["status"] == status {
						result = append(result, m)
					}
				}
			}
			return result
		}
		return nil
	})

	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"name": "Alice", "status": "active"},
			map[string]interface{}{"name": "Bob", "status": "inactive"},
			map[string]interface{}{"name": "Carol", "status": "active"},
		},
	}

	result := tpl.MustExec(data)
	expected := "Alice Carol "
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// Test 6: Subexpression with gt helper for numeric comparison in #if
func TestSubexpressionGtInIf(t *testing.T) {
	tpl := MustParse(`{{#if (gt age 18)}}adult{{else}}minor{{/if}}`)
	tpl.RegisterHelper("gt", func(a, b int) bool {
		return a > b
	})

	result := tpl.MustExec(map[string]interface{}{"age": 21})
	if result != "adult" {
		t.Errorf("Expected 'adult', got %q", result)
	}

	tpl2 := MustParse(`{{#if (gt age 18)}}adult{{else}}minor{{/if}}`)
	tpl2.RegisterHelper("gt", func(a, b int) bool {
		return a > b
	})

	result2 := tpl2.MustExec(map[string]interface{}{"age": 15})
	if result2 != "minor" {
		t.Errorf("Expected 'minor', got %q", result2)
	}
}

// Test 7: Multiple subexpressions in hash values
func TestSubexpressionMultipleInHash(t *testing.T) {
	tpl := MustParse(`{{input aria-label=(t "Name") placeholder=(t "Example")}}`)
	tpl.RegisterHelper("input", func(options *Options) SafeString {
		return SafeString(fmt.Sprintf(`<input aria-label="%s" placeholder="%s" />`,
			options.HashStr("aria-label"),
			options.HashStr("placeholder")))
	})
	tpl.RegisterHelper("t", func(key string) SafeString {
		return SafeString(key)
	})

	result := tpl.MustExec(nil)
	expected := `<input aria-label="Name" placeholder="Example" />`
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// Test 8: Subexpression result used in #with
func TestSubexpressionInWith(t *testing.T) {
	tpl := MustParse(`{{#with (getUser users "alice")}}{{name}}{{/with}}`)
	tpl.RegisterHelper("getUser", func(users interface{}, id string) interface{} {
		if m, ok := users.(map[string]interface{}); ok {
			return m[id]
		}
		return nil
	})

	data := map[string]interface{}{
		"users": map[string]interface{}{
			"alice": map[string]interface{}{
				"name": "Alice",
			},
		},
	}

	result := tpl.MustExec(data)
	if result != "Alice" {
		t.Errorf("Expected 'Alice', got %q", result)
	}
}

// Test 9: Chained subexpressions as multiple params
func TestSubexpressionMultipleAsParams(t *testing.T) {
	tpl := MustParse(`{{concat (upper a) (upper b)}}`)
	tpl.RegisterHelper("concat", func(a, b string) string {
		return a + b
	})
	tpl.RegisterHelper("upper", func(val string) string {
		return strings.ToUpper(val)
	})

	result := tpl.MustExec(map[string]string{"a": "hello", "b": "world"})
	expected := "HELLOWORLD"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// Test 10: Subexpression in #unless
func TestSubexpressionInUnless(t *testing.T) {
	// Non-admin case
	tpl := MustParse(`{{#unless (isAdmin role)}}restricted{{else}}welcome{{/unless}}`)
	tpl.RegisterHelper("isAdmin", func(role string) bool {
		return role == "admin"
	})

	result := tpl.MustExec(map[string]string{"role": "user"})
	if result != "restricted" {
		t.Errorf("Expected 'restricted' for non-admin, got %q", result)
	}

	// Admin case
	tpl2 := MustParse(`{{#unless (isAdmin role)}}restricted{{else}}welcome{{/unless}}`)
	tpl2.RegisterHelper("isAdmin", func(role string) bool {
		return role == "admin"
	})

	result2 := tpl2.MustExec(map[string]string{"role": "admin"})
	if result2 != "welcome" {
		t.Errorf("Expected 'welcome' for admin, got %q", result2)
	}
}

// Test 11: Subexpression where helper returns a non-string (bool) used in #if
func TestSubexpressionReturnsNonString(t *testing.T) {
	tpl := MustParse(`{{#if (and a b)}}both{{else}}not both{{/if}}`)
	tpl.RegisterHelper("and", func(a, b bool) bool {
		return a && b
	})

	result := tpl.MustExec(map[string]interface{}{"a": true, "b": true})
	if result != "both" {
		t.Errorf("Expected 'both' when both true, got %q", result)
	}

	tpl2 := MustParse(`{{#if (and a b)}}both{{else}}not both{{/if}}`)
	tpl2.RegisterHelper("and", func(a, b bool) bool {
		return a && b
	})

	result2 := tpl2.MustExec(map[string]interface{}{"a": true, "b": false})
	if result2 != "not both" {
		t.Errorf("Expected 'not both' when b is false, got %q", result2)
	}
}

// Test 12: Subexpression returning int passed to another helper
func TestSubexpressionReturnsInt(t *testing.T) {
	tpl := MustParse(`{{formatNum (add a b)}}`)
	tpl.RegisterHelper("add", func(a, b int) int {
		return a + b
	})
	tpl.RegisterHelper("formatNum", func(n int) string {
		return "sum=" + Str(n)
	})

	result := tpl.MustExec(map[string]interface{}{"a": 3, "b": 4})
	expected := "sum=7"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// Test 13: Deeply nested subexpressions (4 levels)
func TestSubexpressionDeeplyNested(t *testing.T) {
	tpl := MustParse(`{{d (c (b (a val)))}}`)
	tpl.RegisterHelper("a", func(s string) string { return s + "A" })
	tpl.RegisterHelper("b", func(s string) string { return s + "B" })
	tpl.RegisterHelper("c", func(s string) string { return s + "C" })
	tpl.RegisterHelper("d", func(s string) string { return s + "D" })

	result := tpl.MustExec(map[string]string{"val": "X"})
	expected := "XABCD"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

// Test 14: Subexpression with mixed literal and context params
func TestSubexpressionMixedParams(t *testing.T) {
	tpl := MustParse(`{{join (wrap name "**") (wrap title "##")}}`)
	tpl.RegisterHelper("wrap", func(val, delim string) string {
		return delim + val + delim
	})
	tpl.RegisterHelper("join", func(a, b string) string {
		return a + " " + b
	})

	result := tpl.MustExec(map[string]string{"name": "Alice", "title": "Dr"})
	expected := "**Alice** ##Dr##"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}
