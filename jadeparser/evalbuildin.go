package jadeparser

import (
	"bytes"
	"fmt"
)

type operatorfunction struct {
	fnFloat func(float64, float64) float64
	fnBool  func(bool, bool) bool
	fnOther interface{}
}

type operatorMap map[string]operatorfunction

const (
	escapeHtmlFunc = "escapeHtml"
	attributesFunc = "explodeAttributes"
	jadeMixinFunc  = "jadeMixin"
	jadeBlockFunc  = "block"
)

var builtin funcMap = funcMap{
	"+":            addNumOrString,
	"-":            subtract,
	"*":            multiply,
	"/":            divide,
	"&&":           and,
	"and":          and,
	"||":           or,
	"or":           or,
	"==":           eq,
	"!=":           ne,
	"<":            lt,
	">":            gt,
	"<=":           le,
	">=":           ge,
	"not":          not,
	attributesFunc: explodeAttributes,
}

//'var' and '=' is handled internally.

func explodeAttributes(attr interface{}) string {
	buf := new(bytes.Buffer)
	del := ""
	switch val1 := attr.(type) {
	case map[string]interface{}:
		for key, value := range val1 {
			buf.WriteString(del)
			buf.WriteString(key + "=\"")
			buf.WriteString(ObjToString(value) + "\"")
			del = " "
		}
	case *LinearMap:
		for _, key := range val1.Keys() {
			buf.WriteString(del)
			buf.WriteString(key + "=\"")
			buf.WriteString(ObjToString(val1.Get(key)) + "\"")
			del = " "
		}
	default:
		panic("Type not supported as an attribute collection.")
	}
	return buf.String()
}

func not(val bool) bool {
	return !val
}

func addNumOrString(arg1 interface{}, arg2 ...interface{}) interface{} {
	var result float64
	buf := new(bytes.Buffer)
	buf.WriteString(ObjToString(arg1))
	_, havestring := arg1.(string)
	if !havestring {
		result = toReflectValue(arg1).Float()
	}
	for i := 0; i < len(arg2); i++ {
		buf.WriteString(ObjToString(arg2[i]))
		_, isstring := arg2[i].(string)
		havestring = havestring || isstring
		if !havestring {
			result += toReflectValue(arg2[i]).Float()
		}
	}
	if havestring {
		return buf.String()
	} else {
		return result
	}
}

func addString(arg1 interface{}, arg2 ...interface{}) string {
	buf := new(bytes.Buffer)
	buf.WriteString(ObjToString(arg1))

	for i := 0; i < len(arg2); i++ {
		buf.WriteString(ObjToString(arg2[i]))
	}
	return buf.String()
}

func addNum(arg1 interface{}, arg2 ...interface{}) float64 {
	result := toReflectValue(arg1).Float()
	for i := 0; i < len(arg2); i++ {
		result = result + toReflectValue(arg2[i]).Float()
	}
	return result
}

func subtract(arg1 float64, arg2 ...float64) float64 {
	result := arg1
	for i := 0; i < len(arg2); i++ {
		result = result - arg2[i]
	}
	return result
}

func multiply(arg1 float64, arg2 ...float64) float64 {
	result := arg1
	for i := 0; i < len(arg2); i++ {
		result = result * arg2[i]
	}
	return result
}

func divide(arg1 float64, arg2 ...float64) float64 {
	result := arg1
	for i := 0; i < len(arg2); i++ {
		result = result / arg2[i]
	}
	return result
}

// Boolean logic.
func truth(a interface{}) bool {
	t, _ := isTrue(toReflectValue(a))
	return t
}

func and(arg0 interface{}, args ...interface{}) bool {
	if !truth(arg0) {
		return false
	}
	for i := range args {
		arg0 = args[i]
		if !truth(arg0) {
			return false
		}
	}
	return true
}

// or computes the Boolean OR of its arguments, returning
// the first true argument it encounters, or the last argument.
func or(arg0 interface{}, args ...interface{}) bool {
	if truth(arg0) {
		return true
	}
	for i := range args {
		arg0 = args[i]
		if truth(arg0) {
			return true
		}
	}
	return false
}

// eq evaluates the comparison a == b || a == c || ...
func eq(arg1 interface{}, arg2 ...interface{}) (bool, error) {
	fmt.Printf("EQ %v %T %v %T \n", arg1, arg1, arg2[0], arg2[0])
	v1 := toReflectValue(arg1)
	k1, err := basicKind(v1)
	if err != nil {
		return false, err
	}
	if len(arg2) == 0 {
		return false, errNoComparison
	}
	for _, arg := range arg2 {
		v2 := toReflectValue(arg)
		k2, err := basicKind(v2)
		if err != nil {
			return false, err
		}
		truth := false
		if k1 != k2 {
			if isNumber(v1) && isNumber(v2) {
				v2, err = convertTo(v2, v1.Type())
				if err != nil {
					return false, err
				}
			} else {
				return false, errBadComparison
			}
		}
		switch k1 {
		case boolKind:
			truth = v1.Bool() == v2.Bool()
		case complexKind:
			truth = v1.Complex() == v2.Complex()
		case floatKind:
			truth = v1.Float() == v2.Float()
		case intKind:
			truth = v1.Int() == v2.Int()
		case stringKind:
			truth = v1.String() == v2.String()
		case uintKind:
			truth = v1.Uint() == v2.Uint()
		default:
			panic("invalid kind")
		}
		if truth {
			return true, nil
		}
	}
	return false, nil
}

// ne evaluates the comparison a != b.
func ne(arg1, arg2 interface{}) (bool, error) {
	// != is the inverse of ==.
	equal, err := eq(arg1, arg2)
	return !equal, err
}

// lt evaluates the comparison a < b.
func lt(arg1, arg2 interface{}) (bool, error) {
	v1 := toReflectValue(arg1)
	k1, err := basicKind(v1)
	if err != nil {
		return false, err
	}
	v2 := toReflectValue(arg2)
	k2, err := basicKind(v2)
	if err != nil {
		return false, err
	}
	truth := false
	if k1 != k2 {
		if isNumber(v1) && isNumber(v2) {
			v2, err = convertTo(v2, v1.Type())
			if err != nil {
				return false, err
			}
		} else {
			return false, errBadComparison
		}
	}
	switch k1 {
	case boolKind, complexKind:
		return false, errBadComparisonType
	case floatKind:
		truth = v1.Float() < v2.Float()
	case intKind:
		truth = v1.Int() < v2.Int()
	case stringKind:
		truth = v1.String() < v2.String()
	case uintKind:
		truth = v1.Uint() < v2.Uint()
	default:
		panic("invalid kind")
	}
	return truth, nil
}

// le evaluates the comparison <= b.
func le(arg1, arg2 interface{}) (bool, error) {
	// <= is < or ==.
	lessThan, err := lt(arg1, arg2)
	if lessThan || err != nil {
		return lessThan, err
	}
	return eq(arg1, arg2)
}

// gt evaluates the comparison a > b.
func gt(arg1, arg2 interface{}) (bool, error) {
	// > is the inverse of <=.
	lessOrEqual, err := le(arg1, arg2)
	if err != nil {
		return false, err
	}
	return !lessOrEqual, nil
}

// ge evaluates the comparison a >= b.
func ge(arg1, arg2 interface{}) (bool, error) {
	// >= is the inverse of <.
	lessThan, err := lt(arg1, arg2)
	if err != nil {
		return false, err
	}
	return !lessThan, nil
}
