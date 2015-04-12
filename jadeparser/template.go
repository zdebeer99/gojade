package jadeparser

import (
	"io/ioutil"
	"path/filepath"
	"strings"
)

type Template struct {
	Name     string
	File     []byte
	Template *ParseResult
	IsJade   bool
}

type TemplateLoader interface {
	SetViewPath(string)
	Load(string) *Template
}

//Default Template Loader
type templateLoader struct {
	viewPath string
}

func (this *templateLoader) SetViewPath(path string) {
	this.viewPath = path
}

func (this *templateLoader) Load(name string) *Template {
	filename, err := this.findfile(name)
	if err != nil {
		panic(err)
	}
	file, err := this.loadfile(filename)
	if err != nil {
		panic(err)
	}
	template := new(Template)
	template.Name = name
	template.File = file

	if this.isJadeFile(filename) {
		template.IsJade = true
		template.Template = Parse(string(template.File))
		return template
	} else {
		template.IsJade = false
		return template
	}
}

// floadfile Find and Load a file.
func (this *templateLoader) floadfile(filename string) ([]byte, error) {
	filename, err := this.findfile(filename)
	if err != nil {
		return nil, err
	}
	return this.loadfile(filename)
}

func (this *templateLoader) findfile(filename string) (string, error) {
	filename = strings.Trim(filename, " ")
	if !strings.ContainsRune(filename, '.') {
		filename = filename + ".jade"
	}
	if !strings.HasSuffix(this.viewPath, "/") {
		filename = this.viewPath + "/" + filename
	} else {
		filename = this.viewPath + filename
	}
	filename, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}
	return filename, nil
}

func (this *templateLoader) isJadeFile(filename string) bool {
	return strings.HasSuffix(filename, ".jade")
}

func (this *templateLoader) loadfile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}
