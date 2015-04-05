// GoJade Gin Example demonstrates how to integrate GoJade into Gin.
// The ginRenderJade struct at the bottom of this file is used to link
// GoJade into gin's HTML Renderer.
// Gin https://gin-gonic.github.io/gin/
package main

import (
	"github.com/zdebeer99/gojade"
	"net/http"
)

var jade *gojade.GoJade

//model, this struct is passed to the jade passed.
type pageModel struct {
	Title    string
	Name     string
	Age      int
	Children []string
}

// Home page handler.
func index(rw http.ResponseWriter, req *http.Request) {
	//init model data
	data := &pageModel{"GoJade http Example", "Glen Loveless", 32, []string{"Joe", "Marco", "Mimi"}}
	//render jade page. GoJade will automatically append .jade to a file name if no file extension is specified.
	jade.RenderFileW(rw, "index", data)
}

func main() {

	//Configure the gojade template engine
	jade = gojade.NewGoJade()
	jade.ViewPath = "./views"

	//configure http
	http.HandleFunc("/", index)
	http.ListenAndServe("localhost:3000", nil)
}
