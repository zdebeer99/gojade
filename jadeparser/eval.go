package jadeparser

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"reflect"
	"strconv"
	"strings"
)

type funcMap map[string]interface{}

type nilValue struct {
	Name   string
	Reason string
}

var TreeNodeType reflect.Type = reflect.TypeOf(new(TreeNode))
var LinearMapType reflect.Type = reflect.TypeOf(new(LinearMap))
var EmptyString = reflect.ValueOf("")
var nilValueType = reflect.TypeOf(nilValue{})

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

func registerFunction(addto map[string]reflect.Value, name string, fn interface{}) {
	fnvalue := reflect.ValueOf(fn)
	switch fnvalue.Kind() {
	case reflect.Func:
		addto[name] = fnvalue
	default:
		panic("argument is not a function. " + fnvalue.String())
	}
}

func (this *EvalJade) warning(warning string, node *TreeNode, args ...interface{}) {
	this.Log = append(this.Log, fmt.Sprintf("Warning: "+warning, args...))
}

// errorf formats the error and terminates processing.
func (this *EvalJade) errorf(node *TreeNode, format string, args ...interface{}) {
	err := ""
	if this.currPart != nil {
		err = fmt.Sprintf("Template %q ", this.currPart.Name)
		if node != nil {
			err += fmt.Sprintf("Linenumber %v ", LineNumber(string(this.currPart.File), node.Pos))
		}
	} else if this.currTemplate != nil {
		err = fmt.Sprintf("Template %q ", this.currTemplate.Name)
		if node != nil {
			err += fmt.Sprintf("Linenumber %v ", LineNumber(string(this.currTemplate.File), node.Pos))
		}
	}
	//	name := doublePercent(s.tmpl.Name())
	//	if s.node == nil {
	//		format = fmt.Sprintf("template: %s: %s", name, format)
	//	} else {
	//		location, context := s.tmpl.ErrorContext(s.node)
	//		format = fmt.Sprintf("template: %s: executing %q at <%s>: %s", location, name, doublePercent(context), format)
	//	}
	panic(fmt.Errorf(err+format, args...))
}

func LineNumber(source string, pos int) int {
	if pos < len(source) {
		return 1 + strings.Count(source[:pos], "\n")
	} else {
		return 1 + strings.Count(source, "\n")
	}
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

func (this *EvalJade) getValue(node *TreeNode) reflect.Value {
	if node == nil {
		return EmptyString
	}
	var result interface{}
	switch val := node.Value.(type) {
	case *NumberToken:
		result = val.Value
	case *TextToken:
		result = val.Text
	case *BoolToken:
		result = val.Value
	case *GroupToken:
		return this.getGroup(node, val)
	case *OperatorToken:
		return this.evalOperator(node, val)
	case *FuncToken:
		return this.evalFunc(node, val)
	default:
		this.errorf(node, "Invalid Type, Cannot get value of token type %T.", val)
	}
	return toReflectValue(result)
}

func (this *EvalJade) getValueAs(node *TreeNode, argtype reflect.Type) (reflect.Value, error) {
	value := this.getValue(node)
	rvalue := toReflectValue(value)
	return this.validateType(rvalue, argtype)
}

func (this *EvalJade) getGroup(node *TreeNode, group *GroupToken) reflect.Value {
	if group.GroupType == "{}" {
		result := &LinearMap{make(map[string]interface{}), make([]string, 0)}
		for _, item := range node.items {
			if kv, ok := item.Value.(*KeyValueToken); ok {
				result.Set(kv.Key, this.getValue(kv.Value).Interface())
			} else {
				this.errorf(node, "Invalid Map item. All items in a map must be of type KeyValueTokens. found %s", item.String())
			}
		}
		return toReflectValue(result)
	}
	if group.GroupType == "[]" {
		result := make([]interface{}, len(node.items))
		for i, item := range node.items {
			result[i] = this.getValue(item)
		}
		return toReflectValue(result)
	}
	if group.GroupType == "()" {
		if len(node.items) != 1 {
			this.errorf(node, "Math Function Should have 1 operator in the tree")
		}
		return this.getValue(node.items[0])
	}

	result := make([]interface{}, len(node.items))
	for i, item := range node.items {
		result[i] = this.getValue(item)
	}
	return toReflectValue(result)
}

func (this *EvalJade) getBool(node *TreeNode) bool {
	val1 := this.getValue(node)
	result, _ := isTrue(toReflectValue(val1))
	return result
}

func (this *EvalJade) getNumber(node *TreeNode) float64 {
	val1 := this.getValue(node).Interface()
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

func (this *EvalJade) jadeBlock(node *TreeNode, blockfn *FuncToken) {
	blockname := ""
	if len(blockfn.Arguments) > 0 {
		if bname, ok := blockfn.Arguments[0].Value.(*FuncToken); ok {
			blockname = bname.Name
		}
	}
	var block *jadePart
	if len(blockname) == 0 {
		//mixin block has no name an is stored in the stack.
		if blockvar, ok := this.stack.GetOk("block"); ok {
			block = &jadePart{"", blockvar.Interface().(*TreeNode), this.currTemplate.File}
		}
	} else {
		//normal page blocks has a name and is stored in the page
		block = this.Blocks[blockname]
	}
	if block == nil {
		this.errorf(node, "Block %q not found.", blockname)
	}
	this.currPart = block
	this.evalContent(block.Part)
	this.currPart = nil
}

func (this *EvalJade) jadeMixin(val *TreeNode, token *FuncToken) string {
	fn, ok := token.Arguments[0].Value.(*FuncToken)
	if !ok {
		this.errorf(val, "Expecting mixin function call.")
	}
	mixindef, ok := this.Mixins[fn.Name]
	if !ok {
		this.errorf(val, "Mixin %q not found.", fn.Name)
	}
	this.currPart = mixindef
	this.stack.AddLayer()
	defer this.stack.DropLayer()

	//set context
	mixinfn := mixindef.Part
	if fnplaceholder, ok := mixinfn.Value.(*FuncToken); ok {
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
						node.AddElement(NewTreeNode(NewKeyValueToken(key, v.Items()[1])))
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
	this.evalContent(mixinfn)
	this.currPart = nil
	return ""
}

func (this *EvalJade) jadeEach(node *TreeNode, fn *FuncToken) {
	if len(fn.Arguments) != 3 {
		this.errorf(node, "each statement invalid number of arguments, expecting at least 2. found %v", len(fn.Arguments))
	}
	var ivalue, index string
	if varvalue, ok := fn.Arguments[0].Value.(*FuncToken); ok && varvalue.IsIdentity {
		ivalue = varvalue.Name
	} else {
		this.errorf(node, "First argument of 'each' keyword must be a variable name.")
	}
	if _, ok := fn.Arguments[1].Value.(*EmptyToken); ok {
		index = ""
	} else if varvalue, ok := fn.Arguments[1].Value.(*FuncToken); ok && varvalue.IsIdentity {
		index = varvalue.Name
	} else {
		this.errorf(node, "Second argument of 'each' keyword must be a variable name or left blank.")
	}

	this.stack.AddLayer()
	defer this.stack.DropLayer()

	arrayValue := this.getValue(fn.Arguments[2])
	if !arrayValue.IsValid() {
		this.errorf(node, "value '%s' after 'each in' cannot be nil. ", fn.Arguments[2])
	}
	array := arrayValue.Interface()
	if array == nil {
		this.errorf(node, "value '%s' after 'each in' not found. ", fn.Arguments[2])
	}

	switch arrayValue.Kind() {
	case reflect.Array, reflect.Slice:
		for i := 0; i < arrayValue.Len(); i++ {
			itemvalue := arrayValue.Index(i)
			if len(index) > 0 {
				this.stack.Set(index, i)
			}
			this.stack.Set(ivalue, itemvalue)
			this.evalContent(node)
		}
	case reflect.Map:
		keys := arrayValue.MapKeys()
		for i := 0; i < len(keys); i++ {
			itemvalue := arrayValue.MapIndex(keys[i])
			if len(index) > 0 {
				this.stack.Set(index, keys[i])
			}
			this.stack.Set(ivalue, itemvalue)
			this.evalContent(node)
		}
	case reflect.Ptr:
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
		this.errorf(node, "Invalid value type after 'in' keyword, expecting an array, map or number found %s", arrayValue.Kind())
	case reflect.Float64:
		cnt := int(arrayValue.Float())
		for i := 0; i < cnt; i++ {
			this.stack.Set(ivalue, i)
			this.evalContent(node)
		}
	default:
		this.errorf(node, "Invalid value type after 'in' keyword, expecting an array, map or number found %s", arrayValue.Kind())
	}
}

func (this *EvalJade) jadeInclude(fn *FuncToken) {
	if len(fn.Arguments) > 0 {
		filename := this.getText(fn.Arguments[0])
		this.RenderFile(filename)
	}
}

func (this *EvalJade) writeText(text string) {
	this.writer.write(text)
}

func (this *EvalJade) getText(node *TreeNode) string {
	return ObjToString(this.getValue(node))
}

func ObjToString(val interface{}) string {
	switch val2 := val.(type) {
	case reflect.Value:
		if !val2.IsValid() {
			return ""
		}
		if val2.Type().AssignableTo(nilValueType) {
			return ""
		}
		return ObjToString(val2.Interface())
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
			del = ","
		}
		return buf.String()
	case *LinearMap:
		buf := new(bytes.Buffer)
		del := ""
		for _, key := range val2.Keys() {
			buf.WriteString(del)
			buf.WriteString(key + ":")
			buf.WriteString(ObjToString(val2.Get(key)))
			del = ","
		}
		return buf.String()
	case float32:
		return strconv.FormatFloat(float64(val2), 'g', 15, 32)
	case float64:
		return strconv.FormatFloat(val2, 'g', 15, 64)
	default:
		return fmt.Sprint(val)
	}
}

func (this *EvalJade) getIdentityValue(node *TreeNode, token Token) (reflect.Value, bool) {
	var val1 reflect.Value
	var err error
	switch identity := token.(type) {
	case *FuncToken:
		if !identity.IsIdentity {
			this.errorf(node, "Expecting a Variable Name on %v", identity.Name)
		}
		if sval, ok := this.stack.GetOk(identity.Name); ok {
			val1 = this.findIdentityValue(node, sval, identity, true)
		} else {
			val1 = this.findIdentityValue(node, this.data, identity, false)
		}
		if err != nil {
			switch err.(type) {
			case VariableNotDefined:
				this.warning("%s on %q", node, err.Error(), identity.String())
				return reflect.Value{}, false
			}
			this.errorf(node, "Varaible Error '%s': %s", identity.String(), err.Error())
		}
		if !val1.IsValid() {
			return reflect.Value{}, false
		}
		if val1.CanInterface() {
			return this.toCommonType(val1), true
		} else {
			//Just panic from interface(), until we find better way to handle error.
			return val1, true
		}
	default:
		this.errorf(node, "Unexpected token in identity field.")
	}
	return reflect.Value{}, false
}

func (this *EvalJade) findIdentityValue(node *TreeNode, rval reflect.Value, identity *FuncToken, got bool) reflect.Value {
	var mval reflect.Value = rval
	var err, err2 error
	if !got && len(identity.Name) > 0 {
		if identity.IsIdentity {
			mval, err2 = this.getVariableValue(rval, identity.Name)
		} else {
			//if the identity item is a function call the function.
			meth := rval.MethodByName(identity.Name)
			if meth.IsValid() {
				mval, err2 = this.callFunc(meth, identity.Name, identity.Arguments)
			} else {
				this.errorf(node, "function %s not found on struct %v", identity.Name, rval.Kind())
			}
		}
	}
	if identity.Index != nil {
		if err2 != nil {
			//Only Raise the nil ref error, if another operation is done on this value.
			this.errorf(node, err2.Error())
		}
		index := this.getText(identity.Index)
		mval, err = this.getVariableValue(mval, index)
	}
	if identity.Next != nil {
		if err2 != nil {
			//Only Raise the nil ref error, if another operation is done on this value.
			this.errorf(node, err2.Error())
		}
		mval = this.findIdentityValue(node, mval, identity.Next, false)
		err = nil
	}
	if err != nil {
		this.errorf(node, err.Error())
	}
	return mval
}

func (this *EvalJade) getVariableValue(rval reflect.Value, name string) (result reflect.Value, err error) {
	if !rval.IsValid() {
		//err = fmt.Errorf("Invalid Variable. '%s'", name)
		result = newNilValue(name, "Parent Object nil")
		return
	}
	switch rval.Kind() {
	case reflect.Map:
		rindex := reflect.ValueOf(name)
		result = rval.MapIndex(rindex)
		if !result.IsValid() {
			result = newNilValue(name, "Variable Not Found")
			err = VariableNotDefined{fmt.Errorf("Variable %q not defined on map", name)}
		}
		result = toBasicReflectValue(result, name)
		return
	case reflect.Array, reflect.Slice:
		var index int64
		index, err = strconv.ParseInt(name, 10, strconv.IntSize)
		if err != nil {
			result = newNilValue(name, "Invalid Index")
			err = fmt.Errorf("Array %q index not of type int.", name)
			return
		}
		result = rval.Index(int(index))
		if !result.IsValid() {
			result = newNilValue(name, "Invalid Index")
			err = fmt.Errorf("Invalid Index %q on %s", name, rval)
		}
		result = toBasicReflectValue(result, name)
		return
	case reflect.Struct:
		result = rval.FieldByName(name)
		if !result.IsValid() {
			result = newNilValue(name, "Variable Not Defined")
			err = VariableNotDefined{fmt.Errorf("Variable %q not defined on struct.", name)}
		}
		result = toBasicReflectValue(result, name)
		return
	case reflect.Interface, reflect.Ptr:
		//Handle LinearMap struct
		if rval.Type() == LinearMapType {
			if val1, ok := rval.Interface().(*LinearMap); ok {
				if val2 := val1.Get(name); val2 != nil {
					result = toReflectValue(val2)
				} else {
					result = newNilValue(name, "Not Found on Object.")
					err = VariableNotDefined{fmt.Errorf("Variable %q Not Found on Object.", name)}
				}
				return
			}
		}
		//Other Pointers
		return this.getVariableValue(rval.Elem(), name)
	default:
		//Handle LinearMap struct
		if rval.Type() == LinearMapType {
			if val1, ok := rval.Interface().(*LinearMap); ok {
				if val2 := val1.Get(name); val2 != nil {
					result = toReflectValue(val2)
				} else {
					result = newNilValue(name, "Not Found on Object.")
					err = VariableNotDefined{fmt.Errorf("Variable %q Not Found on Object.", name)}
				}
				return
			}
		}
		//Other Values
		err = fmt.Errorf("Variable not resolved %s", name)
		return
	}
}

func (this *EvalJade) toCommonType(val1 reflect.Value) reflect.Value {
	var float float64
	switch val1.Kind() {
	case reflect.String, reflect.Bool, reflect.Float64:
		return val1
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val1.Convert(reflect.TypeOf(float))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val1.Convert(reflect.TypeOf(float))
	case reflect.Float32:
		return val1.Convert(reflect.TypeOf(float))
	default:
		return val1
	}
}

func (this *EvalJade) findFunction(name string) reflect.Value {
	//first check the model class
	if this.data.IsValid() && this.data.NumMethod() > 0 {
		meth := this.data.MethodByName(name)
		if meth.IsValid() {
			return meth
		}
	}
	//now check for registered functions.
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

func (this *EvalJade) evalOperator(node *TreeNode, token *OperatorToken) reflect.Value {
	switch token.Operator {
	case "?":
		return this.conditional(node)
	}
	fn := this.findFunction(token.Operator)
	val1, err := this.callFunc(fn, token.Operator, node.items)
	if err != nil {
		this.errorf(node, "Error on operator %q Error: %v", token.Operator, err)
	}
	return val1
}

func (this *EvalJade) evalFunc(node *TreeNode, token *FuncToken) reflect.Value {
	if token.IsIdentity {
		val1, _ := this.getIdentityValue(node, token)
		return val1
	}
	switch token.Name {
	case "mixin":
		return EmptyString
	case "if", "unless", "else", "when", "default":
		panic("Internal Error. Function called from wrong place. function " + token.Name)
	case "case":
		this.writer.jadecase(node, token)
		return EmptyString
	case "var":
		if len(token.Arguments) != 2 {
			panic("var, expects 2 arguments, a variable name and a value. Ex: city='New York'")
		}
		this.setvariable(token.Arguments[0], token.Arguments[1])
		return EmptyString
	case "each":
		this.jadeEach(node, token)
		return EmptyString
	case escapeHtmlFunc:
		return toReflectValue(this.escapeHtml(token.Arguments[0]))
	case jadeMixinFunc:
		return toReflectValue(this.jadeMixin(node, token))
	case jadeBlockFunc:
		this.jadeBlock(node, token)
		return EmptyString
	case "include":
		this.jadeInclude(token)
		return EmptyString
	case "extends":
		return EmptyString
	}
	fn := this.findFunction(token.Name)
	val1, err := this.callFunc(fn, token.Name, token.Arguments)
	if err != nil {
		this.errorf(node, "External function %q Error: %v", token.Name, err)
	}
	return val1
}

func (this *EvalJade) evalFile(filename string) *Template {
	template := this.Loader.Load(filename)
	if template.IsJade {
		this.buildJadeFromParseResult(template)
		if len(template.Root.Extends) > 0 {
			return this.evalFile(template.Root.Extends)
		}
		this.currTemplate = template
		this.Exec(template.Root.Root)
	} else {
		this.writeText(string(template.File))
	}
	return template
}

// callFunc executes a function or method call. If it's a method, fun already has the receiver bound, so
// it looks just like a function call.  The arg list, if non-nil, includes (in the manner of the shell), arg[0]
// as the function itself.
func (s *EvalJade) callFunc(fun reflect.Value, name string, args []*TreeNode) (result reflect.Value, err error) {
	defer errRecover(&err)
	typ := fun.Type()
	numIn := len(args)
	numFixed := len(args)
	if typ.IsVariadic() {
		numFixed = typ.NumIn() - 1 // last arg is the variadic one.
		if numIn < numFixed {
			err = fmt.Errorf("wrong number of args for %s: want at least %d got %d", name, typ.NumIn()-1, len(args))
		}
	} else if numIn < typ.NumIn()-1 || !typ.IsVariadic() && numIn != typ.NumIn() {
		err = fmt.Errorf("wrong number of args for %s: want %d got %d %v", name, typ.NumIn(), len(args), args)
	}
	if !goodFunc(typ) {
		// TODO: This could still be a confusing error; maybe goodFunc should provide info.
		err = fmt.Errorf("can't call method/function %q with %d results", name, typ.NumOut())
	}
	// Build the arg list.
	argv := make([]reflect.Value, numIn)
	// Args must be evaluated. Fixed args first.
	i := 0
	for ; i < numFixed && i < len(args); i++ {
		argv[i], err = s.getValueAs(args[i], typ.In(i))
		if err != nil {
			err = fmt.Errorf("Argument %q %v", typ.In(i).Name(), err)
			return
		}
	}
	// Now the ... args.
	if typ.IsVariadic() {
		argType := typ.In(typ.NumIn() - 1).Elem() // Argument is a slice.
		for ; i < len(args); i++ {
			argv[i], err = s.getValueAs(args[i], argType)
			if err != nil {
				err = fmt.Errorf("Argument %q %v", argType.Name(), err)
				return
			}
		}
	}
	fnresult := fun.Call(argv)
	// If we have an error that is not nil, stop execution and return that error to the caller.
	if len(fnresult) == 2 && !fnresult[1].IsNil() {
		//s.at(node)
		err = fmt.Errorf("error calling %s: %s", name, fnresult[1].Interface().(error))
	}
	result = fnresult[0]
	return
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

func (this *EvalJade) conditional(node *TreeNode) reflect.Value {
	if len(node.items) != 2 {
		panic("? condition requires at least 2 arguments. condition?trueValue:falseValue Ex: true?'true value'. found: " + node.String())
	}
	var truevalue, falsevalue *TreeNode
	if split, ok := node.items[1].Value.(*OperatorToken); ok && split.Operator == ":" {
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
	if !v.IsValid() {
		return invalidKind, fmt.Errorf("invalid type for comparison, cannot compare nil value.")
	}
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
	return invalidKind, fmt.Errorf("invalid type for comparison of value %v", v.Kind())
}

// validateType guarantees that the value is valid and assignable to the type.
func (s *EvalJade) validateType(value reflect.Value, typ reflect.Type) (result reflect.Value, err error) {
	result = value
	if !value.IsValid() {
		if typ == nil || canBeNil(typ) {
			// An untyped nil interface{}. Accept as a proper nil value.
			return reflect.Zero(typ), nil
		}
		err = fmt.Errorf("invalid value; expected %s", typ)
		return
	}
	if value.Type().AssignableTo(nilValueType) {
		if typ == nil || canBeNil(typ) {
			// An untyped nil interface{}. Accept as a proper nil value.
			return reflect.Zero(typ), nil
		}
		nilv := value.Interface().(nilValue)
		err = fmt.Errorf("nil value; expected type %s. The variable %q is nil becuase %s", typ, nilv.Name, nilv.Reason)
		return
	}
	if typ != nil && !value.Type().AssignableTo(typ) {
		if value.Kind() == reflect.Interface && !value.IsNil() {
			result = value.Elem()
			if result.Type().AssignableTo(typ) {
				return
			}
			// fallthrough
		}
		//		if typ.Kind() == reflect.Float64 {
		//			return s.toCommonType(value)
		//		}
		// Does one dereference or indirection work? We could do more, as we
		// do with method receivers, but that gets messy and method receivers
		// are much more constrained, so it makes more sense there than here.
		// Besides, one is almost always all you need.
		switch {
		case value.Kind() == reflect.Ptr && value.Type().Elem().AssignableTo(typ):
			result = value.Elem()
			if !value.IsValid() {
				err = fmt.Errorf("dereference of nil pointer of type %s", typ)
				return
			}
		case reflect.PtrTo(value.Type()).AssignableTo(typ) && value.CanAddr():
			result = value.Addr()
		default:
			result, err = convertTo(value, typ)
			if err != nil {
				err = fmt.Errorf("wrong type for value; expected %s; got %s.", typ, value.Type())
				return
			}
			return
		}
	}

	return
}

func convertTo(value reflect.Value, typ reflect.Type) (result reflect.Value, err error) {
	defer errRecover(&err)
	result = value.Convert(typ)
	return
}

// isTrue reports whether the value is 'true', in the sense of not the zero of its type,
// and whether the value has a meaningful truth value.
func isTrue(val reflect.Value) (truth, ok bool) {
	if !val.IsValid() {
		// Something like var x interface{}, never set. It's a form of nil.
		return false, true
	}
	if isNullValue(val) {
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

// errRecover is the handler that turns panics into returns from the top
// level of Parse.
func errRecover(errp *error) {
	e := recover()
	if e != nil {
		switch err := e.(type) {
		case error:
			*errp = err
		default:
			*errp = fmt.Errorf("%v", e)
		}
	}
}

func toReflectValue(value interface{}) reflect.Value {
	switch val1 := value.(type) {
	case reflect.Value:
		return val1
	default:
		return reflect.ValueOf(value)
	}
}

func toBasicReflectValue(value reflect.Value, name string) reflect.Value {
	if !value.IsValid() {
		return newNilValue(name, "")
	}
	switch value.Kind() {
	case reflect.Interface:
		return value.Elem()
	}
	return value
}

func newNilValue(name string, reason string) reflect.Value {
	return reflect.ValueOf(nilValue{name, reason})
}

func isNullValue(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Ptr, reflect.Interface:
		return value.IsNil()
	case reflect.Struct:
		return value.Type().AssignableTo(nilValueType)
	default:
		return false
	}
}

func isNumber(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Float32, reflect.Float64:
		return true
	}
	return false
}
