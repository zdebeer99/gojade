package parser

import (
	"io"
	"reflect"
)

type EvalJade struct {
	data     dataMap
	builtin  map[string]reflect.Value
	Extfunc  map[string]reflect.Value
	writer   *jadewriter
	doctype  string
	stack    *ContextStack
	Blocks   map[string]*TreeNode
	Mixins   map[string]*TreeNode
	Beautify bool
	Log      []string
	viewPath string
}

func NewEvalJade(wr io.Writer) *EvalJade {
	eval := new(EvalJade)
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

func (this *EvalJade) SetData(data map[string]interface{}) {
	this.data = data
}

func (this *EvalJade) SetViewPath(viewpath string) {
	this.viewPath = viewpath
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
	fnvalue := reflect.ValueOf(fn)
	switch fnvalue.Kind() {
	case reflect.Func:
		this.Extfunc[name] = fnvalue
	default:
		panic("argument is not a function. " + fnvalue.String())
	}
}

func (this *EvalJade) evalFile(filename string) *ParseResult {
	filename, err := this.findfile(filename)
	if err != nil {
		panic(err)
	}
	template, err := this.loadfile(filename)
	if err != nil {
		panic(err)
	}
	if this.isJadeFile(filename) {
		parse := Parse(string(template))
		this.BuildJadeFromParseResult(parse)
		return parse
	} else {
		this.writeText(string(template))
		return nil
	}
}

func (this *EvalJade) RenderFile(filename string) {
	parse := this.evalFile(filename)
	if parse == nil {
		return
	}
	for len(parse.Extends) > 0 {
		parse = this.evalFile(parse.Extends)
	}
	this.Exec(parse.Root)
}

func (this *EvalJade) RenderString(template string) {

	parse := Parse(template)
	this.BuildJadeFromParseResult(parse)
	for len(parse.Extends) > 0 {
		template, err := this.loadfile(parse.Extends)
		if err != nil {
			panic(err)
		}
		parse = Parse(string(template))
		this.BuildJadeFromParseResult(parse)
	}
	this.Exec(parse.Root)
	return
}
