package jadeparser

// Expression Pump, iterate trough an expression.
func (this *parser) pumpJade(startbranch stateFn) {
	this.state = startbranch
	for this.state != nil {
		this.state = this.state(this)
	}
	endo := this.commit()
	if len(endo) > 0 || !this.scan.IsEOF() {
		if !this.scan.IsEOF() && len(endo) == 0 {
			for i := 0; i < 10; i++ {
				this.scan.Next()
				if this.scan.IsEOF() {
					break
				}
			}
			endo = this.commit()
		}
		if this.err == nil {
			this.error("Unexpected end of expression. '%s...' not parsed. ", endo)
		}
	}
}

/*
parse expressions
[value part][operator part] repeat
*/

//
func branchExpressionValuePart(this *parser) stateFn {
	scan := this.scan
	scan.SkipSpaces()
	if scan.IsEOF() {
		return nil
	}
	if scan.ScanNumber() {
		this.add(NewNumberToken(scan.Commit()))
		return branchExpressionOperatorPart
	}
	if scan.ScanWord() {
		return branchExpressionAfterWord
	}
	if scan.AcceptNewLine() {
		return nil
	}
	r := scan.Next()
	switch r {
	case '"', '\'':
		scan.Backup()
		txt := this.parseText()
		this.add(NewTextToken(txt))
		return branchExpressionOperatorPart
	case '(':
		this.parseOpenBracket()
		return branchExpressionValuePart
	case '[':
		this.curr.AddElement(this.parseArray())
		return branchExpressionOperatorPart
	case '{':
		this.curr.AddElement(this.parseMap())
		return branchExpressionOperatorPart
	case '!': //Not Operator
		this.parseNot()
		return branchExpressionOperatorPart
	}
	//ignore unknown characters and leave them for external parser.
	scan.Backup()
	return nil
}

func branchExpressionAfterWord(this *parser) stateFn {
	this.parseIdentity()
	return branchExpressionOperatorPart
}

//
func branchExpressionOperatorPart(this *parser) stateFn {
	scan := this.scan
	scan.SkipSpaces()

	if scan.IsEOF() {
		return nil
	}
	if this.AcceptOperator() {
		this.parseOperator()
		return branchExpressionValuePart
	}
	if scan.Accept("=") {
		this.parseLRFunc()
		this.curr = this.add(NewGroupToken(""))
		return branchExpressionValuePart
	}
	switch scan.Next() {
	case ')':
		//this.ignore()
		return this.parseCloseBracket()
	}
	scan.Backup()
	scan.Ignore()
	//panic("No Operator Found.")
	return nil
}

func branchStartStatement(this *parser) stateFn {
	this.state = branchStartStatement
	scan := this.scan

	if scan.IsEOF() || this.err != nil {
		return nil
	}
	if !this.parseIndent() {
		return branchStartStatement
	}
	if scan.Prefix("//-") || scan.Prefix("//") {
		return this.parseComment()
	}

	switch scan.Next() {
	case '|':
		scan.Ignore()
		this.replaceNode(this.getContent())
		return branchStartStatement
	case '<':
		scan.Backup()
		this.replaceNode(this.getContent())
		return branchStartStatement
	case '-':
		return branchCode
	case '+':
		return branchCode
	}
	scan.Backup()

	//Handle Buffered code without a tag.
	if scan.Prefix("=") || scan.Prefix("!=") {
		this.curr.Value = NewEmptyToken()
		return branchCode
	}
	if scan.ScanHtmlWord() {
		return this.parseHtmlTag()
	}
	return branchAfterHtmlTag(this)
}

func branchStartStatementDot(this *parser) stateFn {
	this.ignore()
	if this.scan.Peek() == ' ' {
		this.warning("Space found after Block Content character. '.'")
		this.scan.SkipSpaces()
	}
	if this.scan.AcceptNewLine() {
		return branchMultiLineContent
	}
	return this.parseHtmlTagClass()
}

func branchAfterHtmlTag(this *parser) stateFn {
	scan := this.scan
	if scan.AcceptNewLine() {
		return branchStartStatement
	}
	if scan.Prefix("!=") {
		return branchCode
	}
	switch scan.Next() {
	case '(':
		return this.parseAttribute()
	case '.':
		return branchStartStatementDot
	case '#':
		return this.parseHtmlTagId()
	case ' ':
		this.ignore()
		return branchContent
	case '=':
		return branchCode
	case ':':
		scan.SkipSpaces()
		scan.Ignore()
		if scan.ScanWord() {
			return this.parseBlockExpansion()
		}
	case '/':
		tag, ok := this.curr.Value.(*HtmlTagToken)
		if ok {
			tag.SelfClosing = true
			if scan.AcceptNewLine() || scan.IsEOF() {
				this.ignore()
				return branchStartStatement
			} else {
				this.error("After Tag. Self closing tag cannot have content.")
			}
		} else {
			this.error("After Tag. Expecting '/' after a tag")
			return nil
		}
	}
	if scan.IsEOF() {
		return nil
	}
	this.error("After Tag. Invalid char after tag name.")
	return nil
}

func branchContent(this *parser) stateFn {
	this.parseContent()
	return branchStartStatement
}

func branchMultiLineContent(this *parser) stateFn {
	this.parseMultilineContent()
	return branchStartStatement
}

func branchHtmlDocType(this *parser) stateFn {
	token, ok := this.curr.Value.(*HtmlDocTypeToken)
	if !ok {
		this.error("Expecting the current node to be of type doctype.")
		return nil
	}
	txtnode := this.getContent()
	if txt, ok := txtnode.Value.(*TextToken); ok {
		token.Attributes = append(token.Attributes, txt.Text)
		return branchStartStatement
	}
	this.error("Expecting doctype type argument.")
	return nil
}

func branchAttributeEnd(this *parser) stateFn {
	scan := this.scan
	if scan.IsEOF() {
		return nil
	}
	if scan.Prefix(".\n") || scan.Prefix(".\r\n") {
		this.ignore()
		return branchMultiLineContent
	}
	if scan.AcceptNewLine() {
		return branchStartStatement
	}
	if scan.Prefix("!=") {
		return branchCode
	}
	if scan.Prefix("&attributes(") {
		this.parseAndAttribute()
		return branchAttributeEnd
	}
	r := scan.Next()
	switch r {
	case ' ':
		return branchContent
	case '/':
		this.ignore()
		tag, ok := this.curr.Value.(*HtmlTagToken)
		if ok {
			tag.SelfClosing = true
			if scan.AcceptNewLine() || scan.IsEOF() {
				return branchStartStatement
			} else {
				this.error("After Attribute. Self closing tag cannot have content.")
			}
		} else {
			this.error("After Attribute. Expecting '/' after a tag")
			return nil
		}
	case '=':
		return branchCode
	case ':':
		scan.SkipSpaces()
		scan.Ignore()
		if scan.ScanWord() {
			return this.parseBlockExpansion()
		}
	}
	this.error("Unexpected character after Attributes %q %v", string(r), r)
	return nil
}

func branchCode(this *parser) stateFn {
	switch this.commit() {
	case "=":
		fnescapeHtml := NewFuncToken(escapeHtmlFunc)
		fnescapeHtml.AddArgument(this.parseExpression())
		this.add(fnescapeHtml)
	case "!=":
		this.curr.AddElement(this.parseExpression())
	case "-":
		this.parseUnbufferedCode()
	case "+":
		getMixin := NewFuncToken(jadeMixinFunc)
		getMixin.AddArgument(this.parseExpression())
		this.replace(getMixin)
	default:
		this.error("Unexpected char")
	}
	return branchStartStatement
}

func branchEnd(this *parser) stateFn {
	return nil
}
