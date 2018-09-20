package xmlstruct

import (
	"fmt"
	"reflect"
)

func ParseStruct(i interface{}) (*XMLRoot, error) {
	ret := &XMLRoot{}
	val := reflect.ValueOf(i)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return ret, fmt.Errorf("Interface is not a struct")
	}

	if getXMLName(val).Local == "" {
		return nil, fmt.Errorf("Invalid XML struct")
	}

	ret.Ele = &XMLEle{Val: val, pos: 1, prnt: ret}

	return ret, nil
}

func MustParseStruct(i interface{}) *XMLRoot {
	ret, err := ParseStruct(i)

	if err != nil {
		panic(err)
	}

	return ret
}
