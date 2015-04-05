// GoJade Gin Example demonstrates how to integrate GoJade into Gin.
// The ginRenderJade struct at the bottom of this file is used to link
// GoJade into gin's HTML Renderer.
// Gin https://gin-gonic.github.io/gin/
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/zdebeer99/gojade"
	"net/http"
)

//model, this struct is passed to the jade passed.
type pageModel struct {
	Title    string
	Name     string
	Age      int
	Children []string
}

// Home page handler.
func index(ctx *gin.Context) {
	//init model data
	data := &pageModel{"GoJade Gin Example", "Ben Loveless", 64, []string{"Mike", "Sue", "Jhon"}}
	//render jade page. GoJade will automatically append .jade to a file name is no file extension is specified.
	ctx.HTML(http.StatusOK, "index", data)
}

func main() {

	//Configure the gojade template engine
	jade := gojade.NewGoJade()
	jade.ViewPath = "./views"

	//Setup Gin
	r := gin.Default()
	//Link Gin to GoJade
	r.HTMLRender = &ginRenderJade{jade}
	//setup routes
	r.Static("/static", "./static")
	r.GET("/", index)
	//run http service
	r.Run(":3000")
}

// ginRenderJade Wraps GoJade and implements interface gin.render.Render
type ginRenderJade struct {
	Template *gojade.GoJade
}

// gin.render.Render Required method to render using Gin Context.Html()
func (this *ginRenderJade) Render(w http.ResponseWriter, code int, data ...interface{}) error {
	writeHeader(w, code, "text/html")
	file := data[0].(string)
	args := data[1]
	return this.Template.RenderFileW(w, file, args)
}

// Some helper function copied from gin.render
func writeHeader(w http.ResponseWriter, code int, contentType string) {
	w.Header().Set("Content-Type", contentType+"; charset=utf-8")
	w.WriteHeader(code)
}
