package parser

import (
	"fmt"
	"strconv"
	"strings"
)

type Token interface {
	Category() TokenCategory
	SetError(err error)
	Error() error
	String() string
}

type TokenCategory int

const (
	CatOther TokenCategory = iota
	CatFunction
	CatValue
)

type EmptyToken struct {
	tokencat TokenCategory
	err      error
}

func NewEmptyToken() *EmptyToken {
	return &EmptyToken{CatOther, nil}
}

func (this *EmptyToken) Category() TokenCategory {
	return this.tokencat
}

func (this *EmptyToken) Error() error {
	return this.err
}

func (this *EmptyToken) SetError(err error) {
	this.err = err
}

func (this *EmptyToken) String() string {
	return "Base()"
}

type ErrorToken struct {
	EmptyToken
}

func NewErrorToken(err string) *ErrorToken {
	return &ErrorToken{EmptyToken{CatOther, fmt.Errorf(err)}}
}

type NumberToken struct {
	EmptyToken
	Value float64
}

func NewNumberToken(value string) *NumberToken {
	node := &NumberToken{EmptyToken{CatValue, nil}, 0}
	val1, err := strconv.ParseFloat(value, 64)
	if err != nil {
		panic("Number node failed to parse string to number. (" + value + ")")
		return node
	}
	node.Value = val1
	return node
}

func (this *NumberToken) String() string {
	return fmt.Sprintf("Number(%v)", this.Value)
}

type BoolToken struct {
	EmptyToken
	Value bool
}

func NewBoolToken(value string) *BoolToken {
	node := &BoolToken{EmptyToken{CatValue, nil}, false}
	node.Value = strings.ToLower(value) == "true"
	return node
}

func (this *BoolToken) String() string {
	return fmt.Sprintf("Bool(%v)", this.Value)
}

type FuncToken struct {
	EmptyToken
	Name       string
	Arguments  []*TreeNode
	Next       *FuncToken
	IsIdentity bool
	Index      string
}

func NewFuncToken(name string) *FuncToken {
	return &FuncToken{EmptyToken{CatFunction, nil}, name, make([]*TreeNode, 0), nil, false, ""}
}

func NewIdentityToken(name string) *FuncToken {
	return &FuncToken{EmptyToken{CatFunction, nil}, name, make([]*TreeNode, 0), nil, true, ""}
}

func (this *FuncToken) AddArgument(arg *TreeNode) {
	this.Arguments = append(this.Arguments, arg)
}

func (this *FuncToken) String() string {
	var out string
	chain := this
	del := ""
	for chain != nil {
		out += del
		if chain.IsIdentity {
			out += chain.Name
		} else {
			args := make([]string, len(chain.Arguments))
			for i, v := range chain.Arguments {
				args[i] = fmt.Sprintf("%s", strings.Replace(strings.Replace(v.String(), "\n", ",", -1), "  ", "", -1))
			}
			out += fmt.Sprintf("%s(%s)", chain.Name, args)
		}
		if chain.Index != "" {
			out += fmt.Sprintf("[%s]", chain.Index)
		}
		del = "."
		chain = chain.Next
	}
	return out
}

type OperatorToken struct {
	EmptyToken
	Operator string
	lvl      int
}

func NewOperatorToken(operator string) *OperatorToken {
	op := &OperatorToken{EmptyToken{CatFunction, nil}, "", -1}
	op.SetOperator(operator)
	return op
}

func (this *OperatorToken) SetOperator(operator string) {
	this.Operator = operator
	this.lvl = operators.Level(operator)
	if this.lvl < 0 {
		panic(fmt.Errorf("Invalid Operator %q", operator))
	}
}

// OperatorPrecedence return true if the operator argument is lower than the current operator.
func (this *OperatorToken) Precedence(operator string) int {
	lvl := operators.Level(operator)
	switch {
	case lvl == this.lvl:
		return 0
	case lvl > this.lvl:
		return 1
	case lvl < this.lvl:
		return -1
	}
	panic("Unreachable code")
}

func (this *OperatorToken) String() string {
	return fmt.Sprintf("Operator(%s)", this.Operator)
}

type OperatorPrecedence [][]string

func (this OperatorPrecedence) Level(operator string) int {

	for level, operators := range this {
		for _, op := range operators {
			if op == operator {
				return len(this) - level
			}
		}
	}
	return -1
}

func (this OperatorPrecedence) All() []string {
	out := make([]string, 0)
	for _, operators := range this {
		for _, op := range operators {
			out = append(out, op)
		}
	}
	return out
}

var operators OperatorPrecedence = OperatorPrecedence{
	{"*", "/", "%"},
	{"+", "-"},
	{"==", "!=", ">=", "<=", ">", "<"},
	{"&&", "and"},
	{"||", "or"},
	{":"},
	{"?"},
	{"="},
}

var operatorList []string = operators.All()

type LRFuncToken struct {
	EmptyToken
	Name string
}

func NewLRFuncToken(name string) *LRFuncToken {
	return &LRFuncToken{EmptyToken{CatFunction, nil}, name}
}

func (this *LRFuncToken) String() string {
	return fmt.Sprintf("lrfunc(%s)", this.Name)
}

type GroupToken struct {
	EmptyToken
	GroupType string
}

func NewGroupToken(group string) *GroupToken {
	return &GroupToken{EmptyToken{CatOther, nil}, group}
}

func (this *GroupToken) String() string {
	return fmt.Sprintf("Group(%s)", this.GroupType)
}

type HtmlTagToken struct {
	EmptyToken
	TagName     string
	Attributes  []*TreeNode
	SelfClosing bool
}

func NewHtmlTagToken(tagname string) *HtmlTagToken {
	return &HtmlTagToken{EmptyToken{CatOther, nil}, tagname, make([]*TreeNode, 0), false}
}

func (this *HtmlTagToken) String() string {
	attr := make([]string, len(this.Attributes))
	for i, v := range this.Attributes {
		attr[i] = fmt.Sprintf("(%s)", strings.Replace(v.String(), "\n", ",", -1))
	}
	if this.SelfClosing {
		return fmt.Sprintf("HTML%s(%s)/", this.TagName, strings.Join(attr, ","))
	} else {
		return fmt.Sprintf("HTML%s(%s)", this.TagName, strings.Join(attr, ","))
	}
}

func (this *HtmlTagToken) AddAttribute(attr *TreeNode) *TreeNode {
	this.Attributes = append(this.Attributes, attr)
	return attr
}

func (this *HtmlTagToken) addKeyValue(key string, value *TreeNode) *TreeNode {
	kvnode := this.findAttribute(key)
	if kvnode == nil {
		kvnode = NewTreeNode(NewKeyValueToken(key, value))
		this.Attributes = append(this.Attributes, kvnode)
		return kvnode
	}
	kv := kvnode.Value.(*KeyValueToken)
	kv.Value = value
	return kvnode
}

func (this *HtmlTagToken) AddKeyValue(key string, value *TreeNode) *TreeNode {
	if strings.ToLower(key) == "class" {
		return this.SetClass(value)
	}
	return this.addKeyValue(key, value)
}

func (this *HtmlTagToken) findAttribute(key string) *TreeNode {
	key = strings.ToLower(key)
	for _, v := range this.Attributes {
		kv, ok := v.Value.(*KeyValueToken)
		if ok {
			if strings.ToLower(kv.Key) == key {
				return v
			}
		}
	}
	return nil
}

func (this *HtmlTagToken) SetClass(attr *TreeNode) *TreeNode {
	classattr := this.findAttribute("class")
	var kv *KeyValueToken
	if classattr == nil {
		classes := NewTreeNode(NewGroupToken(""))
		classattr = this.addKeyValue("class", classes)
	}
	kv, ok := classattr.Value.(*KeyValueToken)
	if !ok {
		panic("Expecting Key Value Token in attributes.")
	}

	kv.Value.AddElement(attr)
	return classattr
}

type HtmlDocTypeToken struct {
	EmptyToken
	Attributes []string
}

func NewHtmlDocTypeToken() *HtmlDocTypeToken {
	return &HtmlDocTypeToken{EmptyToken{CatOther, nil}, make([]string, 0)}
}

func (this *HtmlDocTypeToken) String() string {
	return fmt.Sprintf("doctype(%s)", strings.Join(this.Attributes, " "))
}

type TextToken struct {
	EmptyToken
	Text string
}

func NewTextToken(text string) *TextToken {
	return &TextToken{EmptyToken{CatValue, nil}, text}
}

func (this *TextToken) String() string {
	return fmt.Sprintf("%q", this.Text)
}

type KeyValueToken struct {
	EmptyToken
	Key   string
	Value *TreeNode
}

func NewKeyValueToken(key string, value *TreeNode) *KeyValueToken {
	return &KeyValueToken{EmptyToken{CatValue, nil}, key, value}
}

func (this *KeyValueToken) String() string {
	return fmt.Sprintf("%s=%s", this.Key, this.Value.String())
}

type CommentToken struct {
	EmptyToken
	CommentType string
}

func NewCommentToken(commentType string) *CommentToken {
	return &CommentToken{EmptyToken{CatValue, nil}, commentType}
}

func (this *CommentToken) String() string {
	return fmt.Sprintf("%s", this.CommentType)
}
