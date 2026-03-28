package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

func TestProcessFile_BasicTemplate(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "test.hbs", "{{name}} {{email}}")

	schema, err := processFile(path, "TestData", nil)
	if err != nil {
		t.Fatal(err)
	}
	if schema.TypeName != "TestData" {
		t.Errorf("expected type name TestData, got %q", schema.TypeName)
	}
	if len(schema.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(schema.Fields))
	}
	if _, ok := schema.Fields["name"]; !ok {
		t.Error("expected 'name' field")
	}
	if _, ok := schema.Fields["email"]; !ok {
		t.Error("expected 'email' field")
	}
}

func TestProcessFile_EachInfersSlice(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "test.hbs", "{{#each items}}{{title}}{{/each}}")

	schema, err := processFile(path, "TestData", nil)
	if err != nil {
		t.Fatal(err)
	}

	fi, ok := schema.Fields["items"]
	if !ok {
		t.Fatal("expected 'items' field")
	}
	if fi.Type != typeSlice {
		t.Errorf("expected typeSlice, got %d", fi.Type)
	}
	if _, ok := fi.Children["title"]; !ok {
		t.Error("expected 'title' child field on items")
	}
}

func TestProcessFile_WithInfersStruct(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "test.hbs", "{{#with user}}{{name}}{{email}}{{/with}}")

	schema, err := processFile(path, "TestData", nil)
	if err != nil {
		t.Fatal(err)
	}

	fi, ok := schema.Fields["user"]
	if !ok {
		t.Fatal("expected 'user' field")
	}
	if fi.Type != typeStruct {
		t.Errorf("expected typeStruct, got %d", fi.Type)
	}
	if _, ok := fi.Children["name"]; !ok {
		t.Error("expected 'name' child field on user")
	}
}

func TestProcessFile_IfInfersInterface(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "test.hbs", "{{#if active}}yes{{/if}}")

	schema, err := processFile(path, "TestData", nil)
	if err != nil {
		t.Fatal(err)
	}

	fi, ok := schema.Fields["active"]
	if !ok {
		t.Fatal("expected 'active' field")
	}
	if fi.Type != typeInterface {
		t.Errorf("expected typeInterface, got %d", fi.Type)
	}
}

func TestProcessFile_NestedPaths(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "test.hbs", "{{user.name}}")

	schema, err := processFile(path, "TestData", nil)
	if err != nil {
		t.Fatal(err)
	}

	fi, ok := schema.Fields["user"]
	if !ok {
		t.Fatal("expected 'user' field")
	}
	if fi.Type != typeStruct {
		t.Errorf("expected typeStruct for nested path parent, got %d", fi.Type)
	}
	if _, ok := fi.Children["name"]; !ok {
		t.Error("expected 'name' child in user")
	}
}

func TestProcessFile_InvalidTemplate(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "bad.hbs", "{{#if}}")

	_, err := processFile(path, "TestData", nil)
	if err == nil {
		t.Fatal("expected error for invalid template")
	}
}

func TestProcessFile_MissingFile(t *testing.T) {
	_, err := processFile("/nonexistent/file.hbs", "TestData", nil)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestGenerateCode_Output(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "test.hbs", "{{name}} {{#each items}}{{title}}{{/each}}")

	schema, err := processFile(path, "TestData", nil)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	opts := genOptions{jsonTags: true}
	generateCode(&buf, "myapp", schema, opts)

	output := buf.String()

	if !strings.Contains(output, "package myapp") {
		t.Error("expected 'package myapp' in output")
	}
	if !strings.Contains(output, "type TestData struct") {
		t.Error("expected 'type TestData struct' in output")
	}
	if !strings.Contains(output, "DO NOT EDIT") {
		t.Error("expected 'DO NOT EDIT' header")
	}
	if !strings.Contains(output, `json:"name"`) {
		t.Error("expected json tag for name")
	}
}

func TestGenerateCode_WithValidation(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "test.hbs", "{{name}}")

	schema, err := processFile(path, "TestData", nil)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	opts := genOptions{jsonTags: true, validate: true}
	generateCode(&buf, "myapp", schema, opts)

	output := buf.String()

	if !strings.Contains(output, "func (d TestData) Validate() error") {
		t.Error("expected Validate method in output")
	}
	if !strings.Contains(output, "import") {
		t.Error("expected import block for validation")
	}
}

func TestGenerateCode_HandlebarsTags(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "test.hbs", "{{first_name}}")

	schema, err := processFile(path, "TestData", nil)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	opts := genOptions{jsonTags: true, handlebarsTags: true}
	generateCode(&buf, "myapp", schema, opts)

	output := buf.String()

	if !strings.Contains(output, `handlebars:"first_name"`) {
		t.Error("expected handlebars tag for snake_case field")
	}
}

func TestDeriveTypeName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"templates/user-profile.hbs", "UserProfileData"},
		{"email.hbs", "EmailData"},
		{"invoice_template.handlebars", "InvoiceTemplateData"},
		{"simple.hbs", "SimpleData"},
	}
	for _, tt := range tests {
		got := deriveTypeName(tt.path)
		if got != tt.want {
			t.Errorf("deriveTypeName(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestToGoName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"name", "Name"},
		{"first_name", "FirstName"},
		{"user-profile", "UserProfile"},
		{"showAvatar", "ShowAvatar"},
		{"email", "Email"},
	}
	for _, tt := range tests {
		got := toGoName(tt.in)
		if got != tt.want {
			t.Errorf("toGoName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestIsTemplateFile(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"template.hbs", true},
		{"template.handlebars", true},
		{"template.HBS", true},
		{"template.html", false},
		{"template.go", false},
	}
	for _, tt := range tests {
		if got := isTemplateFile(tt.name); got != tt.want {
			t.Errorf("isTemplateFile(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestBuildTags(t *testing.T) {
	f := &fieldInfo{Name: "FirstName", JSONName: "first_name"}

	// JSON only
	tag := buildTags(f, genOptions{jsonTags: true})
	if !strings.Contains(tag, `json:"first_name"`) {
		t.Errorf("expected json tag, got %q", tag)
	}

	// JSON + handlebars
	tag = buildTags(f, genOptions{jsonTags: true, handlebarsTags: true})
	if !strings.Contains(tag, `handlebars:"first_name"`) {
		t.Errorf("expected handlebars tag, got %q", tag)
	}

	// No tags
	tag = buildTags(f, genOptions{})
	if tag != "" {
		t.Errorf("expected empty tag, got %q", tag)
	}
}
