package parser

import (
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

func isnull(value, result interface{}) interface{} {
	if value == nil {
		return result
	}
	return value
}
