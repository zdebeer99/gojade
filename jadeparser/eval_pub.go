package jadeparser

import (
	"io"
	"reflect"
)

type EvalJade struct {
	Loader   TemplateLoader
	data     reflect.Value
	builtin  map[string]reflect.Value
	Extfunc  map[string]reflect.Value
	writer   *jadewriter
	doctype  string
	stack    *ContextStack
	Blocks   map[string]*TreeNode
	Mixins   map[string]*TreeNode
	Beautify bool
	Log      []string
}

func NewEvalJade(wr io.Writer) *EvalJade {
	eval := new(EvalJade)
	eval.Loader = new(templateLoader)
	eval.writer = &jadewriter{wr, eval, false}
	eval.builtin = createValueFuncs(builtin)
	eval.Extfunc = make(map[string]reflect.Value)
	eval.registerStandardFunctions()
	eval.stack = NewContextStack()
	eval.Blocks = make(map[string]*TreeNode)
	eval.Mixins = make(map[string]*TreeNode)
	eval.Log = make([]string, 0)
	return eval
}

func (this *EvalJade) SetData(data interface{}) {
	this.data = reflect.ValueOf(data)
}

func (this *EvalJade) SetViewPath(viewpath string) {
	this.Loader.SetViewPath(viewpath)
}

func (this *EvalJade) BuildJadeFromParseResult(result *ParseResult) {
	if result.Err != nil {
		panic(result.Err)
	}
	for k, v := range result.Mixins {
		if _, ok := this.Mixins[k]; !ok {
			this.Mixins[k] = v
		}
	}
	for k, v := range result.Blocks {
		if _, ok := this.Blocks[k]; !ok {
			this.Blocks[k] = v
		}
	}
}

func (this *EvalJade) Exec(parsedJade *TreeNode) {
	this.router(parsedJade)
}

func (this *EvalJade) RegisterFunction(name string, fn interface{}) {
	registerFunction(this.Extfunc, name, fn)
}

func (this *EvalJade) RenderFile(filename string) {
	template := this.evalFile(filename)
	if template.Template == nil {
		return
	}
	for len(template.Template.Extends) > 0 {
		template = this.evalFile(template.Template.Extends)
	}
	this.Exec(template.Template.Root)
}

func (this *EvalJade) RenderString(template string) {

	parse := Parse(template)
	this.BuildJadeFromParseResult(parse)
	for len(parse.Extends) > 0 {
		this.RenderFile(parse.Extends)
	}
	this.Exec(parse.Root)
	return
}
