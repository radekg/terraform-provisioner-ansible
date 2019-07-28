package yaml

import (
	"bytes"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestMarshal(t *testing.T) {
	flowStyle := NewConverter(&ConverterConfig{
		EncodeAsFlow: true,
	})

	tests := []struct {
		conv *Converter
		val  cty.Value
		want string
	}{
		{
			Standard,
			cty.True,
			"true\n...\n",
		},
		{
			Standard,
			cty.StringVal(""),
			"\"\"\n",
		},
		{
			Standard,
			cty.StringVal("hello"),
			"\"hello\"\n",
		},
		{
			Standard,
			cty.StringVal("hello\nworld"),
			"|-\n  hello\n  world\n",
		},
		{
			Standard,
			cty.Zero,
			"0\n...\n",
		},
		{
			Standard,
			cty.PositiveInfinity,
			"+Inf\n...\n",
		},
		{
			Standard,
			cty.NegativeInfinity,
			"-Inf\n...\n",
		},
		{
			Standard,
			cty.EmptyObjectVal,
			"{}\n",
		},
		{
			flowStyle,
			cty.EmptyObjectVal,
			"{}\n",
		},
		{
			Standard,
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("b"),
			}),
			"\"a\": \"b\"\n",
		},
		{
			flowStyle,
			cty.ObjectVal(map[string]cty.Value{
				"a": cty.StringVal("b"),
			}),
			"{\"a\": \"b\"}\n",
		},
		{
			Standard,
			cty.MapVal(map[string]cty.Value{
				"a": cty.StringVal("b"),
			}),
			"\"a\": \"b\"\n",
		},
		{
			flowStyle,
			cty.MapVal(map[string]cty.Value{
				"a": cty.StringVal("b"),
			}),
			"{\"a\": \"b\"}\n",
		},
		{
			Standard,
			cty.EmptyTupleVal,
			"[]\n",
		},
		{
			flowStyle,
			cty.EmptyTupleVal,
			"[]\n",
		},
		{
			Standard,
			cty.TupleVal([]cty.Value{
				cty.StringVal("b"),
			}),
			"- \"b\"\n",
		},
		{
			flowStyle,
			cty.TupleVal([]cty.Value{
				cty.StringVal("b"),
			}),
			"[\"b\"]\n",
		},
		{
			Standard,
			cty.ListVal([]cty.Value{
				cty.StringVal("b"),
			}),
			"- \"b\"\n",
		},
		{
			flowStyle,
			cty.ListVal([]cty.Value{
				cty.StringVal("b"),
			}),
			"[\"b\"]\n",
		},
		{
			Standard,
			cty.SetVal([]cty.Value{
				cty.StringVal("b"),
			}),
			"- \"b\"\n",
		},
		{
			flowStyle,
			cty.SetVal([]cty.Value{
				cty.StringVal("b"),
			}),
			"[\"b\"]\n",
		},
		{
			flowStyle,
			cty.NullVal(cty.String),
			"null\n...\n",
		},
	}

	for _, test := range tests {
		var suffix string
		if test.conv == flowStyle {
			suffix = " in flow style"
		}
		t.Run(test.val.GoString()+suffix, func(t *testing.T) {
			got, err := test.conv.Marshal(test.val)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !bytes.Equal(got, []byte(test.want)) {
				t.Errorf("wrong result\ngot:\n%s\n\nwant:\n%s", got, test.want)
			}
		})
	}
}
