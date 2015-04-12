package gojade

import (
	"io/ioutil"

	"github.com/zdebeer99/gojade/jadeparser"
)

type templateLoader struct {
	ViewPath string
}

func (this *templateLoader) Load(name string) *jadeparser.Template {
	b, err := ioutil.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return &jadeparser.Template{File: b}
}
