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

func TestCollectFiles_SingleFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.hbs", "{{hello}}")

	files, err := collectFiles([]string{filepath.Join(dir, "a.hbs")})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
}

func TestCollectFiles_Directory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.hbs", "{{a}}")
	writeFile(t, dir, "b.handlebars", "{{b}}")
	writeFile(t, dir, "c.txt", "not a template")

	files, err := collectFiles([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 .hbs/.handlebars files, got %d", len(files))
	}
}

func TestCollectFiles_Dedup(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "a.hbs", "{{a}}")

	files, err := collectFiles([]string{path, path})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected dedup to 1 file, got %d", len(files))
	}
}

func TestLintFile_ValidTemplate(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "ok.hbs", "<p>{{name}}</p>")

	result := lintFile(path, nil, nil)
	if !result.Valid {
		t.Fatalf("expected valid, got errors: %v", result.Errors)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(result.Errors))
	}
}

func TestLintFile_InvalidSyntax(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "bad.hbs", "{{#if}}")

	result := lintFile(path, nil, nil)
	if result.Valid {
		t.Fatal("expected invalid for bad syntax")
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected at least one error")
	}
	if result.Errors[0].Type != "parse" {
		t.Fatalf("expected parse error, got %q", result.Errors[0].Type)
	}
}

func TestLintFile_WithDataValidation(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "check.hbs", "{{name}} {{missing_field}}")

	data := map[string]interface{}{
		"name": "Alice",
	}

	result := lintFile(path, data, nil)

	// Should find validation errors for missing_field
	hasValidation := false
	for _, e := range result.Errors {
		if e.Type == "validation" {
			hasValidation = true
		}
	}
	if !hasValidation {
		t.Fatal("expected validation error for missing field")
	}
}

func TestLintFile_MissingFile(t *testing.T) {
	result := lintFile("/nonexistent/file.hbs", nil, nil)
	if result.Valid {
		t.Fatal("expected invalid for missing file")
	}
	if result.Errors[0].Type != "io" {
		t.Fatalf("expected io error, got %q", result.Errors[0].Type)
	}
}

func TestIsTemplateFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"template.hbs", true},
		{"template.handlebars", true},
		{"template.HBS", true},
		{"template.html", false},
		{"template.txt", false},
		{"template.go", false},
	}
	for _, tt := range tests {
		if got := isTemplateFile(tt.path); got != tt.want {
			t.Errorf("isTemplateFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestLintResult_JSONMarshal(t *testing.T) {
	r := lintResult{
		File:  "test.hbs",
		Valid: false,
		Errors: []lintError{
			{Type: "parse", Message: "bad syntax"},
		},
	}

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}

	var decoded lintResult
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.File != "test.hbs" {
		t.Errorf("expected file test.hbs, got %q", decoded.File)
	}
	if decoded.Valid {
		t.Error("expected valid=false")
	}
	if len(decoded.Errors) != 1 || decoded.Errors[0].Type != "parse" {
		t.Errorf("unexpected errors: %v", decoded.Errors)
	}
}
