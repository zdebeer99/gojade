package jadeparser

import (
	"bytes"
	"fmt"
)

var keywords []string = []string{"if", "else", "unless", "case", "when", "default", "each", "mixin", "block", "extends", "include"}

var selfClosingTags = []string{
	"meta",
	"img",
	"link",
	"input",
	"source",
	"area",
	"base",
	"col",
	"br",
	"hr",
}

// branchStartStatement decide what to do at the start of a jade statement.
func (this *parser) parseIndent() bool {
	var lvl int
	r := this.scan.Peek()
	if r == ' ' || r == '\t' {
		var err error
		lvl, err = this.skipIndent(r)
		if err != nil {
			this.error(err)
			return false
		}
		if lvl < 0 {
			return false
		}
	} else {
		if this.scan.AcceptNewLine() {
			return false
		}
		if this.indent.indentType == 0 {
			lvl = 0
		} else {
			var err error
			lvl, err = this.indent.setCurr(0)
			if err != nil {
				this.error(err)
				return false
			}
		}
	}
	tag := NewHtmlTagToken("div")
	depth := this.curr.Depth()

	switch {
	case lvl == depth:
		this.curr = this.add(tag)
	case lvl > depth:
		this.curr = this.stack(tag)
	case lvl < depth:
		this.unstack(this.curr.Depth() - lvl)
		this.curr = this.add(tag)
	}
	return true
}

func (this *parser) parseHtmlTag() stateFn {
	var tag *HtmlTagToken
	tag, ok := this.curr.Value.(*HtmlTagToken)
	if !ok {
		tag = NewHtmlTagToken("div")
		this.curr = this.add(tag)
	}
	tagname := this.commit()
	if tagname == "doctype" {
		this.replace(NewHtmlDocTypeToken())
		return branchHtmlDocType
	}
	if state := this.parseKeyword(tagname); state != nil {
		return state
	}
	if InSlice(selfClosingTags, tagname) {
		tag.SelfClosing = true
	}
	tag.TagName = tagname
	return branchAfterHtmlTag
}

func (this *parser) parseBlockExpansion() stateFn {
	this.curr = this.stack(NewHtmlTagToken("div"))
	return this.parseHtmlTag()
}

func (this *parser) parseAndAttribute() {
	test := this.commit()
	if test != "&attributes(" {
		this.error("Expecting &attributes() found " + test)
		return
	}
	if tag, ok := this.curr.Value.(*HtmlTagToken); ok {
		expr := this.parseExpression()
		if expr != nil {
			fn := NewFuncToken(attributesFunc)
			fn.AddArgument(expr)
			tag.AddAttribute(this.newNode(fn))
		}
	}
	for {
		r := this.scan.Next()
		switch r {
		case ')':
			this.ignore()
			return
		case ' ':
			this.ignore()
			continue
		}
		this.error("Unexpected character %q in &attributes()", r)
	}
}

func (this *parser) parseAttribute() stateFn {
	if this.commit() != "(" {
		this.error("Jade Attributes must start with a '('")
		return nil
	}
	scan := this.scan
	tag, ok := this.curr.Value.(*HtmlTagToken)
	if !ok {
		this.error("Expecting current node to be of type HtmlTag. Node Type: %q", this.curr.Value.String())
		return nil
	}
	var word string
	var mode int
attributes:
	for {
		if scan.ScanHtmlWord() {
			if mode == 1 {
				tag.AddAttribute(this.newNode(NewTextToken(word)))
			}
			word = this.commit()
			mode = 1
		}
		if scan.Prefix("!=") {
			this.ignore()
			expr := this.parseExpression()
			switch mode {
			case 0:
				tag.AddAttribute(expr)
			case 1:
				tag.AddKeyValue(word, expr)
			}
			mode = 0
			continue attributes
		}
		switch scan.Next() {
		case ')':
			this.ignore()
			if mode == 1 {
				tag.AddAttribute(this.newNode(NewTextToken(word)))
			}
			mode = 0
			word = ""
			return branchAttributeEnd
		case ' ', '\n', '\r':
			this.ignore()
			continue attributes
		case ',':
			this.ignore()
			if mode == 1 {
				tag.AddAttribute(this.newNode(NewTextToken(word)))
			}
			mode = 0
			word = ""
			continue attributes
		case '=':
			this.ignore()
			fnescapeHtml := NewFuncToken(escapeHtmlFunc)
			fnescapeHtml.AddArgument(this.parseExpression())
			switch mode {
			case 0:
				tag.AddAttribute(this.newNode(fnescapeHtml))
			case 1:
				tag.AddKeyValue(word, this.newNode(fnescapeHtml))
			}
			mode = 0
			continue attributes
		}
		scan.Backup()
		expr := this.parseExpression()
		if expr == nil {
			this.error("Invalid characters in attributes.")
			return nil
		} else {
			tag.AddAttribute(expr)
		}
	}
}

func (this *parser) parseComment() stateFn {
	comment := NewCommentToken(this.commit())
	this.replace(comment)
	if this.scan.AcceptNewLine() {
		return branchMultiLineContent
	}
	return branchContent
}

func (this *parser) parseKeyword(keyword string) stateFn {
	if InSlice(keywords, keyword) {
		fnkeywork := NewFuncToken(keyword)
		var arg *TreeNode
		var blockExpandsion bool

		//Handle Keywords that allow block expansion after the keyword.
		switch keyword {
		case "when", "default":
			if endo := this.scan.RunTo(":\n"); endo != -1 {
				blockExpandsion = endo == ':'
				exprtxt := this.scan.Commit()
				exprtxt = exprtxt[:len(exprtxt)-1]
				if len(exprtxt) > 0 {
					arg = this.parseExpressionFrom(exprtxt)
				} else {
					arg = nil
				}
			} else {
				arg = this.parseExpression()
			}
		case "extends":
			arg = this.getContent()
			txttoken, ok := arg.Value.(*TextToken)
			if !ok {
				this.error("Expecting a filename after the keyword '%s'", keyword)
				return branchEnd
			}
			this.extends = txttoken.Text
		case "include":
			arg = this.getContent()
			_, ok := arg.Value.(*TextToken)
			if !ok {
				this.error("Expecting a filename after the keyword '%s'", keyword)
				return branchEnd
			}
		default:
			arg = this.parseExpression()
			//handle 'else if'
			if arg != nil {
				if fn, ok := arg.Value.(*FuncToken); ok && fn.Name == "if" {
					if fnkeywork.Name == "else" {
						arg = this.parseExpression()
					} else {
						this.error("Expecting else before if, in a else if statement. found: %q", fnkeywork.Name)
					}
				}
			}
		}

		//link arguments to keyword
		if this.err != nil {
			return branchEnd
		}
		if arg != nil {
			fnkeywork.AddArgument(arg)
		}
		this.replace(fnkeywork)

		//Validation and Special cases
		switch keyword {
		case "when", "default":
			if fn, ok := this.curr.parent.Value.(*FuncToken); !(ok && fn.Name == "case") {
				this.error("Invalid %q, Expecting 'Case' statement before %q", keyword, keyword)
				return branchEnd
			}
			if blockExpandsion {
				this.scan.SkipSpaces()
				this.ignore()
				if this.scan.ScanWord() {
					return this.parseBlockExpansion()
				}
			}
		case "else":
			//verify the else have a if before it.
			cnt1 := len(this.curr.parent.items)
			if cnt1 > 1 {
				iffunc, ok := this.curr.parent.items[cnt1-2].Value.(*FuncToken)
				if !ok || !InSlice([]string{"if", "unless", "else"}, iffunc.Name) {
					this.error("Else statment must have a if or unless statement before it.")
					break
				}
			} else {
				this.error("Else statment must have a if statement before it. Make sure the indentation is correct.")
				return branchEnd
			}
		case "each":
			this.scan.SkipSpaces()
			if this.scan.Next() == ',' {
				this.ignore()
				fnkeywork.AddArgument(this.parseExpression())
			} else {
				this.scan.Backup()
				fnkeywork.AddArgument(this.newNode(NewEmptyToken()))
			}
			instatement := this.parseExpression()
			if fnin, ok := instatement.Value.(*FuncToken); ok && fnin.Name == "in" {
				fnkeywork.AddArgument(this.parseExpression())
			} else {
				this.error("Expecting 'in' keyword after 'each' keyword.")
				return branchEnd
			}
		case "block":
			if arg != nil {
				txttoken, ok := arg.Value.(*FuncToken)
				if !ok || !txttoken.IsIdentity {
					this.error("Expecting block name. found %s ", arg.String())
					return branchEnd
				}
				this.blocks[txttoken.Name] = this.curr
			}
		case "mixin":
			if len(fnkeywork.Arguments) != 1 {
				this.error("mixin missing name.")
				return branchEnd
			}

			mixinNode := fnkeywork.Arguments[0]
			if mixin, ok := mixinNode.Value.(*FuncToken); ok {
				this.mixins[mixin.Name] = this.curr
			}
		default:
		}
		return branchStartStatement
	}
	return nil
}

func (this *parser) parseHtmlTagClass() stateFn {
	scan := this.scan
	tag, ok := this.curr.Value.(*HtmlTagToken)
	if !ok {
		this.error("Expecting a Html Tag before the . operator. Current Tag: %s", this.curr.String())
	}
	if scan.ScanHtmlWord() {
		tag.SetClass(this.newNode(NewTextToken(scan.Commit())))
		return branchAfterHtmlTag
	}
	this.error("Unexpected char, expecting a word after #.")
	return nil
}

func (this *parser) parseHtmlTagId() stateFn {
	scan := this.scan
	tag, ok := this.curr.Value.(*HtmlTagToken)
	if !ok {
		this.error("Expecting a Html Tag before the # operator. Current Tag: %s", this.curr.String())
	}
	scan.Ignore()
	if scan.ScanHtmlWord() {
		tag.AddKeyValue("id", this.newNode(NewTextToken(scan.Commit())))
		return branchAfterHtmlTag
	}
	this.error("Unexpected char, expecting a word after #.")
	return nil
}

func (this *parser) parseContent() {
	this.curr.AddElement(this.getContent())
}

func (this *parser) parseMultilineContent() {
	scan := this.scan
	var buf bytes.Buffer
	initlvl := this.getIndent()
	//if the next line is only a next line skip it.
	for initlvl == -1 {
		buf.WriteRune('\n')
		initlvl = this.getIndent()
	}
	contentIndent := this.indent.curr
	//ignore indentation but keep extra spaces.
	scan.SetPosition(scan.StartPosition() + initlvl*this.indent.indentType)
	scan.Ignore()
	start := scan.Position()
	node := this.newNode(NewEmptyToken())
	cnt := 0
	for {
		cnt++
		//		if cnt > 1000000 {
		//			return
		//		}
		if scan.IsEOF() {
			node.AddElement(this.newNode(NewTextToken(buf.String())))
			break
		}
		if scan.AcceptNewLine() {
			start = scan.Position()
			lvl := this.getIndent()
			if lvl <= contentIndent && lvl > -1 {
				node.AddElement(this.newNode(NewTextToken(buf.String())))
				scan.SetStartPosition(start)
				scan.SetPosition(start)
				break
			}
			//ignore indentation but keep extra spaces.
			buf.WriteRune('\n')
			if lvl > -1 {
				scan.SetPosition(start + initlvl*this.indent.indentType)
				scan.Ignore()
			} else {
				scan.SetPosition(start + 1)
			}
			continue
		}
		if this.scan.Prefix("#{") || this.scan.Prefix("!{") {
			escape := this.commit() == "#{"
			node.AddElement(this.newNode(NewTextToken(buf.String())))
			buf.Reset()
			expr := this.parseExpression()
			if expr != nil {
				if escape {
					fnescapeHtml := NewFuncToken(escapeHtmlFunc)
					fnescapeHtml.AddArgument(expr)
					node.AddElement(this.newNode(fnescapeHtml))
				} else {
					node.AddElement(expr)
				}
			}
			this.scan.SkipSpaces()
			if this.scan.Next() == '}' {
				this.ignore()
				continue
			}
			this.error("Invalid end. Expecting '}' character.")
			break
		}
		r := scan.Next()
		buf.WriteRune(r)
	}
	scan.Ignore()
	if len(node.items) == 1 {
		node = node.items[0]
		node.parent = nil
	}
	this.curr.AddElement(node)
}

func (this *parser) parseExpression() *TreeNode {
	node := this.curr
	this.curr = this.newNode(NewGroupToken(""))
	state := branchExpressionValuePart
	for state != nil {
		state = state(this)
	}
	expr := this.curr.Root()
	if len(expr.items) == 0 {
		expr = nil
	} else if len(expr.items) == 1 {
		expr = expr.items[0]
		expr.parent = nil
	}
	this.curr = node
	return expr
}

func (this *parser) parseExpressionFrom(expr string) *TreeNode {
	node, err := ParseExpression(expr)
	if err != nil {
		this.error("Expression Error %s", err.Error())
		return nil
	}
	if len(node.items) == 0 {
		node = nil
	} else if len(node.items) == 1 {
		node = node.items[0]
		node.parent = nil
	}
	return node
}

//parseArguments parse function arguments.
func (this *parser) parseArguments() {
	fn, ok := this.curr.Value.(*FuncToken)
	if !ok {
		this.error("Expecting function before arguments.")
		return
	}
loop:
	for {
		fn.AddArgument(this.parseExpression())
	skippart:
		switch this.scan.Next() {
		case ',':
			this.ignore()
			continue loop
		case ')':
			this.ignore()
			break loop
		case ' ', '\r', '\n':
			this.ignore()
			goto skippart
		}
		this.error("Unexpected character in Arguments." + this.commit())
		break loop
	}
}

func (this *parser) parseUnbufferedCode() {
	var expr *TreeNode
loop1:
	for {
		expr = this.parseExpression()
		if expr == nil {
			return
		}
		if identity, ok := expr.Value.(*FuncToken); ok {
			if identity.Name == "var" {
				fn := NewFuncToken("var")
				expr = this.newNode(fn)
				kv := this.parseExpression()
				if innerfn, ok := kv.Value.(*OperatorToken); ok && innerfn.Operator == "=" {
					if len(kv.items) == 2 {
						fn.AddArgument(kv.items[0])
						fn.AddArgument(kv.items[1])
					} else {
						this.error("Invalid var statement, expecting argument name=value found: " + kv.String())
						return
					}
				} else {
					this.error("Invalid var statement.")
					return
				}
			}
		}
		this.replace(expr.Value)
	loop2:
		for {
			if this.scan.IsNewLine() || this.scan.IsEOF() {
				this.ignore()
				break loop1
			}
			r := this.scan.Next()
			switch r {
			case ' ':
				this.ignore()
				continue loop2
			case ';':
				this.ignore()
				continue loop1
			}
			this.scan.Backup()
			break loop1
		}
	}
}

func (this *parser) getIndent() int {
	scan := this.scan

	var indentchar string = " "
	if this.indent.indentType == 0 {
		switch scan.Peek() {
		case ' ':
			indentchar = " "
		case '\t':
			indentchar = "\t"
			this.indent.setType(1)
		default:
			if scan.AcceptNewLine() {
				return -1
			}
			return 0
		}
	} else if this.indent.indentType == 1 {
		indentchar = "\t"
	}
	cnt := this.scan.AcceptRun(indentchar)
	if this.indent.indentType == 0 {
		if cnt > 1 {
			this.indent.setType(cnt)
		} else {
			this.error("Space indentation requires atleast 2 spaces or more.")
		}
	}
	if scan.AcceptNewLine() {
		return -1
	}
	lvl, err := this.indent.charCountToLevel(cnt)
	if err != nil {
		this.error(err)
	}
	return lvl
}

//TODO: Support tab indentation
func (this *parser) skipIndent(r rune) (int, error) {
	indent := this.indent
	var cnt int
	if indent.indentType > 1 {
		if r != ' ' {
			return -1, fmt.Errorf("Invalid indent, Indent set to space, found %c. Cannot mix indentation", r)
		}
		cnt = this.scan.AcceptRun(" ")
	}
	if indent.indentType == 1 {
		if r != '\t' {
			return -1, fmt.Errorf("Invalid indent, Indent set to tab, found %c. Cannot mix indentation", r)
		}
		cnt = this.scan.AcceptRun("\t")
	}
	if indent.indentType == 0 {
		//Initialize the indent for the first time.
		switch r {
		case ' ':
			cnt = this.scan.AcceptRun(" ")
			if cnt == 1 {
				return -1, fmt.Errorf("Invalid indent, Indent spaces must have at least two spaces.")
			} else {
				indent.setType(cnt)
			}
		case '\t':
			cnt = this.scan.AcceptRun("\t")
			indent.setType(1)
		default:
			return 0, nil
		}
	}
	if this.scan.AcceptNewLine() {
		return -1, nil
	}
	lvl, err := indent.setCurr(cnt)
	if err != nil {
		return -1, err
	}
	if lvl-indent.prev > 1 {
		return -1, fmt.Errorf("Invalid indent, indented to much. prev indentation %v", indent.prev)
	}
	//Move the cursor start position to after the indent.
	this.scan.MoveStart(indent.indentType * indent.curr)
	return lvl, nil
}

func (this *parser) getContent() *TreeNode {
	var buf, code bytes.Buffer
	var node = this.newNode(NewEmptyToken())
	var inCode, escape bool
	for {
		this.ignore()
		if this.scan.AcceptNewLine() || this.scan.IsEOF() {
			if inCode {
				this.error("Missing closing handlebar '}'")
			}
			this.ignore()
			node.AddElement(this.newNode(NewTextToken(buf.String())))
			break
		}
		if this.scan.Prefix("#{") || this.scan.Prefix("!{") {
			codePrefix := this.commit()
			escape = codePrefix == "#{"
			if inCode {
				this.error("Not expecting " + codePrefix + " inside of a code section.")
				break
			}
			node.AddElement(this.newNode(NewTextToken(buf.String())))
			buf.Reset()
			inCode = true
		}
		r := this.scan.Next()
		if r == '}' {
			expr := this.parseExpressionFrom(code.String())
			if expr != nil {
				if escape {
					fnescapeHtml := NewFuncToken(escapeHtmlFunc)
					fnescapeHtml.AddArgument(expr)
					node.AddElement(this.newNode(fnescapeHtml))
				} else {
					node.AddElement(expr)
				}
			}
			escape = false
			inCode = false
			code.Reset()
			continue
		}
		if inCode {
			code.WriteRune(r)
		} else {
			buf.WriteRune(r)
		}
	}
	if len(node.items) == 1 {
		node = node.items[0]
		node.parent = nil
	}
	this.ignore()
	return node
}

func InSlice(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
