package jadeparser

import (
	"fmt"
	"github.com/zdebeer99/gojade/scanner"
	"strconv"
	"strings"
)

type stateFn func(*parser) stateFn

type parser struct {
	scan         *scanner.Scanner
	root         *TreeNode
	curr         *TreeNode
	err          error
	log          []string
	state        stateFn
	indent       *indent
	openBrackets int
	mixins       map[string]*TreeNode
	blocks       map[string]*TreeNode
	extends      string
}

type ParseResult struct {
	Root    *TreeNode
	Err     error
	Log     []string
	Mixins  map[string]*TreeNode
	Blocks  map[string]*TreeNode
	Extends string
}

func NewParser(input string) *parser {
	root := NewTreeNode(NewEmptyToken())
	return &parser{scanner.NewScanner(input), root, root, nil, make([]string, 0), nil, new(indent), 0, make(map[string]*TreeNode), make(map[string]*TreeNode),
		""}
}

func Parse(input string) *ParseResult {
	parse := NewParser(input)
	parse.pumpJade(branchStartStatement)
	return &ParseResult{parse.root, parse.err, parse.log, parse.mixins, parse.blocks, parse.extends}
}

func ParseExpression(input string) (*TreeNode, error) {
	parse := NewParser(input)
	parse.pumpJade(branchExpressionValuePart)
	return parse.root, parse.err
}

func (this *parser) Parse() {
	this.pumpJade(branchExpressionValuePart)
}

func (this *parser) getCurr() Token {
	if this.curr != nil {
		return this.curr.Value
	}
	return nil
}

func (this *parser) newNode(value Token) *TreeNode {
	node := NewTreeNode(value)
	node.Pos = this.scan.Position()
	return node
}

func (this *parser) add(token Token) *TreeNode {
	return this.curr.AddElement(this.newNode(token))
}

func (this *parser) push(token Token) *TreeNode {
	return this.curr.PushElement(this.newNode(token))
}

func (this *parser) lastNode() *TreeNode {
	return this.curr.LastElement()
}

func (this *parser) parentNode() *TreeNode {
	if this.curr == nil || this.curr.parent == nil {
		panic("Parent node is nil.")
	}
	return this.curr.Parent()
}

func (this *parser) stack(token Token) *TreeNode {
	lnode := this.lastNode()
	if lnode == nil {
		//panic("Cannot stack on node with no children")
		return this.add(token)
	}
	return lnode.AddElement(this.newNode(token))
}

// unstack sets the current node to some parent
func (this *parser) unstack(lvl int) {
	if this.curr.Parent() == nil {
		this.error("Invalid Indentation. Root Node")
	}
	for lvl > 0 {
		this.curr = this.curr.Parent()
		lvl--
	}
}

func (this *parser) replace(value Token) *TreeNode {
	this.curr.Pos = this.scan.Position()
	this.curr.Value = value
	return this.curr
}

func (this *parser) replaceNode(node *TreeNode) *TreeNode {
	this.curr = this.curr.ReplaceNode(node)
	this.curr.Pos = this.scan.Position()
	return this.curr
}

func (this *parser) error(err interface{}, a ...interface{}) {
	var errortxt string
	if val, ok := err.(error); ok {
		errortxt = val.Error()
	} else {
		if len(a) > 0 {
			errortxt = fmt.Sprintf(err.(string), a...)
		} else {
			errortxt = err.(string)
		}
	}
	lasttoken := this.commit()
	if len(lasttoken) < 10 {
		for i := len(lasttoken); i < 10 && !this.scan.IsEOF(); i++ {
			this.scan.Next()
		}
		lasttoken = lasttoken + this.commit()
	}
	debug := fmt.Errorf("Line: %v, near %q, Error: %s", this.scan.LineNumber(), lasttoken, errortxt)
	this.add(NewErrorToken(debug.Error()))
	this.err = debug
}

func (this *parser) warning(warning interface{}, a ...interface{}) {
	var warningtxt string
	if val, ok := warning.(error); ok {
		warningtxt = val.Error()
	} else {
		if len(a) > 0 {
			warningtxt = fmt.Sprintf(warning.(string), a...)
		} else {
			warningtxt = warning.(string)
		}
	}
	debug := fmt.Sprintf("Line: %v, Warning: %s", this.scan.LineNumber(), warningtxt)
	this.log = append(this.log, debug)
}

func (this *parser) commit() string {
	return this.scan.Commit()
}

func (this *parser) ignore() {
	this.scan.Ignore()
}

//parseOpenBracket
func (this *parser) parseOpenBracket() bool {
	this.openBrackets++
	this.curr = this.add(NewGroupToken("()"))
	this.commit()
	return true
}

//parseCloseBracket
func (this *parser) parseCloseBracket() stateFn {
	this.openBrackets--
	for {
		v1, ok := this.curr.Value.(*GroupToken)
		if ok && v1.GroupType == "()" {
			this.commit()
			this.curr = this.curr.Parent()
			return branchExpressionOperatorPart
		}
		if ok && v1.GroupType == "" {
			this.scan.Backup()
			return nil
		}
		if this.curr.Parent() == nil {
			this.error("Brackets not closed.")
			return nil
		}
		this.curr = this.curr.Parent()
	}
}

func (this *parser) parseIdentity() {
	scan := this.scan
	identityName := this.commit()
	if strings.ToLower(identityName) == "true" || strings.ToLower(identityName) == "false" {
		this.add(NewBoolToken(identityName))
		return
	}
	placeholder := NewIdentityToken("placeholder")
	placeholder.Next = NewIdentityToken(identityName)
	var chainedItem *FuncToken
	chainedItem = placeholder.Next
loop:
	for {
		r := scan.Next()
		switch r {
		case '.':
			this.ignore()
			if scan.ScanWord() {
				chainedItem.Next = NewIdentityToken(this.commit())
				chainedItem = chainedItem.Next
				continue loop
			} else {
				this.error("Expecting a word after '.' found: '%c'", r)
				break loop
			}
		case '[':
			this.ignore()
			if chainedItem.Index != nil {
				chainedItem.Next = NewIdentityToken("")
				chainedItem = chainedItem.Next
			}
			chainedItem.Index = this.parseExpression()
			if scan.Next() != ']' {
				this.error("Expecting ']' index closing bracket.")
				break loop
			}
			continue loop
		case '(':
			if len(chainedItem.Arguments) == 0 {
				chainedItem.IsIdentity = false
				this.parseFunctionArguments(chainedItem)
				continue loop
			} else {
				chainedItem.Next = NewFuncToken("attributes")
				chainedItem = chainedItem.Next
				this.parseFunctionArguments(chainedItem)
				continue loop
			}
		}
		scan.Backup()
		break loop
	}
	this.add(placeholder.Next)
}

func (this *parser) parseFunctionArguments(ftoken *FuncToken) {
	scan := this.scan
loop:
	for {
		expr := this.parseExpression()
		if expr != nil {
			ftoken.AddArgument(expr)
		}

		r := scan.Next()
		switch r {
		case ' ':
			scan.Ignore()
			continue loop
		case ',':
			scan.Ignore()
			continue loop
		case ')':
			//ftoken.AddArgument(this.curr.Root())
			//this.curr = currnode
			scan.Ignore()
			break loop
		}
		//this.curr = currnode
		if scan.IsEOF() {
			this.error("Arguments missing end bracket. End of file reached.")
			break loop
		}

	}
}

func (this *parser) AcceptOperator() bool {
	scan := this.scan
	for _, op := range operatorList {
		if scan.Prefix(op) {
			return true
		}
	}
	return false
}

//parseOperator
func (this *parser) parseOperator() bool {
	operator := this.commit()
	lastnode := this.lastNode()
	onode, ok := this.getCurr().(*OperatorToken)
	//push excisting operator up in tree structure
	if ok {
		//operator is the same current operator ignore
		if onode.Operator == operator {
			return true
		}
		//change order for */ presedence
		if onode.Precedence(operator) > 0 {
			if lastnode != nil {
				this.curr = lastnode.PushElement(this.newNode(NewOperatorToken(operator)))
				return true
			}
		}
		//after */ presedence fallback and continue pushing +- operators from the bottom.
		if onode.Precedence(operator) < 0 {
			for {
				v1, ok := this.curr.Parent().Value.(*OperatorToken)
				if ok && v1.Precedence(operator) <= 0 {
					this.curr = this.curr.Parent()
				} else {
					break
				}
			}
		}
		//standard operator push
		this.curr = this.push(NewOperatorToken(operator))
		return true
	}
	//set previous found value as argument of the operator
	if lastnode != nil {
		this.curr = lastnode.PushElement(this.newNode(NewOperatorToken(operator)))
	} else {
		this.error(fmt.Sprintf("Expecting a value before operator %q", operator))
		this.state = nil
	}
	return true
}

//parseLRFunc
func (this *parser) parseLRFunc() bool {
	lrfunc := this.commit()
	lastnode := this.lastNode()
	if lastnode != nil {
		this.curr = lastnode.PushElement(this.newNode(NewLRFuncToken(lrfunc)))
	} else {
		this.error(fmt.Sprintf("Expecting a value before operator %q", lrfunc))
		this.state = nil
	}
	return false
}

func (this *parser) parseText() string {
	scan := this.scan
	r := scan.Next()
	if r == '"' || r == '\'' {
		scan.Ignore()
		endqoute := r
		for {
			r = scan.Next()
			if r == endqoute {
				scan.Backup()
				txt := scan.Commit()
				scan.Next()
				scan.Ignore()
				return txt
			}
			if scan.IsEOF() {
				this.error("Missing Qoute and end of text.")
				return "Error"
			}
		}
	}
	return ""
}

func (this *parser) parseNot() stateFn {
	if this.commit() == "!" {
		fnnot := NewFuncToken("not")
		node := this.curr
		this.add(fnnot)
		//TODO Support brackets and functions
		if this.scan.ScanWord() {
			this.curr = this.newNode(NewGroupToken(""))
			this.parseIdentity()
			fnnot.AddArgument(this.curr)
			this.curr = node
		}
		return branchExpressionOperatorPart
	}
	return branchExpressionValuePart
}

func (this *parser) parseArray() *TreeNode {
	scan := this.scan
	token := scan.Commit()
	if token != "[" {
		this.error("Expecting [ before array.")
	}
	curr := this.curr
	group := this.newNode(NewGroupToken("[]"))
	this.curr = group
loop1:
	for {
	loop2:
		for {
			switch scan.Next() {
			case ' ':
				continue loop2
			case ',':
				this.ignore()
				continue loop1
			case ']':
				this.ignore()
				break loop1
			}
			scan.Backup()
			break loop2
		}
		expr := this.parseExpression()
		if expr != nil {
			group.AddElement(expr)
		} else {
			this.error("Expecting a value inside array.")
			break loop1
		}
	}
	this.curr = curr
	return group
}

func (this *parser) parseMap() *TreeNode {
	scan := this.scan
	token := scan.Commit()
	if token != "{" {
		this.error("Expecting { before map.")
	}
	curr := this.curr
	group := this.newNode(NewGroupToken("{}"))
	this.curr = group
loop1:
	for {
		this.ignore()
		expr := this.parseExpression()
		if expr != nil {
			if operator, ok := expr.Value.(*OperatorToken); ok && operator.Operator == ":" {
				switch key := expr.items[0].Value.(type) {
				case *FuncToken:
					if key.IsIdentity {
						group.AddElement(this.newNode(NewKeyValueToken(key.Name, expr.items[1])))
					} else {
						panic("Expecting key name, found function.")
					}
				case *TextToken:
					group.AddElement(this.newNode(NewKeyValueToken(key.Text, expr.items[1])))
				case *NumberToken:
					group.AddElement(this.newNode(NewKeyValueToken(strconv.FormatFloat(key.Value, byte('f'), -1, 64), expr.items[1])))
				default:
					this.error("Invalid Key Value in Map, expicting json syntax of the form {name:value,name:value} found " + expr.items[0].String())
					break loop1
				}
			} else {
				this.error("Invalid Map, expicting json syntax of the form {name:value,name:value} found " + expr.String())
				break loop1
			}
		}
	loop2:
		for {
			switch scan.Next() {
			case ' ':
				continue loop2
			case ',':
				this.ignore()
				continue loop1
			case '}':
				this.ignore()
				break loop1
			}
			scan.Backup()
			break loop1
		}
	}
	this.curr = curr
	return group
}
