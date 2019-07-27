package cty_test

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestPathApply(t *testing.T) {
	tests := []struct {
		Start   cty.Value
		Path    cty.Path
		Want    cty.Value
		WantErr string
	}{
		{
			cty.StringVal("hello"),
			nil,
			cty.StringVal("hello"),
			``,
		},
		{
			cty.StringVal("hello"),
			(cty.Path)(nil).Index(cty.StringVal("boop")),
			cty.NilVal,
			`at step 0: not a map type`,
		},
		{
			cty.StringVal("hello"),
			(cty.Path)(nil).Index(cty.NumberIntVal(0)),
			cty.NilVal,
			`at step 0: not a list type`,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
			}),
			(cty.Path)(nil).Index(cty.NumberIntVal(0)),
			cty.StringVal("hello"),
			``,
		},
		{
			cty.TupleVal([]cty.Value{
				cty.StringVal("hello"),
			}),
			(cty.Path)(nil).Index(cty.NumberIntVal(0)),
			cty.StringVal("hello"),
			``,
		},
		{
			cty.ListValEmpty(cty.String),
			(cty.Path)(nil).Index(cty.NumberIntVal(0)),
			cty.NilVal,
			`at step 0: value does not have given index key`,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
			}),
			(cty.Path)(nil).Index(cty.NumberIntVal(1)),
			cty.NilVal,
			`at step 0: value does not have given index key`,
		},
		{
			cty.ListVal([]cty.Value{
				cty.StringVal("hello"),
			}),
			(cty.Path)(nil).Index(cty.NumberIntVal(0)).GetAttr("foo"),
			cty.NilVal,
			`at step 1: not an object type`,
		},
		{
			cty.ListVal([]cty.Value{
				cty.EmptyObjectVal,
			}),
			(cty.Path)(nil).Index(cty.NumberIntVal(0)).GetAttr("foo"),
			cty.NilVal,
			`at step 1: object has no attribute "foo"`,
		},
		{
			cty.NullVal(cty.List(cty.String)),
			(cty.Path)(nil).Index(cty.NumberIntVal(0)),
			cty.NilVal,
			`at step 0: cannot index a null value`,
		},
		{
			cty.NullVal(cty.Map(cty.String)),
			(cty.Path)(nil).Index(cty.NumberIntVal(0)),
			cty.NilVal,
			`at step 0: cannot index a null value`,
		},
		{
			cty.NullVal(cty.EmptyObject),
			(cty.Path)(nil).GetAttr("foo"),
			cty.NilVal,
			`at step 0: cannot access attributes on a null value`,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v %#v", test.Start, test.Path), func(t *testing.T) {
			got, gotErr := test.Path.Apply(test.Start)
			t.Logf("testing path apply\nstart: %#v\npath:  %#v", test.Start, test.Path)

			if test.WantErr != "" {
				if gotErr == nil {
					t.Fatalf("succeeded, but want error\nwant error: %s", test.WantErr)
				}

				if gotErrStr := gotErr.Error(); gotErrStr != test.WantErr {
					t.Fatalf("wrong error\ngot error:  %s\nwant error: %s", gotErrStr, test.WantErr)
				}
				return
			}

			if gotErr != nil {
				t.Fatalf("failed, but want success\ngot error: %s", gotErr.Error())
			}
			if !test.Want.RawEquals(got) {
				t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestPathEquals(t *testing.T) {
	tests := []struct {
		A, B   cty.Path
		Equal  bool
		Prefix bool
	}{
		{
			A:      nil,
			B:      nil,
			Equal:  true,
			Prefix: true,
		},
		{
			A:      cty.Path{},
			B:      cty.Path{},
			Equal:  true,
			Prefix: true,
		},
		{
			A: cty.Path{nil},
			B: cty.Path{cty.GetAttrStep{Name: "attr"}},
		},
		{
			A: cty.Path{
				cty.GetAttrStep{Name: "attr"},
				cty.IndexStep{Key: cty.UnknownVal(cty.String)},
				cty.GetAttrStep{Name: "attr"},
			},
			B: cty.Path{
				cty.GetAttrStep{Name: "attr"},
				cty.IndexStep{Key: cty.StringVal("key")},
				cty.GetAttrStep{Name: "attr"},
			},
		},
		{
			A: cty.Path{
				cty.GetAttrStep{Name: "attr"},
				cty.IndexStep{Key: cty.ListVal([]cty.Value{cty.UnknownVal(cty.String)})},
				cty.GetAttrStep{Name: "attr"},
			},
			B: cty.Path{
				cty.GetAttrStep{Name: "attr"},
				cty.IndexStep{Key: cty.ListVal([]cty.Value{cty.StringVal("known")})},
				cty.GetAttrStep{Name: "attr"},
			},
		},
		{
			A: cty.Path{
				cty.GetAttrStep{Name: "attr"},
				cty.IndexStep{Key: cty.UnknownVal(cty.String)},
			},
			B: cty.Path{
				cty.GetAttrStep{Name: "attr"},
				cty.IndexStep{Key: cty.StringVal("known")},
				cty.GetAttrStep{Name: "attr"},
			},
		},
		{
			A: cty.Path{
				cty.GetAttrStep{Name: "attr"},
				cty.IndexStep{Key: cty.StringVal("known")},
			},
			B: cty.Path{
				cty.GetAttrStep{Name: "attr"},
				cty.IndexStep{Key: cty.StringVal("known")},
				cty.GetAttrStep{Name: "attr"},
			},
		},
		{
			A: cty.Path{
				cty.GetAttrStep{Name: "attr"},
				cty.IndexStep{Key: cty.StringVal("known")},
				cty.GetAttrStep{Name: "attr"},
			},
			B: cty.Path{
				cty.GetAttrStep{Name: "attr"},
				cty.IndexStep{Key: cty.StringVal("known")},
			},
			Prefix: true,
		},
		{
			A: cty.Path{
				cty.GetAttrStep{Name: "attr"},
				cty.IndexStep{Key: cty.UnknownVal(cty.String)},
			},
			B: cty.Path{
				cty.GetAttrStep{Name: "attr"},
				cty.IndexStep{Key: cty.UnknownVal(cty.String)},
			},
			Prefix: true,
			Equal:  true,
		},
		{
			A: cty.Path{
				cty.GetAttrStep{Name: "attr"},
				cty.IndexStep{Key: cty.NumberFloatVal(0)},
				cty.GetAttrStep{Name: "attr"},
			},
			B: cty.Path{
				cty.GetAttrStep{Name: "attr"},
				cty.IndexStep{Key: cty.NumberIntVal(0)},
				cty.GetAttrStep{Name: "attr"},
			},
			Equal:  true,
			Prefix: true,
		},
		{
			A: cty.Path{
				cty.GetAttrStep{Name: "attr"},
				cty.IndexStep{Key: cty.NumberIntVal(1)},
				cty.GetAttrStep{Name: "attr"},
			},
			B: cty.Path{
				cty.GetAttrStep{Name: "attr"},
				cty.IndexStep{Key: cty.NumberIntVal(0)},
				cty.GetAttrStep{Name: "attr"},
			},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d-%#v", i, test.A), func(t *testing.T) {
			if test.Equal != test.A.Equals(test.B) {
				t.Fatalf("%#v.Equals(%#v) != %t", test.A, test.B, test.Equal)
			}
			if test.Prefix != test.A.HasPrefix(test.B) {
				t.Fatalf("%#v.HasPrefix(%#v) != %t", test.A, test.B, test.Prefix)
			}
		})
	}
}
