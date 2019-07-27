package yaml

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestUnmarshal(t *testing.T) {
	tests := map[string]struct {
		converter *Converter
		src       string
		ty        cty.Type
		want      cty.Value
		wantErr   string
	}{
		"single string doublequote": {
			Standard,
			`"hello"`,
			cty.String,
			cty.StringVal("hello"),
			``,
		},
		"single True string doublequote": {
			Standard,
			`"True"`,
			cty.String,
			cty.StringVal("True"),
			``,
		},
		"single .INF string doublequote": {
			Standard,
			`".INF"`,
			cty.String,
			cty.StringVal(".INF"),
			``,
		},
		"single string singlequote": {
			Standard,
			`'hello'`,
			cty.String,
			cty.StringVal("hello"),
			``,
		},
		"single False string singlequote": {
			Standard,
			`'False'`,
			cty.String,
			cty.StringVal("False"),
			``,
		},
		"single NULL string singlequote": {
			Standard,
			`'NULL'`,
			cty.String,
			cty.StringVal("NULL"),
			``,
		},
		"single string literal": {
			Standard,
			"|\n  hello\n  world",
			cty.String,
			cty.StringVal("hello\nworld"),
			``,
		},
		"single string folded": {
			Standard,
			">\n  hello\n  world",
			cty.String,
			cty.StringVal("hello world"),
			``,
		},
		"single string implied": {
			Standard,
			`hello`,
			cty.String,
			cty.StringVal("hello"),
			``,
		},
		"single string implied not merge": {
			Standard,
			`<<`,
			cty.String,
			cty.StringVal("<<"),
			``,
		},
		"single string short tag": {
			Standard,
			`!!str true`,
			cty.String,
			cty.StringVal("true"),
			``,
		},
		"single string long tag": {
			Standard,
			`!<tag:yaml.org,2002:str> true`,
			cty.String,
			cty.StringVal("true"),
			``,
		},
		"single bool implied true": {
			Standard,
			`true`,
			cty.Bool,
			cty.True,
			``,
		},
		"single bool implied converted to string": {
			Standard,
			`yes`,                 // YAML defines this as being a boolean true...
			cty.String,            // but we want a string result...
			cty.StringVal("true"), // so the boolean is converted to string using cty's rules
			``,
		},
		"single bool implied false": {
			Standard,
			`false`,
			cty.Bool,
			cty.False,
			``,
		},
		"single bool short tag": {
			Standard,
			`!!bool true`,
			cty.Bool,
			cty.True,
			``,
		},
		"single bool long tag": {
			Standard,
			`!<tag:yaml.org,2002:bool> true`,
			cty.Bool,
			cty.True,
			``,
		},
		"single bool short tag invalid": {
			Standard,
			`!!bool bananas`,
			cty.Bool,
			cty.NilVal,
			`cannot parse "bananas" as tag:yaml.org,2002:bool`,
		},
		"single float implied by prefix": {
			Standard,
			`.5`,
			cty.Number,
			cty.NumberFloatVal(0.5),
			``,
		},
		"single float implied by parsability": {
			Standard,
			`1.5`,
			cty.Number,
			cty.NumberFloatVal(1.5),
			``,
		},
		"single float short tag": {
			Standard,
			`!!float 1.5`,
			cty.Number,
			cty.NumberFloatVal(1.5),
			``,
		},
		"single int implied by parsability": {
			Standard,
			`12`,
			cty.Number,
			cty.NumberIntVal(12),
			``,
		},
		"single int negative implied by parsability": {
			Standard,
			`-12`,
			cty.Number,
			cty.NumberIntVal(-12),
			``,
		},
		"single int short tag": {
			Standard,
			`!!int 1`,
			cty.Number,
			cty.NumberIntVal(1),
			``,
		},
		"single positive infinity implied": {
			Standard,
			`+Inf`,
			cty.Number,
			cty.PositiveInfinity,
			``,
		},
		"single negative infinity implied": {
			Standard,
			`-Inf`,
			cty.Number,
			cty.NegativeInfinity,
			``,
		},
		"single NaN implied": {
			Standard,
			`.NaN`,
			cty.Number,
			cty.NilVal,
			`floating point NaN is not supported`,
		},
		"single timestamp implied": {
			Standard,
			`2006-1-2`,
			cty.String,
			cty.StringVal("2006-01-02T00:00:00Z"),
			``,
		},
		"single timestamp short tag": {
			Standard,
			`!!timestamp 2006-1-2`,
			cty.String,
			cty.StringVal("2006-01-02T00:00:00Z"),
			``,
		},
		"single binary short tag": {
			Standard,
			`!!binary 'aGVsbG8='`,
			cty.String,
			cty.StringVal("aGVsbG8="),
			``,
		},
		"single binary short tag invalid base64": {
			Standard,
			`!!binary '>>>>>>>>>'`,
			cty.String,
			cty.NilVal,
			`cannot parse ">>>>>>>>>" as tag:yaml.org,2002:binary: not valid base64`,
		},
		"single null implied": {
			Standard,
			`null`,
			cty.String,
			cty.NullVal(cty.String),
			``,
		},
		"single scalar invalid tag": {
			Standard,
			`!!nope foo`,
			cty.String,
			cty.NilVal,
			`unsupported tag "tag:yaml.org,2002:nope"`,
		},

		"mapping empty flow mode": {
			Standard,
			`{}`,
			cty.Map(cty.String),
			cty.MapValEmpty(cty.String),
			``,
		},
		"mapping flow mode": {
			Standard,
			`{a: 1, b: true}`,
			cty.Object(map[string]cty.Type{
				"a": cty.Number,
				"b": cty.Bool,
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NumberIntVal(1),
				"b": cty.True,
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
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NumberIntVal(1),
				"b": cty.True,
			}),
			``,
		},

		"mapping with sequence multi-line mode": {
			Standard,
			`
a: 1
b:
  - foo
  - <<
  - baz
`,
			cty.Object(map[string]cty.Type{
				"a": cty.Number,
				"b": cty.List(cty.String),
			}),
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.NumberIntVal(1),
				"b": cty.ListVal([]cty.Value{
					cty.StringVal("foo"),
					cty.StringVal("<<"),
					cty.StringVal("baz"),
				}),
			}),
			``,
		},
		"sequence empty flow mode": {
			Standard,
			`[]`,
			cty.Set(cty.String),
			cty.SetValEmpty(cty.String),
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
			cty.TupleVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.True,
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
			cty.TupleVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("<<"),
				cty.True,
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
			cty.Map(cty.List(cty.String)),
			cty.MapVal(map[string]cty.Value{
				"foo": cty.ListVal([]cty.Value{
					cty.StringVal("x"),
				}),
				"bar": cty.ListVal([]cty.Value{
					cty.StringVal("x"),
				}),
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
			cty.DynamicPseudoType,
			cty.NilVal,
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
			cty.Map(cty.Map(cty.String)),
			cty.MapVal(map[string]cty.Value{
				"foo": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("b"),
				}),
				"bar": cty.MapVal(map[string]cty.Value{
					"a": cty.StringVal("b"),
					"c": cty.StringVal("d"),
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
			cty.List(cty.String),
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("a"),
			}),
			``,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, gotErr := test.converter.Unmarshal([]byte(test.src), test.ty)

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
			if !test.want.RawEquals(got) {
				t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}
