package yaml

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestImpliedType(t *testing.T) {
	tests := map[string]struct {
		converter *Converter
		src       string
		want      cty.Type
		wantErr   string
	}{
		"single string doublequote": {
			Standard,
			`"hello"`,
			cty.String,
			``,
		},
		"single string singlequote": {
			Standard,
			`'hello'`,
			cty.String,
			``,
		},
		"single string implied": {
			Standard,
			`hello`,
			cty.String,
			``,
		},
		"single string implied not merge": {
			Standard,
			`<<`,
			cty.String,
			``,
		},
		"single string short tag": {
			Standard,
			`!!str true`,
			cty.String,
			``,
		},
		"single string long tag": {
			Standard,
			`!<tag:yaml.org,2002:str> true`,
			cty.String,
			``,
		},
		"single bool implied true": {
			Standard,
			`true`,
			cty.Bool,
			``,
		},
		"single bool implied false": {
			Standard,
			`false`,
			cty.Bool,
			``,
		},
		"single bool short tag": {
			Standard,
			`!!bool true`,
			cty.Bool,
			``,
		},
		"single bool long tag": {
			Standard,
			`!<tag:yaml.org,2002:bool> true`,
			cty.Bool,
			``,
		},
		"single bool short tag invalid": {
			Standard,
			`!!bool bananas`,
			cty.NilType,
			`cannot parse "bananas" as tag:yaml.org,2002:bool`,
		},
		"single float implied by prefix": {
			Standard,
			`.2`,
			cty.Number,
			``,
		},
		"single float implied by parsability": {
			Standard,
			`1.2`,
			cty.Number,
			``,
		},
		"single float short tag": {
			Standard,
			`!!float 1.2`,
			cty.Number,
			``,
		},
		"single int implied by parsability": {
			Standard,
			`12`,
			cty.Number,
			``,
		},
		"single int negative implied by parsability": {
			Standard,
			`-12`,
			cty.Number,
			``,
		},
		"single int short tag": {
			Standard,
			`!!int 1`,
			cty.Number,
			``,
		},
		"single positive infinity implied": {
			Standard,
			`+Inf`,
			cty.Number,
			``,
		},
		"single negative infinity implied": {
			Standard,
			`-Inf`,
			cty.Number,
			``,
		},
		"single NaN implied": {
			Standard,
			`.NaN`,
			cty.NilType,
			`floating point NaN is not supported`,
		},
		"single timestamp implied": {
			Standard,
			`2006-1-2`,
			cty.String,
			``,
		},
		"single timestamp short tag": {
			Standard,
			`!!timestamp 2006-1-2`,
			cty.String,
			``,
		},
		"single binary short tag": {
			Standard,
			`!!binary 'aGVsbG8='`,
			cty.String,
			``,
		},
		"single binary short tag invalid base64": {
			Standard,
			`!!binary '>>>>>>>>>'`,
			cty.NilType,
			`cannot parse ">>>>>>>>>" as tag:yaml.org,2002:binary: not valid base64`,
		},
		"single null implied": {
			Standard,
			`null`,
			cty.DynamicPseudoType,
			``,
		},
		"single scalar invalid tag": {
			Standard,
			`!!nope foo`,
			cty.NilType,
			`unsupported tag "tag:yaml.org,2002:nope"`,
		},

		"mapping empty flow mode": {
			Standard,
			`{}`,
			cty.EmptyObject,
			``,
		},
		"mapping flow mode": {
			Standard,
			`{a: 1, b: true}`,
			cty.Object(map[string]cty.Type{
				"a": cty.Number,
				"b": cty.Bool,
			}),
			``,
		},
		"mapping multi-line mode": {
			Standard,
			`
a: 1
b: true
`,
			cty.Object(map[string]cty.Type{
				"a": cty.Number,
				"b": cty.Bool,
			}),
			``,
		},

		"mapping with sequence multi-line mode": {
			Standard,
			`
a: 1
b:
  - foo
  - bar
  - baz
`,
			cty.Object(map[string]cty.Type{
				"a": cty.Number,
				"b": cty.Tuple([]cty.Type{
					cty.String,
					cty.String,
					cty.String,
				}),
			}),
			``,
		},
		"sequence empty flow mode": {
			Standard,
			`[]`,
			cty.EmptyTuple,
			``,
		},
		"sequence flow mode": {
			Standard,
			`[a, b, true]`,
			cty.Tuple([]cty.Type{
				cty.String,
				cty.String,
				cty.Bool,
			}),
			``,
		},
		"sequence multi-line mode": {
			Standard,
			`
- a
- <<
- true
`,
			cty.Tuple([]cty.Type{
				cty.String,
				cty.String,
				cty.Bool,
			}),
			``,
		},

		"alias": {
			Standard,
			`
foo: &bar
  - x
bar: *bar
`,
			cty.Object(map[string]cty.Type{
				"foo": cty.Tuple([]cty.Type{cty.String}),
				"bar": cty.Tuple([]cty.Type{cty.String}),
			}),
			``,
		},
		"alias cyclic": {
			Standard,
			`
foo: &bar
  - x
  - *bar
`,
			cty.NilType,
			`on line 3, column 5: cannot refer to anchor "bar" from inside its own definition`,
		},
		"alias merge": {
			Standard,
			`
foo: &bar
  a: b
bar:
  <<: *bar
  c: d
`,
			cty.Object(map[string]cty.Type{
				"foo": cty.Object(map[string]cty.Type{
					"a": cty.String,
				}),
				"bar": cty.Object(map[string]cty.Type{
					"a": cty.String,
					"c": cty.String,
				}),
			}),
			``,
		},
		"alias scalar": {
			Standard,
			`
- &foo a
- b
- *foo
`,
			cty.Tuple([]cty.Type{
				cty.String,
				cty.String,
				cty.String,
			}),
			``,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, gotErr := test.converter.ImpliedType([]byte(test.src))

			if gotErr != nil {
				if test.wantErr == "" {
					t.Fatalf("wrong error\ngot:  %s\nwant: (no error)", gotErr.Error())
				}
				if got, want := gotErr.Error(), test.wantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			}

			if test.wantErr != "" {
				t.Fatalf("wrong error\ngot:  (no error)\nwant: %s", test.wantErr)
			}
			if !test.want.Equals(got) {
				t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}
