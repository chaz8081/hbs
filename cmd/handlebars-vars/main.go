// handlebars-vars extracts all variables referenced in Handlebars template files
// and outputs them as structured JSON.
//
// Usage:
//
//	handlebars-vars [flags] <files...>
//	handlebars-vars template.hbs
//	handlebars-vars --required-only template.hbs
//	handlebars-vars --helpers "formatDate,currency" template.hbs
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	handlebars "github.com/chaz8081/hbs/v4"
)

type fileVars struct {
	File      string    `json:"file"`
	Variables []varInfo `json:"variables"`
}

type varInfo struct {
	Path       string   `json:"path"`
	Required   bool     `json:"required"`
	Conditions []string `json:"conditions,omitempty"`
	Line       int      `json:"line"`
	Position   int      `json:"position"`
	Source     string   `json:"source,omitempty"`
}

func main() {
	helpersFlag := flag.String("helpers", "", "Comma-separated list of known helper names")
	requiredOnly := flag.Bool("required-only", false, "Only output required (unconditional) variables")
	compact := flag.Bool("compact", false, "Output compact JSON (no indentation)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: handlebars-vars [flags] <files...>\n\n")
		fmt.Fprintf(os.Stderr, "Extracts all variables referenced in Handlebars template files.\n")
		fmt.Fprintf(os.Stderr, "Outputs structured JSON to stdout.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  handlebars-vars template.hbs\n")
		fmt.Fprintf(os.Stderr, "  handlebars-vars --required-only template.hbs\n")
		fmt.Fprintf(os.Stderr, "  handlebars-vars --helpers \"formatDate,currency\" template.hbs\n")
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

	var results []fileVars
	hasErrors := false

	for _, file := range flag.Args() {
		fv, err := extractFileVars(file, helpers, *requiredOnly)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error processing %s: %s\n", file, err)
			hasErrors = true
			continue
		}
		results = append(results, fv)
	}

	// Output
	enc := json.NewEncoder(os.Stdout)
	if !*compact {
		enc.SetIndent("", "  ")
	}

	if len(results) == 1 {
		if err := enc.Encode(results[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON output: %s\n", err)
			os.Exit(1)
		}
	} else {
		if err := enc.Encode(results); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON output: %s\n", err)
			os.Exit(1)
		}
	}

	if hasErrors {
		os.Exit(1)
	}
}

func extractFileVars(path string, helpers map[string]bool, requiredOnly bool) (fileVars, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return fileVars{}, err
	}

	tpl, err := handlebars.Parse(string(source))
	if err != nil {
		return fileVars{}, fmt.Errorf("parse error: %w", err)
	}

	vars := handlebars.ExtractVariables(tpl, helpers)

	result := fileVars{File: path}
	for _, v := range vars {
		if requiredOnly && !v.Required {
			continue
		}
		result.Variables = append(result.Variables, varInfo{
			Path:       v.Path,
			Required:   v.Required,
			Conditions: v.Conditions,
			Line:       v.Location.Line,
			Position:   v.Location.Pos,
			Source:     v.Source,
		})
	}

	return result, nil
}
