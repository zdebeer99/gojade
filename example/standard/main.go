// GoJade Standard Example demonstrates how to use gojade with go standard http.
package main

import (
	"github.com/zdebeer99/gojade"
	"net/http"
)

// create a global variable referencing GoJade
var jade *gojade.Engine

// model, this struct is passed to the jade view.
type pageModel struct {
	Title    string
	Name     string
	Age      int
	Children []string
}

// Home page handler.
func index(rw http.ResponseWriter, req *http.Request) {
	//init model data
	data := &pageModel{"GoJade http Example", "Glen Lovelace", 32, []string{"Joe", "Marco", "Mimi"}}
	//render jade page. GoJade will automatically append .jade to a file name if no file extension is specified.
	jade.RenderFileW(rw, "index", data)
}

func main() {

	//Configure the gojade template engine
	jade = gojade.New()
	jade.ViewPath = "./views"

	//configure http
	http.HandleFunc("/", index)
	http.ListenAndServe("localhost:3000", nil)
}
