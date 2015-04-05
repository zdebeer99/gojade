package parser

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

type funcMap map[string]interface{}

var TreeNodeType reflect.Type = reflect.TypeOf(new(TreeNode))
var LinearMapType reflect.Type = reflect.TypeOf(new(LinearMap))

// createValueFuncs turns a FuncMap into a map[string]reflect.Value
func createValueFuncs(funcMap funcMap) map[string]reflect.Value {
	m := make(map[string]reflect.Value)
	addValueFuncs(m, funcMap)
	return m
}

// addValueFuncs adds to values the functions in funcs, converting them to reflect.Values.
func addValueFuncs(out map[string]reflect.Value, in funcMap) {
	for name, fn := range in {
		v := reflect.ValueOf(fn)
		if v.Kind() != reflect.Func {
			panic("value for " + name + " not a function")
		}
		if !goodFunc(v.Type()) {
			panic(fmt.Errorf("can't install method/function %q with %d results", name, v.Type().NumOut()))
		}
		out[name] = v
	}
}

func (this *EvalJade) warning(warning string, node *TreeNode, args ...interface{}) {
	this.Log = append(this.Log, fmt.Sprintf("Warning: "+warning, args...))
}

func (this *EvalJade) router(node *TreeNode) {
	writer := this.writer
	switch val := node.Value.(type) {
	case *EmptyToken:
		this.evalContent(node)
	case *HtmlDocTypeToken:
		writer.HtmlDocType(val)
	case *HtmlTagToken:
		writer.HtmlTag(node, val)
	case *NumberToken:
		writer.write(strconv.FormatFloat(val.Value, 0, 0, 64))
	case *TextToken:
		writer.text(val)
	case *GroupToken:
		writer.Group(node)
	case *KeyValueToken:
		writer.KeyValueToken(val)
	case *LRFuncToken:
		writer.lrfunc(node, val)
	case *FuncToken:
		writer.stdfunc(node, val)
	case *CommentToken:
		writer.Comment(node, val)
	}
}

func (this *EvalJade) getValue(node *TreeNode) interface{} {
	if node == nil {
		return ""
	}
	switch val := node.Value.(type) {
	case *NumberToken:
		return val.Value
	case *TextToken:
		return val.Text
	case *BoolToken:
		return val.Value
	case *GroupToken:
		return this.getGroup(node, val)
	case *OperatorToken:
		return this.evalOperator(node, val)
	case *LRFuncToken:
		return this.evalLRFunc(node, val)
	case *FuncToken:
		return this.evalFunc(node, val)
	default:
		panic(fmt.Errorf("Invalid Type, Cannot get value of token type %T.", val))
	}
}

func (this *EvalJade) getGroup(node *TreeNode, group *GroupToken) interface{} {
	if group.GroupType == "{}" {
		result := &LinearMap{make(map[string]interface{}), make([]string, 0)}
		for _, item := range node.items {
			if kv, ok := item.Value.(*KeyValueToken); ok {
				result.Set(kv.Key, this.getValue(kv.Value))
			} else {
				panic("Invalid Map item. All items in a map must be of type KeyValueTokens. found " + item.String())
			}
		}
		return result
	}
	if group.GroupType == "[]" {
		result := make([]interface{}, len(node.items))
		for i, item := range node.items {
			result[i] = this.getValue(item)
		}
		return result
	}

	result := make([]interface{}, len(node.items))
	for i, item := range node.items {
		result[i] = this.getValue(item)
	}
	return result
}

func (this *EvalJade) getBool(node *TreeNode) bool {
	val1 := this.getValue(node)
	result, _ := isTrue(reflect.ValueOf(val1))
	return result
}

func (this *EvalJade) getNumber(node *TreeNode) float64 {
	val1 := this.getValue(node)
	switch val2 := val1.(type) {
	case int:
		return float64(val2)
	case float64:
		return val2
	case string:
		fl, err := strconv.ParseFloat(val2, 64)
		if err != nil {
			panic(fmt.Errorf("Failed to convert %q to a number.", val2))
		}
		return fl
	default:
		panic(fmt.Errorf("Invalid type. Token with the value %v and type %T conversion not implemented yet.", val2, val2))
	}
}

func (this *EvalJade) escapeHtml(val *TreeNode) string {
	return html.EscapeString(this.getText(val))
}

func (this *EvalJade) jadeBlock(blockfn *FuncToken) {
	blockname := ""
	if len(blockfn.Arguments) > 0 {
		if bname, ok := blockfn.Arguments[0].Value.(*FuncToken); ok {
			blockname = bname.Name
		}
	}
	var block *TreeNode
	if len(blockname) == 0 {
		//mixin block has no name an is stored in the stack.
		if blockvar, ok := this.stack.GetOk("block"); ok {
			block = blockvar.Interface().(*TreeNode)
		}
	} else {
		//normal page blocks has a name and is stored in the page
		block = this.Blocks[blockname]
	}
	if block == nil {
		panic("Block '" + blockname + "' not found.")
	}
	this.evalContent(block)
}

func (this *EvalJade) jadeMixin(val *TreeNode, token *FuncToken) string {
	fn, ok := token.Arguments[0].Value.(*FuncToken)
	if !ok {
		panic("Expecting mixin function call.")
	}
	mixindef, ok := this.Mixins[fn.Name]
	if !ok {
		panic("Mixin '" + fn.Name + "' not found.")
	}

	this.stack.AddLayer()
	defer this.stack.DropLayer()

	//set context
	if fnplaceholder, ok := mixindef.Value.(*FuncToken); ok {
		if fndef, ok := fnplaceholder.Arguments[0].Value.(*FuncToken); ok && !fndef.IsIdentity {
			for i, arg := range fndef.Arguments {
				argidentity := arg.Value.(*FuncToken)
				this.stack.Set(argidentity.Name, this.getValue(fn.Arguments[i]))
			}
			if fn.Next != nil && fn.Next.Name == "attributes" {
				attributes := NewGroupToken("{}")
				node := NewTreeNode(attributes)
				for _, v := range fn.Next.Arguments {
					if op, ok := v.Value.(*OperatorToken); ok && op.Operator == "=" {
						key := v.Items()[0].Value.(*FuncToken).Name
						node.Add(NewKeyValueToken(key, v.Items()[1]))
					} else {
						panic("Expecting Key Value pairs seperated by '=' found '" + v.String() + "'")
					}
				}
				this.stack.Set("attributes", this.getGroup(node, attributes))
			}
		}
	}
	//set block
	if len(val.Items()) > 0 {
		this.stack.Set("block", val)
	}

	this.evalContent(mixindef)

	return ""
}

func (this *EvalJade) jadeEach(node *TreeNode, fn *FuncToken) {
	if len(fn.Arguments) != 3 {
		panic("each statement does not have the right number of arguments. Example each value,[index] in array. found " + string(len(fn.Arguments)))
	}
	var ivalue, index string
	if varvalue, ok := fn.Arguments[0].Value.(*FuncToken); ok && varvalue.IsIdentity {
		ivalue = varvalue.Name
	} else {
		panic("First argument of 'each' keyword must be a variable name.")
	}
	if _, ok := fn.Arguments[1].Value.(*EmptyToken); ok {
		index = ""
	} else if varvalue, ok := fn.Arguments[1].Value.(*FuncToken); ok && varvalue.IsIdentity {
		index = varvalue.Name
	} else {
		panic("Second argument of 'each' keyword must be a variable name or left blank.")
	}

	this.stack.AddLayer()
	defer this.stack.DropLayer()

	array := this.getValue(fn.Arguments[2])
	if array == nil {
		panic(fmt.Sprintf("value '%s' after 'each in' not found. ", fn.Arguments[2]))
	}
	//handle iterating jsondata object
	if val1, ok := array.(*LinearMap); ok {
		for _, k := range val1.keys {
			if len(index) > 0 {
				this.stack.Set(index, k)
			}
			this.stack.Set(ivalue, val1.Get(k))
			this.evalContent(node)
		}
		return
	}

	arrayValue := reflect.ValueOf(array)
	switch arrayValue.Kind() {
	case reflect.Array, reflect.Slice:
		for i := 0; i < arrayValue.Len(); i++ {
			itemvalue := arrayValue.Index(i)
			if len(index) > 0 {
				this.stack.Set(index, i)
			}
			this.stack.Set(ivalue, itemvalue.Interface())
			this.evalContent(node)
		}
	case reflect.Map:
		keys := arrayValue.MapKeys()
		for i := 0; i < len(keys); i++ {
			itemvalue := arrayValue.MapIndex(keys[i])
			if len(index) > 0 {
				this.stack.Set(index, keys[i].Interface())
			}
			this.stack.Set(ivalue, itemvalue.Interface())
			this.evalContent(node)
		}
	case reflect.Float64:
		cnt := int(arrayValue.Float())
		for i := 0; i < cnt; i++ {
			this.stack.Set(ivalue, i)
			this.evalContent(node)
		}
	default:
		panic(fmt.Sprintf("Expecting an Array or Map after 'in' keyword. found %s", array))
	}
}

func (this *EvalJade) jadeInclude(fn *FuncToken) {
	if len(fn.Arguments) > 0 {
		filename := this.getText(fn.Arguments[0])
		this.RenderFile(filename)
	}
}

func (this *EvalJade) isJadeFile(filename string) bool {
	return strings.HasSuffix(filename, ".jade")
}

func (this *EvalJade) writeText(text string) {
	this.writer.write(text)
}

func (this *EvalJade) getText(node *TreeNode) string {
	return ObjToString(this.getValue(node))
}

func ObjToString(val interface{}) string {
	switch val2 := val.(type) {
	case string:
		return val2
	case []interface{}:
		buf := new(bytes.Buffer)
		del := ""
		for _, item := range val2 {
			buf.WriteString(del)
			buf.WriteString(ObjToString(item))
			del = " "
		}
		return buf.String()
	case map[string]interface{}:
		buf := new(bytes.Buffer)
		del := ""
		for key, value := range val2 {
			buf.WriteString(del)
			buf.WriteString(key + ":")
			buf.WriteString(ObjToString(value))
			del = ";"
		}
		return buf.String()
	case *LinearMap:
		buf := new(bytes.Buffer)
		del := ""
		for _, key := range val2.Keys() {
			buf.WriteString(del)
			buf.WriteString(key + ":")
			buf.WriteString(ObjToString(val2.Get(key)))
			del = ";"
		}
		return buf.String()
	default:
		return fmt.Sprint(val)
	}
}

func (this *EvalJade) getIdentityValue(node *TreeNode, token Token) interface{} {
	var val1 reflect.Value
	var err error
	switch identity := token.(type) {
	case *FuncToken:
		if !identity.IsIdentity {
			panic("Expecting a Variable Name. Functions called from data not supported yet.")
		}
		if sval, ok := this.stack.GetOk(identity.Name); ok {
			val1, err = this.getIdentityRValue(sval, identity.Next)
		} else {
			val1, err = this.getIdentityRValue(this.data, identity)
		}
		if err != nil {
			switch err.(type) {
			case VariableNotDefined:
				this.warning("%s on %q", node, err.Error(), identity.String())
				return nil
			}
			panic(fmt.Errorf("Varaible Error '%s': %s", identity.String(), err.Error()))
		}
		if !val1.IsValid() {
			return nil
		}
		if val1.CanInterface() {
			return this.toCommonType(val1.Interface())
		} else {
			//Just panic from interface(), until we find better way to handle error.
			return val1.Interface()
		}
	default:
		panic("Unexpected token in identity field.")
	}
}

func (this *EvalJade) getIdentityRValue(rval reflect.Value, identity *FuncToken) (result reflect.Value, err error) {
	if identity == nil {
		return rval, nil
	}
	if !rval.IsValid() {
		err = fmt.Errorf("Invalid Variable. '%s'", identity.String())
		return
	}
	if !identity.IsIdentity {
		panic("functions on type not supported yet. function: " + identity.String())
	}
	switch rval.Kind() {
	case reflect.Map:
		rindex := reflect.ValueOf(identity.Name)
		mval := rval.MapIndex(rindex)
		if !mval.IsValid() {
			return mval, VariableNotDefined{fmt.Errorf("Variable %q not defined on map", identity.Name)}
		}
		return this.getIdentityRValue(mval, identity.Next)
	case reflect.Array, reflect.Slice:
		if identity.Index == "" {
			return rval, nil
		}
		mval := rval.Index(getIdentityIndex(rval, identity.Index))
		if !mval.IsValid() {
			return mval, fmt.Errorf("Invalid Index %q on variable %q on %s", identity.Index, identity.Name, rval)
		}
		return this.getIdentityRValue(mval, identity.Next)
	case reflect.Struct:
		mval := rval.FieldByName(identity.Name)
		if !mval.IsValid() {
			return mval, VariableNotDefined{fmt.Errorf("Variable %q not defined on struct.", identity.Name)}
		}
		return this.getIdentityRValue(mval, identity.Next)
	case reflect.Interface, reflect.Ptr:
		//Handle LinearMap struct
		if rval.Type() == LinearMapType {
			if val1, ok := rval.Interface().(*LinearMap); ok {
				return this.getIdentityRValue(reflect.ValueOf(val1.Get(identity.Name)), identity.Next)
			}
		}
		//Other Pointers
		return this.getIdentityRValue(rval.Elem(), identity)
	default:
		//Handle LinearMap struct
		if rval.Type() == LinearMapType {
			if val1, ok := rval.Interface().(*LinearMap); ok {
				return this.getIdentityRValue(reflect.ValueOf(val1.Get(identity.Name)), identity.Next)
			}
		}
		//Other Values
		return this.getIdentityRValue(rval, identity.Next)
	}
}

func getIdentityIndex(obj interface{}, index string) int {
	i, err := strconv.ParseInt(index, 10, 32)
	if err != nil {
		panic("Failed to convert index '" + index + "' to a int. " + err.Error())
	}
	return int(i)
}

func (this *EvalJade) toCommonType(val1 interface{}) interface{} {
	switch val2 := val1.(type) {
	case string, bool, float64:
		return val2
	case byte:
		return float64(val2)
	case int:
		return float64(val2)
	case int16:
		return float64(val2)
	case int32:
		return float64(val2)
	case uint:
		return float64(val2)
	default:
		return val1
	}
}

func (this *EvalJade) findFunction(name string) reflect.Value {
	fn, ok := this.builtin[name]
	if !ok {
		fn, ok = this.Extfunc[name]
		if !ok {
			panic(fmt.Errorf("Function %q not found.", name))
		}
	}
	return fn
}

func (this *EvalJade) evalContent(node *TreeNode) {
	var ifresult int
	for _, item := range node.Items() {
		//handle if statements
		fntoken, ok := item.Value.(*FuncToken)
		if ok && (fntoken.Name == "if" || fntoken.Name == "unless") {
			ifresult = this.evalIfElse(item, fntoken)
			continue
		}
		if ok && fntoken.Name == "else" {
			if ifresult == 2 {
				if len(fntoken.Arguments) > 0 {
					ifresult = this.evalIfElse(item, fntoken)
					continue
				}
				this.evalContent(item)
				ifresult = 0
			}
			continue
		}
		ifresult = 0
		//handle everything else
		this.router(item)
	}
}

/// evalIf returns
/// 0 - no if statement was found,
/// 1 - if returned true
/// 2 - if returned false
func (this *EvalJade) evalIfElse(node *TreeNode, token *FuncToken) int {
	if token.Name == "if" {
		if len(token.Arguments) != 1 {
			panic(fmt.Errorf("Invalid number of arguments statement %q, expecting 1 found %v", node.String(), len(token.Arguments)))
		}
		if this.getBool(token.Arguments[0]) {
			this.evalContent(node)
			return 1
		} else {
			return 2
		}
	}
	if token.Name == "unless" {
		if len(token.Arguments) != 1 {
			panic(fmt.Errorf("Invalid number of arguments statement %q, expecting 1 found %v", node.String(), len(token.Arguments)))
		}
		if !this.getBool(token.Arguments[0]) {
			this.evalContent(node)
			return 1
		} else {
			return 2
		}
	}
	if token.Name == "else" && len(token.Arguments) > 0 {
		if len(token.Arguments) != 1 {
			panic(fmt.Errorf("Invalid number of arguments statement %q, expecting 1 found %v", node.String(), len(token.Arguments)))
		}
		if this.getBool(token.Arguments[0]) {
			this.evalContent(node)
			return 1
		} else {
			return 2
		}
	}

	return 0
}

func (this *EvalJade) evalOperator(node *TreeNode, token *OperatorToken) interface{} {
	switch token.Operator {
	case "?":
		return this.conditional(node)
	}
	fn := this.findFunction(token.Operator)
	return this.callFunc(fn, node.items)
}

func (this *EvalJade) evalLRFunc(node *TreeNode, token *LRFuncToken) interface{} {
	return 0
}

func (this *EvalJade) evalFunc(node *TreeNode, token *FuncToken) interface{} {
	if token.IsIdentity {
		return this.getIdentityValue(node, token)
	}
	switch token.Name {
	case "mixin":
		return ""
	case "if", "unless", "else", "when", "default":
		panic("Internal Error. Function called from wrong place. function " + token.Name)
	case "case":
		this.writer.jadecase(node, token)
		return ""
	case "var":
		if len(token.Arguments) != 2 {
			panic("var, expects 2 arguments, a variable name and a value. Ex: city='New York'")
		}
		this.setvariable(token.Arguments[0], token.Arguments[1])
		return ""
	case "each":
		this.jadeEach(node, token)
		return ""
	case escapeHtmlFunc:
		return this.escapeHtml(token.Arguments[0])
	case jadeMixinFunc:
		return this.jadeMixin(node, token)
	case jadeBlockFunc:
		this.jadeBlock(token)
		return ""
	case "include":
		this.jadeInclude(token)
		return ""
	}
	fn := this.findFunction(token.Name)
	return this.callFunc(fn, token.Arguments)
}

func (this *EvalJade) callFunc(fn reflect.Value, args []*TreeNode) interface{} {
	fntype := fn.Type()
	argv := make([]reflect.Value, len(args))

	for i := 0; i < len(args); i++ {
		argpos := i
		if i >= fntype.NumIn() {
			argpos = fntype.NumIn() - 1
		}
		if fntype.In(argpos) == TreeNodeType {
			argv[i] = reflect.ValueOf(args[i])
		} else {
			argv[i] = reflect.ValueOf(this.getValue(args[i]))
		}
	}
	if fntype.NumOut() == 0 {
		fn.Call(argv)
		return nil
	}
	result := fn.Call(argv)
	return result[0].Interface()
}

func (this *EvalJade) setvariable(nameNode *TreeNode, valueNode *TreeNode) {
	varname, ok := nameNode.Value.(*FuncToken)
	if !ok {
		panic("var declaration expecting variable name. Found " + nameNode.Value.String())
	}
	if !varname.IsIdentity {
		panic("var declaration expecting variable name. Found Function " + nameNode.Value.String())
	}

	this.stack.SetGlobal(varname.Name, this.getValue(valueNode))
}

func (this *EvalJade) conditional(node *TreeNode) interface{} {
	if len(node.items) != 2 {
		panic("? condition requires at least 2 arguments. condition?trueValue:falseValue Ex: true?'true value'. found: " + node.String())
	}
	var truevalue, falsevalue *TreeNode
	if split, ok := node.items[1].Value.(*OperatorToken); ok {
		if split.Operator != ":" {
			panic("Expecting : after ? conditional statement. found " + split.Operator)
		}
		truevalue = node.items[1].items[0]
		falsevalue = node.items[1].items[1]
	} else {
		truevalue = node.items[1]
	}
	if this.getBool(node.items[0]) {
		return this.getValue(truevalue)
	} else {
		return this.getValue(falsevalue)
	}
}

// floadfile Find and Load a file.
func (this *EvalJade) floadfile(filename string) ([]byte, error) {
	filename, err := this.findfile(filename)
	if err != nil {
		return nil, err
	}
	return this.loadfile(filename)
}

func (this *EvalJade) findfile(filename string) (string, error) {
	filename = strings.Trim(filename, " ")
	if !strings.ContainsRune(filename, '.') {
		filename = filename + ".jade"
	}
	if !strings.HasSuffix(this.viewPath, "/") {
		filename = this.viewPath + "/" + filename
	} else {
		filename = this.viewPath + filename
	}
	filename, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}
	return filename, nil
}

func (this *EvalJade) loadfile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}

/*
********************************************************************************
 */

var (
	errBadComparisonType = errors.New("invalid type for comparison")
	errBadComparison     = errors.New("incompatible types for comparison")
	errNoComparison      = errors.New("missing argument for comparison")
)

type kind int

const (
	invalidKind kind = iota
	boolKind
	complexKind
	intKind
	floatKind
	integerKind
	stringKind
	uintKind
)

func basicKind(v reflect.Value) (kind, error) {
	switch v.Kind() {
	case reflect.Bool:
		return boolKind, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intKind, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintKind, nil
	case reflect.Float32, reflect.Float64:
		return floatKind, nil
	case reflect.Complex64, reflect.Complex128:
		return complexKind, nil
	case reflect.String:
		return stringKind, nil
	}
	return invalidKind, errBadComparisonType
}

// isTrue reports whether the value is 'true', in the sense of not the zero of its type,
// and whether the value has a meaningful truth value.
func isTrue(val reflect.Value) (truth, ok bool) {
	if !val.IsValid() {
		// Something like var x interface{}, never set. It's a form of nil.
		return false, true
	}
	switch val.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		truth = val.Len() > 0
	case reflect.Bool:
		truth = val.Bool()
	case reflect.Complex64, reflect.Complex128:
		truth = val.Complex() != 0
	case reflect.Chan, reflect.Func, reflect.Ptr, reflect.Interface:
		truth = !val.IsNil()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		truth = val.Int() != 0
	case reflect.Float32, reflect.Float64:
		truth = val.Float() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		truth = val.Uint() != 0
	case reflect.Struct:
		truth = true // Struct values are always true.
	default:
		return
	}
	return truth, true
}
