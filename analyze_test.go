package handlebars

import (
	"sort"
	"testing"
)

func TestExtractVariables_SimpleVariable(t *testing.T) {
	tpl := MustParse("Hello {{name}}")
	vars := ExtractVariables(tpl, nil)

	assertHasVar(t, vars, "name", true, nil)
}

func TestExtractVariables_MultipleVariables(t *testing.T) {
	tpl := MustParse("{{firstName}} {{lastName}}")
	vars := ExtractVariables(tpl, nil)

	assertHasVar(t, vars, "firstName", true, nil)
	assertHasVar(t, vars, "lastName", true, nil)
}

func TestExtractVariables_DottedPath(t *testing.T) {
	tpl := MustParse("{{user.address.city}}")
	vars := ExtractVariables(tpl, nil)

	assertHasVar(t, vars, "user.address.city", true, nil)
}

func TestExtractVariables_IfBlockConditional(t *testing.T) {
	tpl := MustParse("{{#if premium}}{{billing.plan}}{{/if}}")
	vars := ExtractVariables(tpl, nil)

	// premium is the condition variable — it's required (needs to be checked)
	assertHasVar(t, vars, "premium", true, nil)
	// billing.plan is conditional on premium
	assertHasVar(t, vars, "billing.plan", false, []string{"premium"})
}

func TestExtractVariables_IfElseBlock(t *testing.T) {
	tpl := MustParse("{{#if active}}{{name}}{{else}}{{fallbackName}}{{/if}}")
	vars := ExtractVariables(tpl, nil)

	assertHasVar(t, vars, "active", true, nil)
	assertHasVar(t, vars, "name", false, []string{"active"})
	assertHasVar(t, vars, "fallbackName", false, []string{"active"})
}

func TestExtractVariables_NestedIf(t *testing.T) {
	tpl := MustParse("{{#if a}}{{#if b}}{{deep}}{{/if}}{{/if}}")
	vars := ExtractVariables(tpl, nil)

	assertHasVar(t, vars, "a", true, nil)
	assertHasVar(t, vars, "b", false, []string{"a"})
	assertHasVar(t, vars, "deep", false, []string{"a", "b"})
}

func TestExtractVariables_UnlessBlock(t *testing.T) {
	tpl := MustParse("{{#unless hidden}}{{content}}{{/unless}}")
	vars := ExtractVariables(tpl, nil)

	assertHasVar(t, vars, "hidden", true, nil)
	assertHasVar(t, vars, "content", false, []string{"hidden"})
}

func TestExtractVariables_EachBlock(t *testing.T) {
	tpl := MustParse("{{#each items}}{{name}}{{/each}}")
	vars := ExtractVariables(tpl, nil)

	// items is required — it's the iteration target
	assertHasVar(t, vars, "items", true, nil)
	// name inside each is contextual — we can't resolve it against outer context
	// but we still record it
}

func TestExtractVariables_WithBlock(t *testing.T) {
	tpl := MustParse("{{#with user}}{{name}}{{/with}}")
	vars := ExtractVariables(tpl, nil)

	assertHasVar(t, vars, "user", true, nil)
}

func TestExtractVariables_HelperWithParams(t *testing.T) {
	// When a helper is called with params, the params are variables, not the helper name
	helpers := map[string]bool{"formatDate": true}
	tpl := MustParse("{{formatDate createdAt}}")
	vars := ExtractVariables(tpl, helpers)

	assertNoVar(t, vars, "formatDate") // helper, not a variable
	assertHasVar(t, vars, "createdAt", true, nil)
}

func TestExtractVariables_HelperWithMultipleParams(t *testing.T) {
	helpers := map[string]bool{"concat": true}
	tpl := MustParse("{{concat firstName lastName}}")
	vars := ExtractVariables(tpl, helpers)

	assertNoVar(t, vars, "concat")
	assertHasVar(t, vars, "firstName", true, nil)
	assertHasVar(t, vars, "lastName", true, nil)
}

func TestExtractVariables_HelperWithHashParams(t *testing.T) {
	helpers := map[string]bool{"input": true}
	tpl := MustParse("{{input value=userName placeholder=defaultText}}")
	vars := ExtractVariables(tpl, helpers)

	assertNoVar(t, vars, "input")
	assertHasVar(t, vars, "userName", true, nil)
	assertHasVar(t, vars, "defaultText", true, nil)
}

func TestExtractVariables_AmbiguousWithoutHelper(t *testing.T) {
	// Without a helper registry, {{foo}} is treated as a variable
	tpl := MustParse("{{foo}}")
	vars := ExtractVariables(tpl, nil)

	assertHasVar(t, vars, "foo", true, nil)
}

func TestExtractVariables_AmbiguousWithHelper(t *testing.T) {
	// With a helper registry, {{foo}} is recognized as a helper
	helpers := map[string]bool{"foo": true}
	tpl := MustParse("{{foo}}")
	vars := ExtractVariables(tpl, helpers)

	assertNoVar(t, vars, "foo")
}

func TestExtractVariables_UnambiguousHelper(t *testing.T) {
	// {{foo bar}} is unambiguously a helper call — foo has params
	// Even without a helper registry, foo should not be a variable
	tpl := MustParse("{{foo bar}}")
	vars := ExtractVariables(tpl, nil)

	assertNoVar(t, vars, "foo") // foo is a helper (has params)
	assertHasVar(t, vars, "bar", true, nil)
}

func TestExtractVariables_IgnoresDataVariables(t *testing.T) {
	// @index, @key, @first, @last are built-in data variables
	tpl := MustParse("{{#each items}}{{@index}}:{{name}}{{/each}}")
	vars := ExtractVariables(tpl, nil)

	assertHasVar(t, vars, "items", true, nil)
	assertNoVar(t, vars, "@index")
}

func TestExtractVariables_IgnoresLiterals(t *testing.T) {
	helpers := map[string]bool{"helper": true}
	tpl := MustParse(`{{helper "literal" 42 true}}`)
	vars := ExtractVariables(tpl, helpers)

	if len(vars) != 0 {
		t.Errorf("expected no variables, got %v", vars)
	}
}

func TestExtractVariables_PartialParams(t *testing.T) {
	tpl := MustParse("{{> myPartial user}}")
	vars := ExtractVariables(tpl, nil)

	assertHasVar(t, vars, "user", true, nil)
}

func TestExtractVariables_PartialHashParams(t *testing.T) {
	tpl := MustParse("{{> myPartial name=hero}}")
	vars := ExtractVariables(tpl, nil)

	assertHasVar(t, vars, "hero", true, nil)
}

func TestExtractVariables_Deduplication(t *testing.T) {
	// Same variable used twice should only appear once
	tpl := MustParse("{{name}} and {{name}}")
	vars := ExtractVariables(tpl, nil)

	count := 0
	for _, v := range vars {
		if v.Path == "name" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 entry for 'name', got %d", count)
	}
}

func TestExtractVariables_DedupPreservesRequired(t *testing.T) {
	// If a variable appears both required and conditional, it should be marked required
	tpl := MustParse("{{name}}{{#if active}}{{name}}{{/if}}")
	vars := ExtractVariables(tpl, nil)

	v := findVar(vars, "name")
	if v == nil {
		t.Fatal("expected 'name' variable")
	}
	if !v.Required {
		t.Error("'name' should be required since it appears unconditionally")
	}
}

func TestExtractVariables_ComplexTemplate(t *testing.T) {
	source := `
Hello {{firstName}} {{lastName}},

{{#if premium}}
  Your plan: {{billing.plan}}
  {{#if billing.address}}
    City: {{billing.address.city}}
  {{/if}}
{{/if}}

{{#each orders}}
  Order: {{orderId}}
{{/each}}
`
	helpers := map[string]bool{}
	tpl := MustParse(source)
	vars := ExtractVariables(tpl, helpers)

	assertHasVar(t, vars, "firstName", true, nil)
	assertHasVar(t, vars, "lastName", true, nil)
	assertHasVar(t, vars, "premium", true, nil)
	assertHasVar(t, vars, "billing.plan", false, []string{"premium"})
	assertHasVar(t, vars, "billing.address", false, []string{"premium"})
	assertHasVar(t, vars, "billing.address.city", false, []string{"premium", "billing.address"})
	assertHasVar(t, vars, "orders", true, nil)
}

func TestExtractVariables_Location(t *testing.T) {
	tpl := MustParse("{{name}}")
	vars := ExtractVariables(tpl, nil)

	if len(vars) == 0 {
		t.Fatal("expected at least one variable")
	}
	if vars[0].Location.Line == 0 && vars[0].Location.Pos == 0 {
		t.Error("expected non-zero location")
	}
}

func TestExtractVariables_BuiltInHelpers(t *testing.T) {
	// Built-in helpers (if, each, unless, with) should not appear as variables
	tpl := MustParse("{{#if x}}{{#each y}}{{#unless z}}{{#with w}}{{/with}}{{/unless}}{{/each}}{{/if}}")
	vars := ExtractVariables(tpl, nil)

	assertNoVar(t, vars, "if")
	assertNoVar(t, vars, "each")
	assertNoVar(t, vars, "unless")
	assertNoVar(t, vars, "with")
}

// --- test helpers ---

func findVar(vars []TemplateVar, path string) *TemplateVar {
	for i := range vars {
		if vars[i].Path == path {
			return &vars[i]
		}
	}
	return nil
}

func assertHasVar(t *testing.T, vars []TemplateVar, path string, required bool, conditions []string) {
	t.Helper()

	v := findVar(vars, path)
	if v == nil {
		paths := make([]string, len(vars))
		for i, vv := range vars {
			paths[i] = vv.Path
		}
		t.Errorf("variable %q not found in %v", path, paths)
		return
	}

	if v.Required != required {
		t.Errorf("variable %q: expected Required=%v, got %v", path, required, v.Required)
	}

	if conditions == nil {
		if len(v.Conditions) != 0 {
			t.Errorf("variable %q: expected no conditions, got %v", path, v.Conditions)
		}
	} else {
		sort.Strings(conditions)
		actual := make([]string, len(v.Conditions))
		copy(actual, v.Conditions)
		sort.Strings(actual)

		if len(actual) != len(conditions) {
			t.Errorf("variable %q: expected conditions %v, got %v", path, conditions, actual)
			return
		}
		for i := range conditions {
			if conditions[i] != actual[i] {
				t.Errorf("variable %q: expected conditions %v, got %v", path, conditions, actual)
				return
			}
		}
	}
}

func assertNoVar(t *testing.T, vars []TemplateVar, path string) {
	t.Helper()
	if findVar(vars, path) != nil {
		t.Errorf("variable %q should not be in results", path)
	}
}
