package jadeparser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

func (this *EvalJade) registerStandardFunctions() {

	registerFunction(this.builtin, "len", length)
	registerFunction(this.builtin, "upper", upper)
	registerFunction(this.builtin, "lower", lower)
	registerFunction(this.builtin, "format", format)
	registerFunction(this.builtin, "isnull", isnull)
	registerFunction(this.builtin, "ifnull", ifnull)
	registerFunction(this.builtin, "json", tojson)
}

func length(value interface{}) int {
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

func isnull(value interface{}) bool {
	if value == nil {
		return true
	}
	return false
}

func ifnull(value, result interface{}) interface{} {
	if value == nil {
		return result
	}
	return value
}

func tojson(value interface{}) string {
	buf := new(bytes.Buffer)
	b, err := json.Marshal(value)
	if err != nil {
		buf.WriteString(err.Error())
		return buf.String()
	}
	buf.Write(b)
	return buf.String()
}
