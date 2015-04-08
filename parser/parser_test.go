package parser

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

// Set testonly to a test number to only test that example.
var testonly int = -1

// Test Samples in file delimited by @jade, @html keywords. see 'verifyjade.jade' for example.
func TestVerifyJade(t *testing.T) {
	data := map[string]interface{}{
		"pageTitle":       "Hello Jade",
		"youAreUsingJade": true,
		"num1":            5,
		"num2":            2,
		"list":            []int{1, 2, 3, 4, 5},
	}
	content, err := Load("../res/verifyjade.jade")
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
func _TestParseJade(t *testing.T) {
	template, err := Load("../res/test.jade")
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

type testTemplateData struct {
	PageTitle       string
	YouAreUsingJade bool
	Person          testSubTemplateData
	Children        []string
}

type testSubTemplateData struct {
	Name string
	Age  int
}

func (this *testTemplateData) Hello() string {
	return "Hello GoJade"
}

// Test parsing and evaluating jade.
func _TestDataMap(t *testing.T) {
	template, err := Load("../res/test.jade")
	if err != nil {
		t.Error(err)
		return
	}
	buf := new(bytes.Buffer)
	//test using map
	data := map[string]interface{}{
		"PageTitle":       "Hello Jade",
		"YouAreUsingJade": true,
		"Person":          testSubTemplateData{"ben", 32},
		"Children":        []string{"sue", "mike", "alex"},
	}
	result := Parse(template)
	fmt.Println(result.Root)
	renderJade(buf, template, data)
	fmt.Println(buf.String())
}

// Test parsing and evaluating jade.
func _TestDataStruct(t *testing.T) {
	template, err := Load("../res/test.jade")
	if err != nil {
		t.Error(err)
		return
	}
	buf := new(bytes.Buffer)
	data := &testTemplateData{"Hello Jade", true, testSubTemplateData{"ben", 32}, []string{"sue", "mike", "alex"}}
	renderJade(buf, template, data)
	//fmt.Println(buf.String(), j.Log)
}

// Test parsing jade extends functions.
func _TestParseExtends(t *testing.T) {
	template, err := Load("../res/extends/index.jade")
	if err != nil {
		t.Error(err)
		return
	}
	result := Parse(template)
	fmt.Println(result.Root, result.Extends)
}

// store jade and html result for comparison.
type verifyItem struct {
	name string
	jade string
	html string
}

// Load a test file
func Load(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Save a test result
func Save(filename string, data []byte) {
	ioutil.WriteFile(filename, data, os.ModePerm)
}

// RenderJade shortcut function.
func renderJade(buf *bytes.Buffer, template string, data interface{}) *EvalJade {
	eval := NewEvalJade(buf)
	eval.SetData(data)
	eval.RenderString(template)
	return eval
}

// evaluate jade
func evaltest(t *testing.T, i int, item *verifyItem, data map[string]interface{}) {
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
