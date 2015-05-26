package jadeparser

import (
	"io"
	"strings"
)

type jadewriter struct {
	wr          io.Writer
	template    *EvalJade
	lastnewline bool
}

func (this *jadewriter) write(txt string) (int, error) {
	this.lastnewline = false
	if this.template.Beautify {
		this.lastnewline = strings.HasSuffix(txt, "\n")
	}
	return this.wr.Write([]byte(txt))
}

func (this *jadewriter) writeValue(val interface{}) (int, error) {
	return this.write(ObjToString(val))
}

func (this *jadewriter) router(node *TreeNode) {
	this.template.router(node)
}

func (this *jadewriter) HtmlDocType(doctype *HtmlDocTypeToken) {
	arg := strings.Trim(doctype.Attributes[0], " ")
	this.template.doctype = arg
	switch arg {
	case "html":
		this.write("<!DOCTYPE html>")
	case "xml":
		this.write(`<?xml version="1.0" encoding="utf-8" ?>`)
	case "transitional":
		this.write(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">`)
	case "strict":
		this.write(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">`)
	case "frameset":
		this.write(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Frameset//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-frameset.dtd">`)
	case "1.1":
		this.write(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">`)
	case "basic":
		this.write(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML Basic 1.1//EN" "http://www.w3.org/TR/xhtml-basic/xhtml-basic11.dtd">`)
	case "mobile":
		this.write(`<!DOCTYPE html PUBLIC "-//WAPFORUM//DTD XHTML Mobile 1.2//EN" "http://www.openmobilealliance.org/tech/DTD/xhtml-mobile12.dtd">`)
	default:
		panic("Invalid doctype '" + arg + "'")
	}
	this.beautifyNewLine()
}

func (this *jadewriter) HtmlTag(node *TreeNode, tag *HtmlTagToken) {
	indent := this.beautifyIndent(node)
	this.write(indent + "<")
	this.write(tag.TagName)
	for _, attr := range tag.Attributes {
		this.AttributeItem(attr)
	}
	if tag.SelfClosing {
		if this.template.doctype == "html" {
			this.write(">")
		} else {
			this.write("/>")
		}
		this.beautifyNewLine()
		return
	} else {
		this.write(">")
		if len(node.items) > 1 {
			this.beautifyNewLine()
		}
		this.template.evalContent(node)
		if len(node.items) > 1 {
			this.beautifyNewLine()
		} else {
			indent = ""
		}
		this.write(indent + "</")
		this.write(tag.TagName)
		this.write(">")
		this.beautifyNewLine()
	}
}

func (this *jadewriter) Comment(node *TreeNode, comment *CommentToken) {
	indent := this.beautifyIndent(node)
	if comment.CommentType == "//" {
		this.write(indent + "<!--")
		for _, content := range node.items {
			this.router(content)
		}
		this.write("-->")
	}
}

func (this *jadewriter) AttributeItem(node *TreeNode) {
	switch val := node.Value.(type) {
	case *KeyValueToken:
		this.KeyValueAttribute(val)
	case *TextToken:
		if this.template.doctype == "html" {
			this.write(" ")
			this.write(val.Text)
		} else {
			this.write(" ")
			this.write(val.Text + "=\"" + val.Text + "\"")
		}
	default:
		this.write(" ")
		this.router(node)
	}
}

func (this *jadewriter) Group(node *TreeNode) {
	group, ok := node.Value.(*GroupToken)
	if !ok {
		panic("Expecting a group token.")
	}
	var start, end, del string
	del = ","
	switch group.GroupType {
	case "()":
		start = "("
		end = ")"
		del = " "
	case "[]":
		start = ""
		end = ""
		del = " "
	case "{}":
		start = "{"
		end = "}"
	default:
		start = ""
		end = ""
		del = " "
	}
	this.GroupWriter(node, start, end, del)

}

func (this *jadewriter) GroupWriter(node *TreeNode, start, end, del string) {
	this.write(start)
	cnt := len(node.items)
	for i, val1 := range node.items {
		this.router(val1)
		if i < cnt-1 {
			this.write(del)
		}
	}
	this.write(end)
}

//KeyValueAttribute Write key value pairs for attributes.
func (this *jadewriter) KeyValueAttribute(keyvalue *KeyValueToken) {
	valueNode, escape := stripEscapeHtml(keyvalue.Value)
	value := this.template.getValue(valueNode).Interface()
	switch val := value.(type) {
	case bool:
		if val {
			if this.template.doctype == "html" {
				this.write(" ")
				this.write(keyvalue.Key)
			} else {
				this.write(" ")
				this.write(keyvalue.Key + "=\"" + keyvalue.Key + "\"")
			}
		}
		return
	}
	this.write(" ")
	this.write(keyvalue.Key)
	this.write("=\"")
	switch strings.ToLower(keyvalue.Key) {
	case "style":
		this.styleAttribute(valueNode, escape)
	case "class":
		this.classAttribute(valueNode, escape)
	default:
		if escape {
			this.write(this.template.escapeHtml(valueNode))
		} else {
			this.writeValue(value)
		}
	}
	this.write("\"")
}

func (this *jadewriter) styleAttribute(valueNode *TreeNode, escape bool) {
	del := ""
	for _, val1 := range valueNode.items {
		this.write(del)
		switch val2 := val1.Value.(type) {
		case *KeyValueToken:
			this.write(val2.Key)
			this.write(":")
			this.router(val2.Value)
		default:
			panic("Expecting Key Value Pair in style attribute.")
		}
		del = ";"
	}
}

func (this *jadewriter) classAttribute(node *TreeNode, escape bool) {
	this.router(node)
}

//KeyValueToken write a key value token.
func (this *jadewriter) KeyValueToken(keyvalue *KeyValueToken) {
	this.write(keyvalue.Key)
	this.write("=\"")
	this.router(keyvalue.Value)
	this.write("\"")
}

func (this *jadewriter) lrfunc(node *TreeNode, token *LRFuncToken) {
	panic("Not Implemented")
}

func (this *jadewriter) stdfunc(node *TreeNode, token *FuncToken) {
	this.write(ObjToString(this.template.evalFunc(node, token)))
}

func (this *jadewriter) text(token *TextToken) {
	this.write(token.Text)
}

func (this *jadewriter) jadecase(node *TreeNode, fn *FuncToken) {
	var caseval interface{}
	if len(fn.Arguments) == 1 {
		caseval = this.template.getValue(fn.Arguments[0]).Interface()
	}

	var whenprev bool
	for _, whenNode := range node.items {
		when, ok := whenNode.Value.(*FuncToken)
		if !ok {
			panic("Expecting a case or default statement.")
		}
		if when.Name == "when" {
			var whentrue bool
			if caseval == nil {
				whentrue = this.template.getBool(when.Arguments[0])
			} else {
				var err error
				whentrue, err = eq(caseval, this.template.getValue(when.Arguments[0]).Interface())
				if err != nil {
					panic("case: Error on When value. " + err.Error())
				}
			}
			if whentrue || whenprev {
				if len(whenNode.items) > 0 {
					this.template.evalContent(whenNode)
					break
				} else {
					whenprev = true
				}
			}
		}
		if when.Name == "default" {
			this.template.evalContent(whenNode)
		}
	}
}

//************************************
//Helper Functions
//************************************

func stripEscapeHtml(node *TreeNode) (*TreeNode, bool) {
	if fn, ok := node.Value.(*FuncToken); ok && fn.Name == escapeHtmlFunc {
		return fn.Arguments[0], true
	}
	return node, false
}

func (this *jadewriter) beautifyNewLine() {
	if this.template.Beautify && !this.lastnewline {
		this.write("\n")
	}
	this.lastnewline = true
}

func (this *jadewriter) beautifyIndent(node *TreeNode) string {
	indent := ""
	if this.template.Beautify {
		lvl := node.Depth() - 1
		for i := 0; i < lvl; i++ {
			indent = indent + "  "
		}
	}
	return indent
}
