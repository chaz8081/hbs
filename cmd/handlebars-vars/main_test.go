package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractFileVars_BasicVariables(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "test.hbs", "{{name}} {{email}}")

	fv, err := extractFileVars(path, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if fv.File != path {
		t.Errorf("expected file %q, got %q", path, fv.File)
	}
	if len(fv.Variables) < 2 {
		t.Fatalf("expected at least 2 variables, got %d", len(fv.Variables))
	}

	paths := make(map[string]bool)
	for _, v := range fv.Variables {
		paths[v.Path] = true
	}
	if !paths["name"] {
		t.Error("expected 'name' variable")
	}
	if !paths["email"] {
		t.Error("expected 'email' variable")
	}
}

func TestExtractFileVars_RequiredOnly(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "test.hbs", "{{name}} {{#if active}}{{email}}{{/if}}")

	// All variables
	all, err := extractFileVars(path, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	// Required only
	required, err := extractFileVars(path, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	if len(required.Variables) >= len(all.Variables) {
		t.Errorf("expected fewer required variables (%d) than total (%d)",
			len(required.Variables), len(all.Variables))
	}

	for _, v := range required.Variables {
		if !v.Required {
			t.Errorf("expected only required variables, got non-required %q", v.Path)
		}
	}
}

func TestExtractFileVars_NestedPaths(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "test.hbs", "{{user.name}} {{user.email}}")

	fv, err := extractFileVars(path, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, v := range fv.Variables {
		if v.Path == "user.name" || v.Path == "user.email" {
			found = true
		}
	}
	if !found {
		t.Error("expected nested path variables like user.name or user.email")
	}
}

func TestExtractFileVars_WithHelpers(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "test.hbs", "{{formatDate date}} {{name}}")

	helpers := map[string]bool{"formatDate": true}
	fv, err := extractFileVars(path, helpers, false)
	if err != nil {
		t.Fatal(err)
	}

	// formatDate should be recognized as a helper, not a variable
	for _, v := range fv.Variables {
		if v.Path == "formatDate" {
			t.Error("helper 'formatDate' should not appear as a variable")
		}
	}
}

func TestExtractFileVars_InvalidTemplate(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "bad.hbs", "{{#if}}")

	_, err := extractFileVars(path, nil, false)
	if err == nil {
		t.Fatal("expected error for invalid template")
	}
}

func TestExtractFileVars_MissingFile(t *testing.T) {
	_, err := extractFileVars("/nonexistent/file.hbs", nil, false)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestVarInfo_JSONMarshal(t *testing.T) {
	vi := varInfo{
		Path:       "user.name",
		Required:   true,
		Conditions: []string{"active"},
		Line:       5,
		Position:   10,
		Source:      "mustache",
	}

	b, err := json.Marshal(vi)
	if err != nil {
		t.Fatal(err)
	}

	var decoded varInfo
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Path != "user.name" || !decoded.Required || decoded.Line != 5 {
		t.Errorf("unexpected decoded value: %+v", decoded)
	}
}
