package jadeparser

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// Set testonly to a test number to only test that example.
var testonly int = -1

// Test Samples in file delimited by @jade, @html keywords. see 'verifyjade.jade' for example.
func TestVerifyJade(t *testing.T) {
	data := &VerifyStruct{"Hello Jade", false, 5, 10, []int{32, 37, 38, 42}, &PersonModel{"Ben", 32}}
	content, err := load("../res/verifyjade.jade")
	if err != nil {
		t.Errorf("Error loading file for testing. %v", err)
		return
	}
	validation := parseverifyfile(content)

	if testonly > -1 {
		t.Logf("Testing only %v. %s", testonly, validation[testonly].name)
		result := Parse(validation[testonly].jade)
		t.Logf(result.Root.String())
		if result.Err != nil {
			t.Error(result.Err)
		}
		evaltest(t, testonly, validation[testonly], data)

	} else {
		for i, item := range validation {
			if len(item.jade) > 0 {
				evaltest(t, i, item, data)
			}
		}
	}
}

// Render Jade file found on jade-lang home page.
func TestParseJade(t *testing.T) {
	template, err := load("../res/test.jade")
	if err != nil {
		t.Error(err)
		return
	}
	result := Parse(template)
	if result.Err != nil {
		fmt.Println(result.Err)
	}
	//fmt.Println(result.Root)
}

// Test parsing jade extends functions.
func TestEvalExtends(t *testing.T) {
	buf := new(bytes.Buffer)
	eval := NewEvalJade(buf)
	eval.SetViewPath("../res/extends")
	eval.RenderFile("index.jade")
	if buf.String() != "<!DOCTYPE html><html><head><title>Article Title</title></head><body></body><h1>This Part is from the layout.jade file</h1><h2>This is from index.jade</h2></html>" {
		t.Error("Html does not match")
	}
}

type VerifyStruct struct {
	PageTitle       string
	YouAreUsingJade bool
	Num1            int
	Num2            int
	List            []int
	Person          *PersonModel
}

func (this *VerifyStruct) Calc1(Num3 int) int {
	return this.Num1 * this.Num2 * Num3
}

func (this *VerifyStruct) Calc0() int {
	return this.Num1 * this.Num2
}

type PersonModel struct {
	Name string
	Age  int
}

func (this *PersonModel) Born() int {
	return time.Now().Year() - this.Age
}

// store jade and html result for comparison.
type verifyItem struct {
	name string
	jade string
	html string
}

// Load a test file
func load(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Save a test result
func save(filename string, data []byte) {
	ioutil.WriteFile(filename, data, os.ModePerm)
}

// RenderJade shortcut function.
func renderJade(buf *bytes.Buffer, template string, data interface{}) *EvalJade {
	eval := NewEvalJade(buf)
	eval.SetData(data)
	eval.RegisterFunction("safeDivide", func(v1, v2 int) int {
		if v2 == 0 {
			return 0
		}
		return v1 / v2
	})
	eval.RegisterFunction("hello", func(name string) string {
		return "Hello " + name
	})
	eval.RegisterFunction("number5", func() string {
		return "Five"
	})
	eval.RenderString(template)
	return eval
}

// evaluate jade
func evaltest(t *testing.T, i int, item *verifyItem, data interface{}) {
	buf := new(bytes.Buffer)
	t.Logf("Testing %v. %s", i, item.name)
	j := renderJade(buf, item.jade, data)
	if len(j.Log) > 0 {
		for _, item := range j.Log {
			t.Log(item)
		}
	}
	if buf.String() != item.html {
		t.Logf(" (Failed)")
		t.Errorf("%v. %s Failed. Html does not match. \nJade:\n[%s]\nParse To:\n[%s]\nExpected:\n[%s]\n", i, item.name, item.jade, buf.String(), item.html)
	} else {
		t.Logf(" (Passed)")
	}
}

//parseverifyfile
func parseverifyfile(content string) []*verifyItem {
	result := make([]*verifyItem, 0)
	part := new(bytes.Buffer)
	line := new(bytes.Buffer)
	item := new(verifyItem)
	mode := 0
	pos := 0
	for {
		r, w := utf8.DecodeRuneInString(content[pos:])
		line.WriteRune(r)
		pos += w
		if r == '\n' {
			if strings.HasPrefix(line.String(), "@jade") {
				item = new(verifyItem)
				caption := line.String()
				if len(caption) > len("@jade\n") {
					item.name = caption[len("@jade") : len(caption)-1]
				}
				mode = 1
				part.Reset()
				line.Reset()
				continue
			}
			if strings.HasPrefix(line.String(), "@html") {
				if item.jade == "" && mode == 1 {
					item.jade = getPart(part)
					part.Reset()
				}
				mode = 2
				line.Reset()
				continue
			}
			if strings.HasPrefix(line.String(), "@end") {
				switch mode {
				case 1:
					item.jade = getPart(part)
					part.Reset()
				case 2:
					item.html = getPart(part)
					result = append(result, item)
					part.Reset()
				}
				mode = 0
				line.Reset()
				continue
			}
			if line.Len() > 0 && mode > 0 {
				part.Write(line.Bytes())
			}
			line.Reset()
		}
		if pos >= len(content) || r == -1 {
			if part.Len() > 0 && mode == 2 {
				item.html = getPart(part)
				result = append(result, item)
			}
			break
		}
	}
	return result
}

func getPart(part *bytes.Buffer) string {
	if part.Len() > 1 {
		partstr := part.String()
		part.Reset()
		return partstr[:len(partstr)-1]
	}
	return ""
}
