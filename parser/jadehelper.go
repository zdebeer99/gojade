package parser

import "reflect"

// ContextStack is a stack of maps, used to track functions with context
// like for loops, mixins, etc
type ContextStack struct {
	stack []map[string]reflect.Value
	top   int
}

// NewContextStack
func NewContextStack() *ContextStack {
	ctx := &ContextStack{make([]map[string]reflect.Value, 0), -1}
	ctx.AddLayer()
	return ctx
}

// AddLayer
func (this *ContextStack) AddLayer() {
	this.stack = append(this.stack, make(map[string]reflect.Value))
	this.top = len(this.stack) - 1
}

// DropLayer
func (this *ContextStack) DropLayer() {
	if len(this.stack) > 1 {
		this.stack = this.stack[:len(this.stack)-1]
		this.top = len(this.stack) - 1
	}
}

// Set
func (this *ContextStack) Set(name string, value interface{}) {
	this.stack[this.top][name] = reflect.ValueOf(value)
}

// SetGlobal Set a value on the global scope.
func (this *ContextStack) SetGlobal(name string, value interface{}) {
	this.stack[0][name] = reflect.ValueOf(value)
}

// Get
func (this *ContextStack) Get(name string) reflect.Value {
	value, _ := this.GetOk(name)
	return value
}

//GetOk get's a value from the stack with a bool indicating if the value was found or not.
func (this *ContextStack) GetOk(name string) (value reflect.Value, ok bool) {
	layer := this.top
	value, ok = this.stack[layer][name]
	for !ok && layer > 0 {
		layer--
		value, ok = this.stack[layer][name]
	}
	return
}

// LinearMap insure the map is iterated in the same order as the key values was
// added to the map.
type LinearMap struct {
	index map[string]interface{}
	keys  []string
}

func (this *LinearMap) Set(key string, value interface{}) {
	hasKey := false
	for _, k := range this.keys {
		if k == key {
			hasKey = true
			break
		}
	}
	if !hasKey {
		this.keys = append(this.keys, key)
	}
	this.index[key] = value
}

func (this *LinearMap) Get(key string) interface{} {
	return this.index[key]
}

//Keys returns the map keys in the same order the keys was added to the map.
func (this *LinearMap) Keys() []string {
	return this.keys
}
