// handlebars-gen generates Go struct definitions from Handlebars template files.
// It analyzes template variable usage to infer types:
//
//   - {{name}} → string field
//   - {{#if active}} → interface{} field (boolean condition)
//   - {{#each items}} → slice field with element struct
//   - {{#with user}} → nested struct field
//   - {{user.name}} → nested struct with string field
//
// Usage:
//
//	handlebars-gen --package myapp template.hbs
//	handlebars-gen --package myapp --type EmailData templates/email.hbs
//	handlebars-gen --package myapp --dir templates/ --output types_gen.go
//	handlebars-gen --package myapp --validate templates/email.hbs
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	handlebars "github.com/chaz8081/handlebars-go/v4"
)

func main() {
	pkg := flag.String("package", "", "Go package name (required)")
	typeName := flag.String("type", "", "Go struct type name (default: derived from filename)")
	output := flag.String("output", "", "Output file (default: stdout)")
	dir := flag.String("dir", "", "Process all .hbs files in directory")
	helpersFlag := flag.String("helpers", "", "Comma-separated list of known helper names")
	validate := flag.Bool("validate", false, "Generate a Validate() method")
	jsonTags := flag.Bool("json-tags", true, "Include json struct tags")
	handlebarsTags := flag.Bool("handlebars-tags", false, "Include handlebars struct tags")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: handlebars-gen [flags] <files...>\n\n")
		fmt.Fprintf(os.Stderr, "Generates Go struct definitions from Handlebars template files.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  handlebars-gen --package myapp template.hbs\n")
		fmt.Fprintf(os.Stderr, "  handlebars-gen --package myapp --type EmailData email.hbs\n")
		fmt.Fprintf(os.Stderr, "  handlebars-gen --package myapp --dir templates/ --output types_gen.go\n")
		fmt.Fprintf(os.Stderr, "  handlebars-gen --package myapp --validate templates/email.hbs\n")
	}
	flag.Parse()

	if *pkg == "" {
		fmt.Fprintf(os.Stderr, "Error: --package is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Collect files
	var files []string
	if *dir != "" {
		entries, err := ioutil.ReadDir(*dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading directory: %s\n", err)
			os.Exit(1)
		}
		for _, e := range entries {
			if !e.IsDir() && isTemplateFile(e.Name()) {
				files = append(files, filepath.Join(*dir, e.Name()))
			}
		}
	}
	files = append(files, flag.Args()...)

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no template files specified\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Parse known helpers
	var helpers map[string]bool
	if *helpersFlag != "" {
		helpers = make(map[string]bool)
		for _, h := range strings.Split(*helpersFlag, ",") {
			h = strings.TrimSpace(h)
			if h != "" {
				helpers[h] = true
			}
		}
	}

	opts := genOptions{
		jsonTags:       *jsonTags,
		handlebarsTags: *handlebarsTags,
		validate:       *validate,
	}

	// Determine output writer
	var w *os.File
	if *output != "" {
		var err error
		w, err = os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %s\n", err)
			os.Exit(1)
		}
		defer w.Close()
	} else {
		w = os.Stdout
	}

	// Process each file
	for i, file := range files {
		name := *typeName
		if name == "" {
			name = deriveTypeName(file)
		} else if len(files) > 1 {
			// Multiple files with explicit type name — append file suffix
			name = deriveTypeName(file)
		}

		schema, err := processFile(file, name, helpers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %s\n", file, err)
			os.Exit(1)
		}

		if i == 0 {
			// Write package header only once
			generateCode(w, *pkg, schema, opts)
		} else {
			// Additional schemas: just write structs (no package header)
			writeStruct(w, schema.TypeName, schema.Fields, opts, 0)
			if opts.validate {
				writeValidateFunc(w, schema)
			}
		}
	}

	if *output != "" {
		fmt.Fprintf(os.Stderr, "Generated %s\n", *output)
	}
}

func processFile(path string, typeName string, helpers map[string]bool) (*templateSchema, error) {
	source, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	tpl, err := handlebars.Parse(string(source))
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	program := tpl.AST()
	if program == nil {
		return nil, fmt.Errorf("empty template")
	}

	return inferSchema(program, typeName, helpers), nil
}

func isTemplateFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".hbs" || ext == ".handlebars"
}

// deriveTypeName generates a Go type name from a file path.
// e.g., "templates/user-profile.hbs" -> "UserProfileData"
func deriveTypeName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Convert to PascalCase
	goName := toGoName(name)
	if goName == "" {
		goName = "Template"
	}

	return goName + "Data"
}
