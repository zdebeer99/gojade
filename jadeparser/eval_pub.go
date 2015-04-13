package jadeparser

import (
	"io"
	"reflect"
)

type EvalJade struct {
	Loader       TemplateLoader
	currTemplate *Template //Used for debugging purposes.
	currPart     *jadePart
	data         reflect.Value
	builtin      map[string]reflect.Value
	Extfunc      map[string]reflect.Value
	writer       *jadewriter
	doctype      string
	stack        *ContextStack
	Blocks       map[string]*jadePart
	Mixins       map[string]*jadePart
	Beautify     bool
	Log          []string
}

func NewEvalJade(wr io.Writer) *EvalJade {
	eval := new(EvalJade)
	eval.Loader = new(templateLoader)
	eval.writer = &jadewriter{wr, eval, false}
	eval.builtin = createValueFuncs(builtin)
	eval.Extfunc = make(map[string]reflect.Value)
	eval.registerStandardFunctions()
	eval.stack = NewContextStack()
	eval.Blocks = make(map[string]*jadePart)
	eval.Mixins = make(map[string]*jadePart)
	eval.Log = make([]string, 0)
	return eval
}

func (this *EvalJade) SetData(data interface{}) {
	this.data = reflect.ValueOf(data)
}

func (this *EvalJade) SetViewPath(viewpath string) {
	this.Loader.SetViewPath(viewpath)
}

func (this *EvalJade) buildJadeFromParseResult(template *Template) {
	result := template.Root
	if result.Err != nil {
		panic(result.Err)
	}
	for k, v := range result.Mixins {
		if _, ok := this.Mixins[k]; !ok {
			this.Mixins[k] = &jadePart{template.Name, v, template.File}
		}
	}
	for k, v := range result.Blocks {
		if _, ok := this.Blocks[k]; !ok {
			this.Blocks[k] = &jadePart{template.Name, v, template.File}
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
	this.evalFile(filename)
}

func (this *EvalJade) RenderString(template string) {
	parse := Parse(template)
	this.buildJadeFromParseResult(&Template{"fromstring", []byte(template), parse, true})
	if len(parse.Extends) > 0 {
		this.RenderFile(parse.Extends)
	}
	this.currTemplate = &Template{Name: "fromstring", File: []byte(template), Root: parse, IsJade: true}
	this.Exec(parse.Root)
	return
}
