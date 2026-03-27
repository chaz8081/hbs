package handlebars

import (
	"strings"
	"testing"
)

func TestStrictMode_MissingVariableReturnsError(t *testing.T) {
	tpl := MustParse("Hello {{name}}")
	tpl.SetStrict(true)

	_, err := tpl.Exec(map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing variable in strict mode")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("expected error mentioning 'name', got: %s", err)
	}
}

func TestStrictMode_PresentVariableWorks(t *testing.T) {
	tpl := MustParse("Hello {{name}}")
	tpl.SetStrict(true)

	result, err := tpl.Exec(map[string]interface{}{"name": "World"})
	if err != nil {
		t.Fatal(err)
	}
	if result != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", result)
	}
}

func TestStrictMode_MissingNestedPath(t *testing.T) {
	tpl := MustParse("{{user.address.city}}")
	tpl.SetStrict(true)

	_, err := tpl.Exec(map[string]interface{}{
		"user": map[string]interface{}{},
	})
	if err == nil {
		t.Fatal("expected error for missing nested path in strict mode")
	}
}

func TestStrictMode_OffByDefault(t *testing.T) {
	tpl := MustParse("{{missing}}")
	result, err := tpl.Exec(map[string]interface{}{})
	if err != nil {
		t.Fatal("strict mode should be off by default")
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestStrictMode_HelperParamsGetEmptyForMissing(t *testing.T) {
	RegisterHelper("greet", func(name string) string { return "Hi " + name })
	defer RemoveHelper("greet")

	tpl := MustParse("{{greet name}}")
	tpl.SetStrict(true)

	// In strict mode, helper params resolve to empty string if missing
	// (the helper itself is found, so no error)
	result, err := tpl.Exec(map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if result != "Hi " {
		t.Errorf("expected 'Hi ', got %q", result)
	}
}

func TestStrictMode_ConditionalBranchNotChecked(t *testing.T) {
	// In strict mode, variables inside inactive branches should not cause errors
	tpl := MustParse("{{#if show}}{{name}}{{/if}}")
	tpl.SetStrict(true)

	result, err := tpl.Exec(map[string]interface{}{"show": false})
	if err != nil {
		t.Fatalf("should not error for inactive branch: %s", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}
