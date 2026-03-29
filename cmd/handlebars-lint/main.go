// handlebars-lint validates Handlebars template files for syntax errors
// and optionally checks that template variables exist in provided JSON data.
//
// Usage:
//
//	handlebars-lint [flags] <files or directories...>
//	handlebars-lint templates/**/*.hbs
//	handlebars-lint --data fixtures/data.json template.hbs
//	handlebars-lint --json templates/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	handlebars "github.com/chaz8081/hbs/v4"
)

type lintResult struct {
	File   string      `json:"file"`
	Errors []lintError `json:"errors"`
	Valid  bool        `json:"valid"`
}

type lintError struct {
	Type    string `json:"type"` // "parse" or "validation"
	Message string `json:"message"`
	Path    string `json:"path,omitempty"` // field path for validation errors
}

func main() {
	dataFile := flag.String("data", "", "JSON data file to validate template variables against")
	helpersFlag := flag.String("helpers", "", "Comma-separated list of known helper names")
	jsonOutput := flag.Bool("json", false, "Output results as JSON")
	quiet := flag.Bool("quiet", false, "Only show errors, no summary")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: handlebars-lint [flags] <files or directories...>\n\n")
		fmt.Fprintf(os.Stderr, "Validates Handlebars template files for syntax errors.\n")
		fmt.Fprintf(os.Stderr, "Optionally checks template variables against provided JSON data.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  handlebars-lint templates/**/*.hbs\n")
		fmt.Fprintf(os.Stderr, "  handlebars-lint --data data.json template.hbs\n")
		fmt.Fprintf(os.Stderr, "  handlebars-lint --json templates/\n")
	}
	flag.Parse()

	if flag.NArg() == 0 {
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

	// Load data file if provided
	var data interface{}
	if *dataFile != "" {
		b, err := ioutil.ReadFile(*dataFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading data file: %s\n", err)
			os.Exit(1)
		}
		if err := json.Unmarshal(b, &data); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing data JSON: %s\n", err)
			os.Exit(1)
		}
	}

	// Collect files
	files, err := collectFiles(flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error collecting files: %s\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No .hbs files found\n")
		os.Exit(1)
	}

	// Lint each file
	var results []lintResult
	totalErrors := 0

	for _, file := range files {
		result := lintFile(file, data, helpers)
		results = append(results, result)
		totalErrors += len(result.Errors)
	}

	// Output results
	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(results); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON output: %s\n", err)
			os.Exit(1)
		}
	} else {
		for _, r := range results {
			if len(r.Errors) == 0 {
				if !*quiet {
					fmt.Printf("✓ %s\n", r.File)
				}
				continue
			}
			for _, e := range r.Errors {
				fmt.Printf("%s: %s: %s\n", r.File, e.Type, e.Message)
			}
		}

		if !*quiet {
			fmt.Printf("\n%d file(s) checked, %d error(s)\n", len(files), totalErrors)
		}
	}

	if totalErrors > 0 {
		os.Exit(1)
	}
}

func lintFile(path string, data interface{}, helpers map[string]bool) lintResult {
	result := lintResult{File: path, Valid: true}

	source, err := ioutil.ReadFile(path)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, lintError{
			Type:    "io",
			Message: err.Error(),
		})
		return result
	}

	tpl, err := handlebars.Parse(string(source))
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, lintError{
			Type:    "parse",
			Message: err.Error(),
		})
		return result
	}

	// If data provided, validate template variables against it
	if data != nil {
		validationErrors := handlebars.Validate(tpl, data, helpers)
		for _, ve := range validationErrors {
			result.Valid = false
			result.Errors = append(result.Errors, lintError{
				Type:    "validation",
				Message: ve.Message,
				Path:    ve.Path,
			})
		}
	}

	return result
}

func collectFiles(args []string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	for _, arg := range args {
		// Check if it's a glob pattern
		matches, err := filepath.Glob(arg)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", arg, err)
		}

		targets := matches
		if len(targets) == 0 {
			// Not a glob, treat as literal path
			targets = []string{arg}
		}

		for _, target := range targets {
			info, err := os.Stat(target)
			if err != nil {
				return nil, fmt.Errorf("cannot access %q: %w", target, err)
			}

			if info.IsDir() {
				// Walk directory for .hbs files
				err := filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() && isTemplateFile(path) && !seen[path] {
						seen[path] = true
						files = append(files, path)
					}
					return nil
				})
				if err != nil {
					return nil, err
				}
			} else if isTemplateFile(target) && !seen[target] {
				seen[target] = true
				files = append(files, target)
			}
		}
	}

	return files, nil
}

func isTemplateFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".hbs" || ext == ".handlebars"
}
