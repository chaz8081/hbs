package main

import (
	"sort"
	"strings"

	"github.com/chaz8081/handlebars-go/v4/ast"
)

// fieldType represents an inferred Go type for a template variable.
type fieldType int

const (
	typeString    fieldType = iota // default: rendered as text
	typeInterface                  // used as condition (#if / #unless)
	typeSlice                      // used with #each
	typeStruct                     // used with #with or has nested paths
)

// fieldInfo holds inferred info about a single field.
type fieldInfo struct {
	Name     string               // Go-friendly field name (e.g., "Name")
	JSONName string               // original template name (e.g., "name")
	Type     fieldType            // inferred type
	Required bool                 // true if not inside conditional
	Children map[string]*fieldInfo // nested fields (for struct/slice element types)
}

// templateSchema is the top-level inferred schema for a template.
type templateSchema struct {
	TypeName string               // Go struct type name
	Fields   map[string]*fieldInfo
}

// inferSchema walks a parsed template's AST and infers the data schema.
func inferSchema(program *ast.Program, typeName string, helpers map[string]bool) *templateSchema {
	s := &templateSchema{
		TypeName: typeName,
		Fields:   make(map[string]*fieldInfo),
	}

	w := &schemaWalker{
		schema:  s,
		helpers: helpers,
		current: s.Fields,
		depth:   0,
	}
	w.walkProgram(program)

	return s
}

type schemaWalker struct {
	schema     *templateSchema
	helpers    map[string]bool
	current    map[string]*fieldInfo // current field scope
	conditions int                    // depth of conditional nesting
	depth      int                    // context depth (for ../ resolution)
}

func (w *schemaWalker) isHelper(name string) bool {
	switch name {
	case "if", "unless", "each", "with", "lookup", "log", "equal", "raw":
		return true
	}
	if w.helpers != nil {
		return w.helpers[name]
	}
	return false
}

func (w *schemaWalker) ensureField(path string) *fieldInfo {
	parts := strings.Split(path, ".")
	scope := w.current

	var fi *fieldInfo
	for i, part := range parts {
		if part == "this" || part == "." || part == "" {
			continue
		}
		// Skip @data variables
		if strings.HasPrefix(part, "@") {
			return nil
		}

		existing, ok := scope[part]
		if !ok {
			existing = &fieldInfo{
				Name:     toGoName(part),
				JSONName: part,
				Type:     typeString,
				Required: w.conditions == 0,
				Children: make(map[string]*fieldInfo),
			}
			scope[part] = existing
		}

		// Upgrade to required if this occurrence is unconditional
		if w.conditions == 0 {
			existing.Required = true
		}

		// If there are more parts, this must be a struct
		if i < len(parts)-1 {
			if existing.Type == typeString {
				existing.Type = typeStruct
			}
			scope = existing.Children
		}

		fi = existing
	}

	return fi
}

func (w *schemaWalker) walkProgram(node *ast.Program) {
	for _, n := range node.Body {
		w.walkNode(n)
	}
}

func (w *schemaWalker) walkNode(node ast.Node) {
	switch n := node.(type) {
	case *ast.MustacheStatement:
		w.walkExpression(n.Expression)
	case *ast.BlockStatement:
		w.walkBlock(n)
	case *ast.PartialStatement:
		w.walkPartial(n)
	}
}

func (w *schemaWalker) walkExpression(node *ast.Expression) {
	hasParams := len(node.Params) > 0
	hasHash := node.Hash != nil

	if path, ok := node.Path.(*ast.PathExpression); ok {
		helperName := node.HelperName()
		if hasParams || hasHash {
			// Helper call — walk params
			for _, param := range node.Params {
				w.walkParam(param)
			}
			if node.Hash != nil {
				w.walkHash(node.Hash)
			}
		} else if helperName != "" && w.isHelper(helperName) {
			// Zero-arg helper — skip
		} else if !path.Data {
			// Variable reference
			w.ensureField(path.Original)
		}
	} else {
		// Literal or subexpression — walk params
		for _, param := range node.Params {
			w.walkParam(param)
		}
	}
}

func (w *schemaWalker) walkParam(node ast.Node) {
	switch n := node.(type) {
	case *ast.PathExpression:
		if !n.Data && n.Original != "this" && n.Original != "." {
			w.ensureField(n.Original)
		}
	case *ast.SubExpression:
		w.walkExpression(n.Expression)
	}
}

func (w *schemaWalker) walkHash(node *ast.Hash) {
	for _, pair := range node.Pairs {
		w.walkParam(pair.Val)
	}
}

func (w *schemaWalker) walkBlock(node *ast.BlockStatement) {
	helperName := node.Expression.HelperName()

	switch helperName {
	case "if", "unless":
		w.walkConditionalBlock(node)
	case "each":
		w.walkEachBlock(node)
	case "with":
		w.walkWithBlock(node)
	default:
		// Custom block helper or decorator
		if node.Decorator {
			return
		}
		w.walkExpression(node.Expression)
		if node.Program != nil {
			w.walkProgram(node.Program)
		}
		if node.Inverse != nil {
			w.walkProgram(node.Inverse)
		}
	}
}

func (w *schemaWalker) walkConditionalBlock(node *ast.BlockStatement) {
	// The condition variable is used as a boolean test
	if len(node.Expression.Params) > 0 {
		if path, ok := node.Expression.Params[0].(*ast.PathExpression); ok {
			if !path.Data {
				fi := w.ensureField(path.Original)
				if fi != nil && fi.Type == typeString {
					fi.Type = typeInterface // used as condition, could be any truthy type
				}
			}
		}
		// Walk subexpression params
		for _, param := range node.Expression.Params {
			if _, ok := param.(*ast.SubExpression); ok {
				w.walkParam(param)
			}
		}
	}

	// Walk branches with increased condition depth
	w.conditions++
	if node.Program != nil {
		w.walkProgram(node.Program)
	}
	if node.Inverse != nil {
		w.walkProgram(node.Inverse)
	}
	w.conditions--
}

func (w *schemaWalker) walkEachBlock(node *ast.BlockStatement) {
	// The iteration target is a slice
	if len(node.Expression.Params) > 0 {
		if path, ok := node.Expression.Params[0].(*ast.PathExpression); ok {
			if !path.Data {
				fi := w.ensureField(path.Original)
				if fi != nil {
					fi.Type = typeSlice
				}
			}
		}
	}

	// Walk body in a child scope — variables inside #each are fields of the element
	if node.Program != nil {
		// Find the iteration field to attach child fields to
		var elementFields map[string]*fieldInfo
		if len(node.Expression.Params) > 0 {
			if path, ok := node.Expression.Params[0].(*ast.PathExpression); ok {
				if fi := w.findField(path.Original); fi != nil {
					elementFields = fi.Children
				}
			}
		}

		if elementFields != nil {
			oldCurrent := w.current
			w.current = elementFields
			w.walkProgram(node.Program)
			w.current = oldCurrent
		} else {
			w.walkProgram(node.Program)
		}
	}
	if node.Inverse != nil {
		w.walkProgram(node.Inverse)
	}
}

func (w *schemaWalker) walkWithBlock(node *ast.BlockStatement) {
	// The context target is a struct
	if len(node.Expression.Params) > 0 {
		if path, ok := node.Expression.Params[0].(*ast.PathExpression); ok {
			if !path.Data {
				fi := w.ensureField(path.Original)
				if fi != nil {
					fi.Type = typeStruct
				}
			}
		}
	}

	// Walk body in a child scope
	if node.Program != nil {
		if len(node.Expression.Params) > 0 {
			if path, ok := node.Expression.Params[0].(*ast.PathExpression); ok {
				if fi := w.findField(path.Original); fi != nil {
					oldCurrent := w.current
					w.current = fi.Children
					w.walkProgram(node.Program)
					w.current = oldCurrent
					return
				}
			}
		}
		w.walkProgram(node.Program)
	}
	if node.Inverse != nil {
		w.walkProgram(node.Inverse)
	}
}

func (w *schemaWalker) walkPartial(node *ast.PartialStatement) {
	for _, param := range node.Params {
		w.walkParam(param)
	}
	if node.Hash != nil {
		w.walkHash(node.Hash)
	}
}

// findField looks up a field by dot-path in the current scope.
func (w *schemaWalker) findField(path string) *fieldInfo {
	parts := strings.Split(path, ".")
	scope := w.current

	var fi *fieldInfo
	for _, part := range parts {
		if part == "this" || part == "." || part == "" {
			continue
		}
		existing, ok := scope[part]
		if !ok {
			return nil
		}
		fi = existing
		scope = existing.Children
	}
	return fi
}

// sortedFields returns fields sorted alphabetically by JSON name.
func sortedFields(fields map[string]*fieldInfo) []*fieldInfo {
	result := make([]*fieldInfo, 0, len(fields))
	for _, f := range fields {
		result = append(result, f)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].JSONName < result[j].JSONName
	})
	return result
}

// toGoName converts a template field name to a Go-exported name.
// e.g., "first_name" -> "FirstName", "showAvatar" -> "ShowAvatar"
func toGoName(s string) string {
	if s == "" {
		return ""
	}

	// Handle snake_case and kebab-case
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-'
	})

	var result strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		result.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			result.WriteString(part[1:])
		}
	}

	name := result.String()
	if name == "" {
		return strings.ToUpper(s[:1]) + s[1:]
	}
	return name
}
