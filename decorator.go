package handlebars

import (
	"fmt"
	"sync"

	"github.com/chaz8081/handlebars-go/v4/ast"
)

// DecoratorOptions represents the options passed to a decorator function.
type DecoratorOptions struct {
	// eval is the evaluation visitor
	eval *evalVisitor

	// params holds positional parameters passed to the decorator
	params []interface{}

	// hash holds named hash parameters
	hash map[string]interface{}

	// name of the decorator
	name string

	// program is the block content of the decorator (rendered via Fn())
	program func() string

	// astProgram is the raw AST program node for the decorator block
	astProgram *ast.Program
}

// Param returns parameter at given position.
func (d *DecoratorOptions) Param(pos int) interface{} {
	if len(d.params) > pos {
		return d.params[pos]
	}
	return nil
}

// ParamStr returns string representation of parameter at given position.
func (d *DecoratorOptions) ParamStr(pos int) string {
	return Str(d.Param(pos))
}

// Params returns all parameters.
func (d *DecoratorOptions) Params() []interface{} {
	return d.params
}

// Hash returns the entire hash.
func (d *DecoratorOptions) Hash() map[string]interface{} {
	return d.hash
}

// HashProp returns a hash property.
func (d *DecoratorOptions) HashProp(name string) interface{} {
	return d.hash[name]
}

// HashStr returns string representation of a hash property.
func (d *DecoratorOptions) HashStr(name string) string {
	return Str(d.hash[name])
}

// Name returns the decorator name.
func (d *DecoratorOptions) Name() string {
	return d.name
}

// Fn evaluates the decorator block content and returns the result.
func (d *DecoratorOptions) Fn() string {
	if d.program != nil {
		return d.program()
	}
	return ""
}

// RegisterInlinePartial registers an inline partial in the current template scope.
// This is the primary use case for decorators in Handlebars v4.
func (d *DecoratorOptions) RegisterInlinePartial(name string, content string) {
	tpl, err := Parse(content)
	if err != nil {
		d.eval.errPanic(err)
	}
	d.eval.inlinePartials[name] = newPartial(name, content, tpl)
}

// DecoratorFunc is the signature for decorator functions.
type DecoratorFunc func(options *DecoratorOptions) interface{}

// decorators stores all globally registered decorators
var decorators = make(map[string]DecoratorFunc)

// protects global decorators
var decoratorsMutex sync.RWMutex

func init() {
	// register built-in inline decorator
	RegisterDecorator("inline", inlineDecoratorFn)
}

// RegisterDecorator registers a global decorator.
func RegisterDecorator(name string, decorator DecoratorFunc) {
	decoratorsMutex.Lock()
	defer decoratorsMutex.Unlock()

	if _, exists := decorators[name]; exists {
		panic(fmt.Errorf("Decorator already registered: %s", name))
	}

	decorators[name] = decorator
}

// RemoveDecorator unregisters a global decorator.
func RemoveDecorator(name string) {
	decoratorsMutex.Lock()
	defer decoratorsMutex.Unlock()

	delete(decorators, name)
}

// RemoveAllDecorators unregisters all global decorators.
func RemoveAllDecorators() {
	decoratorsMutex.Lock()
	defer decoratorsMutex.Unlock()

	decorators = make(map[string]DecoratorFunc)
}

// findGlobalDecorator finds a globally registered decorator.
func findGlobalDecorator(name string) DecoratorFunc {
	decoratorsMutex.RLock()
	defer decoratorsMutex.RUnlock()

	return decorators[name]
}

// inlineDecoratorFn is the built-in decorator for defining inline partials.
// It uses the AST program directly to preserve template context variables like {{this}}.
func inlineDecoratorFn(options *DecoratorOptions) interface{} {
	name := options.ParamStr(0)
	if name == "" {
		options.eval.errorf("inline decorator requires a partial name argument")
	}

	if options.astProgram != nil {
		tpl := NewTemplateFromAST(options.astProgram)
		options.eval.inlinePartials[name] = newPartial(name, "", tpl)
	}

	return ""
}
