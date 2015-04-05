package gojade

import (
	"bytes"
	"github.com/zdebeer99/gojade/parser"
	"io"
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

// RenderFile Renders a jade file to a bytes.Buffer.
// Example: jade.RenderFile("index.jade",nil).String()
func (this *GoJade) RenderFile(filename string, data interface{}) *bytes.Buffer {
	buf := new(bytes.Buffer)
	eval := this.init(buf)
	eval.SetData(data)
	eval.RenderFile(filename)
	return buf
}

func (this *GoJade) RenderString(template string, data interface{}) *bytes.Buffer {
	buf := new(bytes.Buffer)
	eval := this.init(buf)
	eval.SetData(data)
	eval.RenderString(template)
	return buf
}

// RenderFileW Render a jade file to a io.writer stream.
func (this *GoJade) RenderFileW(wr io.Writer, template string, data interface{}) error {
	eval := this.init(wr)
	eval.SetData(data)
	eval.RenderFile(template)
	return nil
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

func (this *GoJade) init(writer io.Writer) *parser.EvalJade {
	eval := parser.NewEvalJade(writer)
	eval.SetViewPath(this.ViewPath)
	eval.Beautify = this.Beautify
	eval.Extfunc = this.extfunc
	return eval
}
