package handlebars

import (
	"testing"
)

func TestValidate_AllPresent(t *testing.T) {
	tpl := MustParse("Hello {{name}}")
	errs := Validate(tpl, map[string]interface{}{"name": "World"}, nil)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidate_MissingRequired(t *testing.T) {
	tpl := MustParse("{{firstName}} {{lastName}}")
	errs := Validate(tpl, map[string]interface{}{"firstName": "Alice"}, nil)
	assertValidationHasError(t, errs, "lastName")
	assertValidationNoError(t, errs, "firstName")
}

func TestValidate_MissingMultiple(t *testing.T) {
	tpl := MustParse("{{a}} {{b}} {{c}}")
	errs := Validate(tpl, map[string]interface{}{"b": "present"}, nil)
	assertValidationHasError(t, errs, "a")
	assertValidationHasError(t, errs, "c")
	assertValidationNoError(t, errs, "b")
}

func TestValidate_ConditionFalsy_SkipsConditional(t *testing.T) {
	tpl := MustParse("{{#if premium}}{{billing.plan}}{{/if}}")
	errs := Validate(tpl, map[string]interface{}{"premium": false}, nil)
	if len(errs) != 0 {
		t.Errorf("expected no errors (conditional branch inactive), got %v", errs)
	}
}

func TestValidate_ConditionTruthy_ChecksConditional(t *testing.T) {
	tpl := MustParse("{{#if premium}}{{billing.plan}}{{/if}}")
	errs := Validate(tpl, map[string]interface{}{"premium": true}, nil)
	assertValidationHasError(t, errs, "billing.plan")
}

func TestValidate_ConditionMissing(t *testing.T) {
	tpl := MustParse("{{#if premium}}{{billing.plan}}{{/if}}")
	errs := Validate(tpl, map[string]interface{}{}, nil)
	assertValidationHasError(t, errs, "premium")
}

func TestValidate_ElseBranch(t *testing.T) {
	tpl := MustParse("{{#if active}}{{name}}{{else}}{{fallbackName}}{{/if}}")
	errs := Validate(tpl, map[string]interface{}{"active": false}, nil)
	assertValidationHasError(t, errs, "fallbackName")
	assertValidationNoError(t, errs, "name")
}

func TestValidate_ElseBranch_Truthy(t *testing.T) {
	tpl := MustParse("{{#if active}}{{name}}{{else}}{{fallbackName}}{{/if}}")
	errs := Validate(tpl, map[string]interface{}{"active": true}, nil)
	assertValidationHasError(t, errs, "name")
	assertValidationNoError(t, errs, "fallbackName")
}

func TestValidate_NestedConditions(t *testing.T) {
	tpl := MustParse("{{#if a}}{{#if b}}{{deep}}{{/if}}{{/if}}")
	errs := Validate(tpl, map[string]interface{}{"a": true, "b": true}, nil)
	assertValidationHasError(t, errs, "deep")
}

func TestValidate_NestedConditions_OuterFalsy(t *testing.T) {
	tpl := MustParse("{{#if a}}{{#if b}}{{deep}}{{/if}}{{/if}}")
	errs := Validate(tpl, map[string]interface{}{"a": false}, nil)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidate_UnlessBlock(t *testing.T) {
	tpl := MustParse("{{#unless hidden}}{{content}}{{/unless}}")
	errs := Validate(tpl, map[string]interface{}{"hidden": false}, nil)
	assertValidationHasError(t, errs, "content")
}

func TestValidate_UnlessBlock_Truthy(t *testing.T) {
	tpl := MustParse("{{#unless hidden}}{{content}}{{/unless}}")
	errs := Validate(tpl, map[string]interface{}{"hidden": true}, nil)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidate_EachBlock_Present(t *testing.T) {
	tpl := MustParse("{{#each items}}{{name}}{{/each}}")
	errs := Validate(tpl, map[string]interface{}{
		"items": []map[string]interface{}{{"name": "a"}},
	}, nil)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidate_EachBlock_Missing(t *testing.T) {
	tpl := MustParse("{{#each items}}{{name}}{{/each}}")
	errs := Validate(tpl, map[string]interface{}{}, nil)
	assertValidationHasError(t, errs, "items")
}

func TestValidate_WithBlock_Present(t *testing.T) {
	tpl := MustParse("{{#with user}}{{name}}{{/with}}")
	errs := Validate(tpl, map[string]interface{}{
		"user": map[string]interface{}{"name": "Alice"},
	}, nil)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidate_WithBlock_Missing(t *testing.T) {
	tpl := MustParse("{{#with user}}{{name}}{{/with}}")
	errs := Validate(tpl, map[string]interface{}{}, nil)
	assertValidationHasError(t, errs, "user")
}

func TestValidate_DottedPath_Present(t *testing.T) {
	tpl := MustParse("{{user.address.city}}")
	errs := Validate(tpl, map[string]interface{}{
		"user": map[string]interface{}{
			"address": map[string]interface{}{"city": "NYC"},
		},
	}, nil)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidate_DottedPath_Missing(t *testing.T) {
	tpl := MustParse("{{user.address.city}}")
	errs := Validate(tpl, map[string]interface{}{
		"user": map[string]interface{}{},
	}, nil)
	assertValidationHasError(t, errs, "user.address.city")
}

func TestValidate_HelperParams(t *testing.T) {
	helpers := map[string]bool{"formatDate": true}
	tpl := MustParse("{{formatDate createdAt}}")
	errs := Validate(tpl, map[string]interface{}{}, helpers)
	assertValidationHasError(t, errs, "createdAt")
	assertValidationNoError(t, errs, "formatDate")
}

func TestValidate_EmptyStringIsFalsy(t *testing.T) {
	tpl := MustParse("{{#if name}}{{greeting}}{{/if}}")
	errs := Validate(tpl, map[string]interface{}{"name": ""}, nil)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidate_ZeroIsFalsy(t *testing.T) {
	tpl := MustParse("{{#if count}}{{detail}}{{/if}}")
	errs := Validate(tpl, map[string]interface{}{"count": 0}, nil)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidate_NilIsFalsy(t *testing.T) {
	tpl := MustParse("{{#if user}}{{user.name}}{{/if}}")
	errs := Validate(tpl, map[string]interface{}{"user": nil}, nil)
	if len(errs) != 0 {
		t.Errorf("expected no errors (nil is falsy), got %v", errs)
	}
}

func TestValidate_ComplexTemplate(t *testing.T) {
	source := "Hello {{firstName}} {{lastName}},\n{{#if premium}}\n  Plan: {{billing.plan}}\n{{/if}}\n{{#each orders}}\n  Order: {{orderId}}\n{{/each}}"
	tpl := MustParse(source)
	errs := Validate(tpl, map[string]interface{}{
		"firstName": "Alice",
		"premium":   true,
		"orders":    []interface{}{},
	}, nil)
	assertValidationHasError(t, errs, "lastName")
	assertValidationHasError(t, errs, "billing.plan")
	assertValidationNoError(t, errs, "firstName")
	assertValidationNoError(t, errs, "orders")
}

func TestValidate_CollectsAllErrors(t *testing.T) {
	tpl := MustParse("{{a}} {{b}} {{c}} {{d}}")
	errs := Validate(tpl, map[string]interface{}{}, nil)
	if len(errs) != 4 {
		t.Errorf("expected 4 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidate_ValidationErrorFields(t *testing.T) {
	tpl := MustParse("{{name}}")
	errs := Validate(tpl, map[string]interface{}{}, nil)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Path != "name" {
		t.Errorf("expected path 'name', got %q", errs[0].Path)
	}
	if errs[0].Message == "" {
		t.Error("expected non-empty message")
	}
}

func assertValidationHasError(t *testing.T, errs []ValidationError, path string) {
	t.Helper()
	for _, e := range errs {
		if e.Path == path {
			return
		}
	}
	paths := make([]string, len(errs))
	for i, e := range errs {
		paths[i] = e.Path
	}
	t.Errorf("expected error for %q, not found in %v", path, paths)
}

func assertValidationNoError(t *testing.T, errs []ValidationError, path string) {
	t.Helper()
	for _, e := range errs {
		if e.Path == path {
			t.Errorf("unexpected error for %q", path)
			return
		}
	}
}
