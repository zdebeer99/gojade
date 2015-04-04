package gojade

import (
	"bytes"
	"github.com/zdebeer99/gojade/parser"
	"reflect"
)

type GoJade struct {
	ViewPath string
	Beautify bool
	extfunc  map[string]reflect.Value
}

func NewGoJade() *GoJade {
	gojade := new(GoJade)
	gojade.extfunc = make(map[string]reflect.Value)
	return gojade
}

func (this *GoJade) RenderFile(filename string, data map[string]interface{}) *bytes.Buffer {
	buf := new(bytes.Buffer)
	eval := this.init(buf)
	eval.SetData(data)
	eval.RenderFile(filename)
	return buf
}

func (this *GoJade) RenderString(template string, data map[string]interface{}) *bytes.Buffer {
	buf := new(bytes.Buffer)
	eval := this.init(buf)
	eval.SetData(data)
	eval.RenderString(template)
	return buf
}

func (this *GoJade) RegisterFunction(name string, fn interface{}) {
	fnvalue := reflect.ValueOf(fn)
	switch fnvalue.Kind() {
	case reflect.Func:
		this.extfunc[name] = fnvalue
	default:
		panic("argument 'fn' is not a function. " + fnvalue.String())
	}
}

func (this *GoJade) init(buf *bytes.Buffer) *parser.EvalJade {
	eval := parser.NewEvalJade(buf)
	eval.SetViewPath(this.ViewPath)
	eval.Beautify = this.Beautify
	eval.Extfunc = this.extfunc
	return eval
}
