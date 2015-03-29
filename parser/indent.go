package parser

import (
	"errors"
)

type indent struct {
	indentType int
	prev       int
	curr       int
}

func (i *indent) setType(indentType int) {
	i.indentType = indentType
}

func (i *indent) typeName() string {
	switch i.indentType {
	case 0:
		return "undefined"
	case 1:
		return "tab"
	default:
		return "space"
	}
}

//Convert the character count to a valid indentation level.
func (i *indent) charCountToLevel(indent int) (level int, err error) {
	if i.indentType == 0 {
		return 0, errors.New("IndentType not set.")
	}
	return indent / i.indentType, nil

}

func (i *indent) setCurr(CharCount int) (level int, err error) {
	level, err = i.charCountToLevel(CharCount)
	i.prev = i.curr
	i.curr = level
	return
}

func (i *indent) level() int {
	return i.curr
}
