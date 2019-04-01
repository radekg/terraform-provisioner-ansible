package convert

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestUnify(t *testing.T) {
	tests := []struct {
		Input           []cty.Type
		WantType        cty.Type
		WantConversions []bool
	}{
		{
			[]cty.Type{},
			cty.NilType,
			nil,
		},
		{
			[]cty.Type{cty.String},
			cty.String,
			[]bool{false},
		},
		{
			[]cty.Type{cty.Number},
			cty.Number,
			[]bool{false},
		},
		{
			[]cty.Type{cty.Number, cty.Number},
			cty.Number,
			[]bool{false, false},
		},
		{
			[]cty.Type{cty.Number, cty.String},
			cty.String,
			[]bool{true, false},
		},
		{
			[]cty.Type{cty.String, cty.Number},
			cty.String,
			[]bool{false, true},
		},
		{
			[]cty.Type{cty.Bool, cty.String, cty.Number},
			cty.String,
			[]bool{true, false, true},
		},
		{
			[]cty.Type{cty.Bool, cty.Number},
			cty.NilType,
			nil,
		},
		{
			[]cty.Type{
				cty.Object(map[string]cty.Type{"foo": cty.String}),
				cty.Object(map[string]cty.Type{"foo": cty.String}),
			},
			cty.Object(map[string]cty.Type{"foo": cty.String}),
			[]bool{false, false},
		},
		{
			[]cty.Type{
				cty.Object(map[string]cty.Type{"foo": cty.String}),
				cty.Object(map[string]cty.Type{"foo": cty.Number}),
			},
			cty.Object(map[string]cty.Type{"foo": cty.String}),
			[]bool{false, true},
		},
		{
			[]cty.Type{
				cty.Object(map[string]cty.Type{"foo": cty.String}),
				cty.Object(map[string]cty.Type{"bar": cty.Number}),
			},
			cty.Map(cty.String),
			[]bool{true, true},
		},
		{
			[]cty.Type{
				cty.Object(map[string]cty.Type{"foo": cty.String}),
				cty.EmptyObject,
			},
			cty.Map(cty.String),
			[]bool{true, true},
		},
		{
			[]cty.Type{
				cty.Object(map[string]cty.Type{"foo": cty.Bool}),
				cty.Object(map[string]cty.Type{"bar": cty.Number}),
			},
			cty.NilType,
			nil,
		},
		{
			[]cty.Type{
				cty.Object(map[string]cty.Type{"foo": cty.Bool}),
				cty.Object(map[string]cty.Type{"foo": cty.Number}),
			},
			cty.NilType,
			nil,
		},
		{
			[]cty.Type{
				cty.Tuple([]cty.Type{cty.String}),
				cty.Tuple([]cty.Type{cty.String}),
			},
			cty.Tuple([]cty.Type{cty.String}),
			[]bool{false, false},
		},
		{
			[]cty.Type{
				cty.Tuple([]cty.Type{cty.String}),
				cty.Tuple([]cty.Type{cty.Number}),
			},
			cty.Tuple([]cty.Type{cty.String}),
			[]bool{false, true},
		},
		{
			[]cty.Type{
				cty.Tuple([]cty.Type{cty.String}),
				cty.Tuple([]cty.Type{cty.String, cty.Number}),
			},
			cty.List(cty.String),
			[]bool{true, true},
		},
		{
			[]cty.Type{
				cty.Tuple([]cty.Type{cty.String}),
				cty.EmptyTuple,
			},
			cty.List(cty.String),
			[]bool{true, true},
		},
		{
			[]cty.Type{
				cty.Tuple([]cty.Type{cty.Bool}),
				cty.Tuple([]cty.Type{cty.Number}),
			},
			cty.NilType,
			nil,
		},
		{
			[]cty.Type{
				cty.DynamicPseudoType,
				cty.Tuple([]cty.Type{cty.Number}),
			},
			cty.DynamicPseudoType,
			[]bool{true, true},
		},
		{
			[]cty.Type{
				cty.DynamicPseudoType,
				cty.Object(map[string]cty.Type{"num": cty.Number}),
			},
			cty.DynamicPseudoType,
			[]bool{true, true},
		},
		{
			[]cty.Type{
				cty.Tuple([]cty.Type{cty.Number}),
				cty.DynamicPseudoType,
				cty.Object(map[string]cty.Type{"num": cty.Number}),
			},
			cty.NilType,
			nil,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v", test.Input), func(t *testing.T) {
			gotType, gotConvs := Unify(test.Input)
			if gotType == cty.NilType && test.WantType == cty.NilType {
				// okay!
			} else if ((gotType == cty.NilType) != (test.WantType == cty.NilType)) || !test.WantType.Equals(gotType) {
				t.Errorf("wrong result type\ngot:  %#v\nwant: %#v", gotType, test.WantType)
			}

			gotConvsNil := gotConvs == nil
			wantConvsNil := test.WantConversions == nil
			if gotConvsNil && wantConvsNil {
				// Success!
				return
			}

			if gotConvsNil != wantConvsNil {
				if gotConvsNil {
					t.Fatalf("got nil conversions; want %#v", test.WantConversions)
				} else {
					t.Fatalf("got conversions; want nil")
				}
			}

			gotConvsBool := make([]bool, len(gotConvs))
			for i, f := range gotConvs {
				gotConvsBool[i] = f != nil
			}

			if !reflect.DeepEqual(gotConvsBool, test.WantConversions) {
				t.Fatalf(
					"wrong conversions\ngot:  %#v\nwant: %#v",
					gotConvsBool, test.WantConversions,
				)
			}
		})
	}
}
