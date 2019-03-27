package convert

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestMismatchMessage(t *testing.T) {
	tests := []struct {
		GotType, WantType cty.Type
		WantMsg           string
	}{
		{
			cty.Bool,
			cty.Number,
			`number required`,
		},
		{
			cty.EmptyObject,
			cty.Object(map[string]cty.Type{
				"foo": cty.String,
			}),
			`attribute "foo" is required`,
		},
		{
			cty.EmptyObject,
			cty.Object(map[string]cty.Type{
				"foo": cty.String,
				"bar": cty.String,
			}),
			`attributes "bar" and "foo" are required`,
		},
		{
			cty.EmptyObject,
			cty.Object(map[string]cty.Type{
				"foo": cty.String,
				"bar": cty.String,
				"baz": cty.String,
			}),
			`attributes "bar", "baz", and "foo" are required`,
		},
		{
			cty.EmptyObject,
			cty.List(cty.Object(map[string]cty.Type{
				"foo": cty.String,
				"bar": cty.String,
				"baz": cty.String,
			})),
			`list of object required`,
		},
		{
			cty.List(cty.String),
			cty.List(cty.Object(map[string]cty.Type{
				"foo": cty.String,
			})),
			`incorrect list element type: object required`,
		},
		{
			cty.List(cty.EmptyObject),
			cty.List(cty.Object(map[string]cty.Type{
				"foo": cty.String,
			})),
			`incorrect list element type: attribute "foo" is required`,
		},
		{
			cty.Tuple([]cty.Type{cty.EmptyObject}),
			cty.List(cty.Object(map[string]cty.Type{
				"foo": cty.String,
			})),
			`element 0: attribute "foo" is required`,
		},
		{
			cty.List(cty.EmptyObject),
			cty.Set(cty.Object(map[string]cty.Type{
				"foo": cty.String,
			})),
			`incorrect set element type: attribute "foo" is required`,
		},
		{
			cty.Tuple([]cty.Type{cty.EmptyObject}),
			cty.Set(cty.Object(map[string]cty.Type{
				"foo": cty.String,
			})),
			`element 0: attribute "foo" is required`,
		},
		{
			cty.Map(cty.EmptyObject),
			cty.Map(cty.Object(map[string]cty.Type{
				"foo": cty.String,
			})),
			`incorrect map element type: attribute "foo" is required`,
		},
		{
			cty.Object(map[string]cty.Type{"boop": cty.EmptyObject}),
			cty.Map(cty.Object(map[string]cty.Type{
				"foo": cty.String,
			})),
			`element "boop": attribute "foo" is required`,
		},
		{
			cty.Tuple([]cty.Type{cty.EmptyObject, cty.EmptyTuple}),
			cty.List(cty.DynamicPseudoType),
			`all list elements must have the same type`,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v but want %#v", test.GotType, test.WantType), func(t *testing.T) {
			got := MismatchMessage(test.GotType, test.WantType)
			if got != test.WantMsg {
				t.Errorf("wrong message\ngot type:  %#v\nwant type: %#v\ngot message:  %s\nwant message: %s", test.GotType, test.WantType, got, test.WantMsg)
			}
		})
	}
}
