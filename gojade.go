package gojade

import (
	"bytes"
	"github.com/zdebeer99/gojade/parser"
)

type GoJade struct {
	ViewPath string
	Beautify bool
}

func NewGoJade() *GoJade {
	gojade := new(GoJade)
	return gojade
}

func (this *GoJade) RenderFile(filename string, data map[string]interface{}) *bytes.Buffer {
	buf := new(bytes.Buffer)
	eval := parser.NewEvalJade(buf)
	eval.SetViewPath(this.ViewPath)
	eval.SetData(data)
	eval.Beautify = this.Beautify
	eval.RenderFile(filename)
	return buf
}

func (this *GoJade) RenderString(template string, data map[string]interface{}) *bytes.Buffer {
	buf := new(bytes.Buffer)
	eval := parser.NewEvalJade(buf)
	eval.SetViewPath(this.ViewPath)
	eval.SetData(data)
	eval.Beautify = this.Beautify
	eval.RenderString(template)
	return buf
}
