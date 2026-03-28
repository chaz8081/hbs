package handlebars

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/chaz8081/handlebars-go/v4/ast"
)

// ValidationError represents a missing field found during template validation.
type ValidationError struct {
	Path    string // The dot-separated field path that is missing
	Message string // Human-readable description
}

func (e ValidationError) Error() string {
	return e.Message
}

// Validate checks that the provided data contains all fields required by the
// template for the active execution path. It is condition-aware: if a field is
// inside an {{#if}} block and the condition is falsy in the actual data, those
// fields are not checked. Returns a list of all missing fields (collect-all mode).
//
// The helpers parameter provides known helper names for disambiguation, same as
// ExtractVariables. Pass nil if no disambiguation is needed.
func Validate(tpl *Template, data interface{}, helpers map[string]bool) []ValidationError {
	program := tpl.AST()
	if program == nil {
		return nil
	}

	v := &validator{
		data:    data,
		helpers: helpers,
		dataVal: reflect.ValueOf(data),
	}

	v.validateProgram(program, v.dataVal)

	return v.errors
}

// validator walks the AST condition-aware, checking actual data.
type validator struct {
	data    interface{}
	helpers map[string]bool
	dataVal reflect.Value
	errors  []ValidationError
}

func (v *validator) addError(path string) {
	v.errors = append(v.errors, ValidationError{
		Path:    path,
		Message: fmt.Sprintf("required field missing: %s", path),
	})
}

// isHelper checks if a name is a known helper.
func (v *validator) isHelper(name string) bool {
	if builtInBlockHelpers[name] {
		return true
	}
	switch name {
	case "lookup", "log", "equal":
		return true
	}
	if v.helpers != nil {
		return v.helpers[name]
	}
	return false
}

// resolvePathInData checks if a dot-path exists in the given data context.
func (v *validator) resolvePathInData(path string, ctx reflect.Value) bool {
	parts := strings.Split(path, ".")
	current := ctx

	for _, part := range parts {
		current = resolveField(current, part)
		if !current.IsValid() {
			return false
		}
		// Check if the value is a nil pointer/interface
		if isNilValue(current) {
			return false
		}
	}

	return true
}

// resolveTruthyInData checks if a path resolves to a truthy value.
func (v *validator) resolveTruthyInData(path string, ctx reflect.Value) (exists bool, truthy bool) {
	parts := strings.Split(path, ".")
	current := ctx

	for _, part := range parts {
		current = resolveField(current, part)
		if !current.IsValid() {
			return false, false
		}
		if isNilValue(current) {
			return true, false // exists but is nil (falsy)
		}
	}

	// Dereference interfaces/pointers to get the actual value for truthiness check
	dereffed := derefValue(current)
	if !dereffed.IsValid() {
		return true, false
	}

	truth, _ := isTrueValue(dereffed)
	return true, truth
}

// derefValue dereferences pointers and interfaces to get the underlying value.
func derefValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

// resolveField looks up a single field name in a reflect.Value.
func resolveField(ctx reflect.Value, name string) reflect.Value {
	// Dereference pointers and interfaces
	for ctx.Kind() == reflect.Ptr || ctx.Kind() == reflect.Interface {
		if ctx.IsNil() {
			return reflect.Value{}
		}
		ctx = ctx.Elem()
	}

	if !ctx.IsValid() {
		return reflect.Value{}
	}

	switch ctx.Kind() {
	case reflect.Map:
		nameVal := reflect.ValueOf(name)
		if nameVal.Type().AssignableTo(ctx.Type().Key()) {
			return ctx.MapIndex(nameVal)
		}
	case reflect.Struct:
		// Try exact name, then title-cased
		expName := strings.Title(name)
		if tField, ok := ctx.Type().FieldByName(expName); ok && tField.PkgPath == "" {
			return ctx.FieldByIndex(tField.Index)
		}
		// Try struct tags
		for i := 0; i < ctx.NumField(); i++ {
			field := ctx.Type().Field(i)
			if field.Tag.Get("handlebars") == name {
				return ctx.Field(i)
			}
		}
	}

	return reflect.Value{}
}

// isNilValue checks if a reflect.Value represents nil.
func isNilValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return v.IsNil()
	}
	return false
}

func (v *validator) validateProgram(node *ast.Program, ctx reflect.Value) {
	for _, n := range node.Body {
		v.validateNode(n, ctx)
	}
}

func (v *validator) validateNode(node ast.Node, ctx reflect.Value) {
	switch n := node.(type) {
	case *ast.MustacheStatement:
		v.validateMustache(n, ctx)
	case *ast.BlockStatement:
		v.validateBlock(n, ctx)
	case *ast.PartialStatement:
		v.validatePartial(n, ctx)
	}
}

func (v *validator) validateMustache(node *ast.MustacheStatement, ctx reflect.Value) {
	v.validateExpression(node.Expression, ctx)
}

func (v *validator) validateExpression(node *ast.Expression, ctx reflect.Value) {
	hasParams := len(node.Params) > 0
	hasHash := node.Hash != nil

	if path, ok := node.Path.(*ast.PathExpression); ok {
		helperName := node.HelperName()

		if hasParams || hasHash {
			// Helper call — validate params and hash values
			v.validateExpressionParams(node, ctx)
		} else if helperName != "" && v.isHelper(helperName) {
			// Zero-arg helper — skip
		} else {
			// Variable reference — check if it exists in data
			if !path.Data && path.Original != "this" && path.Original != "." {
				if !v.resolvePathInData(path.Original, ctx) {
					v.addError(path.Original)
				}
			}
		}
	} else {
		v.validateExpressionParams(node, ctx)
	}
}

func (v *validator) validateExpressionParams(node *ast.Expression, ctx reflect.Value) {
	for _, param := range node.Params {
		v.validateParamNode(param, ctx)
	}
	if node.Hash != nil {
		v.validateHash(node.Hash, ctx)
	}
}

func (v *validator) validateParamNode(node ast.Node, ctx reflect.Value) {
	switch n := node.(type) {
	case *ast.PathExpression:
		if !n.Data && n.Original != "this" && n.Original != "." {
			if !v.resolvePathInData(n.Original, ctx) {
				v.addError(n.Original)
			}
		}
	case *ast.SubExpression:
		v.validateExpression(n.Expression, ctx)
	}
}

func (v *validator) validateHash(node *ast.Hash, ctx reflect.Value) {
	for _, pair := range node.Pairs {
		v.validateParamNode(pair.Val, ctx)
	}
}

func (v *validator) validateBlock(node *ast.BlockStatement, ctx reflect.Value) {
	helperName := node.Expression.HelperName()

	switch helperName {
	case "if":
		v.validateIfBlock(node, ctx)
	case "unless":
		v.validateUnlessBlock(node, ctx)
	case "each":
		v.validateEachBlock(node, ctx)
	case "with":
		v.validateWithBlock(node, ctx)
	default:
		// Custom block helper — validate expression and both branches
		v.validateExpression(node.Expression, ctx)
		if node.Program != nil {
			v.validateProgram(node.Program, ctx)
		}
		if node.Inverse != nil {
			v.validateProgram(node.Inverse, ctx)
		}
	}
}

func (v *validator) validateIfBlock(node *ast.BlockStatement, ctx reflect.Value) {
	if len(node.Expression.Params) == 0 {
		return
	}

	condPath := ""
	if path, ok := node.Expression.Params[0].(*ast.PathExpression); ok {
		condPath = path.Original
	}

	if condPath == "" {
		// Condition is a subexpression or literal — can't resolve statically
		// Validate both branches conservatively
		if node.Program != nil {
			v.validateProgram(node.Program, ctx)
		}
		if node.Inverse != nil {
			v.validateProgram(node.Inverse, ctx)
		}
		return
	}

	exists, truthy := v.resolveTruthyInData(condPath, ctx)
	if !exists {
		// Condition variable itself is missing
		v.addError(condPath)
		return
	}

	if truthy {
		// Condition is truthy — validate main branch
		if node.Program != nil {
			v.validateProgram(node.Program, ctx)
		}
	} else {
		// Condition is falsy — validate else branch
		if node.Inverse != nil {
			v.validateProgram(node.Inverse, ctx)
		}
	}
}

func (v *validator) validateUnlessBlock(node *ast.BlockStatement, ctx reflect.Value) {
	if len(node.Expression.Params) == 0 {
		return
	}

	condPath := ""
	if path, ok := node.Expression.Params[0].(*ast.PathExpression); ok {
		condPath = path.Original
	}

	if condPath == "" {
		if node.Program != nil {
			v.validateProgram(node.Program, ctx)
		}
		if node.Inverse != nil {
			v.validateProgram(node.Inverse, ctx)
		}
		return
	}

	exists, truthy := v.resolveTruthyInData(condPath, ctx)
	if !exists {
		v.addError(condPath)
		return
	}

	// unless is inverse of if
	if !truthy {
		// Condition is falsy — block runs
		if node.Program != nil {
			v.validateProgram(node.Program, ctx)
		}
	} else {
		// Condition is truthy — else block runs
		if node.Inverse != nil {
			v.validateProgram(node.Inverse, ctx)
		}
	}
}

func (v *validator) validateEachBlock(node *ast.BlockStatement, ctx reflect.Value) {
	if len(node.Expression.Params) == 0 {
		return
	}

	if path, ok := node.Expression.Params[0].(*ast.PathExpression); ok {
		if !path.Data {
			if !v.resolvePathInData(path.Original, ctx) {
				v.addError(path.Original)
			}
		}
	}

	// We don't validate inside #each body against the outer context
	// because the body runs in the iteration item's context
}

func (v *validator) validateWithBlock(node *ast.BlockStatement, ctx reflect.Value) {
	if len(node.Expression.Params) == 0 {
		return
	}

	if path, ok := node.Expression.Params[0].(*ast.PathExpression); ok {
		if !path.Data {
			if !v.resolvePathInData(path.Original, ctx) {
				v.addError(path.Original)
			}
		}
	}

	// #with shifts context — we don't validate body against outer context
}

func (v *validator) validatePartial(node *ast.PartialStatement, ctx reflect.Value) {
	for _, param := range node.Params {
		v.validateParamNode(param, ctx)
	}
	if node.Hash != nil {
		v.validateHash(node.Hash, ctx)
	}
}
