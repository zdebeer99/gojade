package gojade

import (
	"io/ioutil"
	"os"
	"testing"
)

//Set testonly to a test number to only test that example.
var testonly int = -1

type verifyItem struct {
	name string
	jade string
	html string
}

func TestRenderFiles(t *testing.T) {
	jade := New()
	jade.ViewPath = "res"
	save("res/html/test.html", jade.RenderFile("test", nil).Bytes())
	//extends
	jade.ViewPath = "res/extends"
	save("res/html/extends.html", jade.RenderFile("index", nil).Bytes())

	//inheritance
	jade.ViewPath = "res/inheritance"
	data := map[string]interface{}{"title": "List of Pets", "pets": []string{"Dog", "Cat", "Bird"}}
	save("res/html/inheritance_a.html", jade.RenderFile("page-a", data).Bytes())
	save("res/html/inheritance_b.html", jade.RenderFile("page-b", data).Bytes())

	//includes
	jade.ViewPath = "res/includes"
	save("res/html/include.html", jade.RenderFile("index", data).Bytes())
	save("res/html/include_text.html", jade.RenderFile("index_text", data).Bytes())

}

func load(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func save(filename string, data []byte) {
	ioutil.WriteFile(filename, data, os.ModePerm)
}
