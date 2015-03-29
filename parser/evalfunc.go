package parser

import (
	"fmt"
	"reflect"
	"strings"
)

func (this *EvalJade) registerStandardFunctions() {
	this.RegisterFunction("len", len)
	this.RegisterFunction("upper", upper)
	this.RegisterFunction("lower", lower)
	this.RegisterFunction("format", format)

}

func len(value interface{}) int {
	rvalue := reflect.ValueOf(value)
	return rvalue.Len()
}

func upper(txt string) string {
	return strings.ToUpper(txt)
}

func lower(txt string) string {
	return strings.ToLower(txt)
}

func format(txt string, args ...interface{}) string {
	return fmt.Sprintf(txt, args...)
}
