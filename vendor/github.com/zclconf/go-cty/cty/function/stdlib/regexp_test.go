package stdlib

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestRegex(t *testing.T) {
	tests := []struct {
		Pattern cty.Value
		String  cty.Value
		Want    cty.Value
	}{
		{
			cty.StringVal("[a-z]+"),
			cty.StringVal("135abc456def789"),
			cty.StringVal("abc"),
		},
		{
			cty.StringVal("([0-9]*)([a-z]*)"),
			cty.StringVal("135abc456def"),
			cty.TupleVal([]cty.Value{
				cty.StringVal("135"),
				cty.StringVal("abc"),
			}),
		},
		{
			cty.StringVal(`^(?:(?P<scheme>[^:/?#]+):)?(?://(?P<authority>[^/?#]*))?(?P<path>[^?#]*)(?:\?(?P<query>[^#]*))?(?:#(?P<fragment>.*))?`),
			cty.StringVal("http://www.ics.uci.edu/pub/ietf/uri/#Related"),
			cty.ObjectVal(map[string]cty.Value{
				"scheme":    cty.StringVal("http"),
				"authority": cty.StringVal("www.ics.uci.edu"),
				"path":      cty.StringVal("/pub/ietf/uri/"),
				"query":     cty.NullVal(cty.String), // query portion isn't present at all, because there's no ?
				"fragment":  cty.StringVal("Related"),
			}),
		},
		{
			cty.StringVal("([0-9]*)([a-z]*)"),
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.Tuple([]cty.Type{
				cty.String,
				cty.String,
			})),
		},
		{
			cty.StringVal("(?P<num>[0-9]*)"),
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.Object(map[string]cty.Type{
				"num": cty.String,
			})),
		},
		{
			cty.UnknownVal(cty.String),
			cty.StringVal("135abc456def"),
			cty.DynamicVal,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Regex(%#v, %#v)", test.Pattern, test.String), func(t *testing.T) {
			got, err := Regex(test.Pattern, test.String)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf(
					"wrong result\npattern: %#v\nstring:  %#v\ngot:     %#v\nwant:    %#v",
					test.Pattern, test.String, got, test.Want,
				)
			}
		})
	}
}

func TestRegexAll(t *testing.T) {
	tests := []struct {
		Pattern cty.Value
		String  cty.Value
		Want    cty.Value
	}{
		{
			cty.StringVal("[a-z]+"),
			cty.StringVal("135abc456def789"),
			cty.ListVal([]cty.Value{
				cty.StringVal("abc"),
				cty.StringVal("def"),
			}),
		},
		{
			cty.StringVal("([0-9]*)([a-z]*)"),
			cty.StringVal("135abc456def"),
			cty.ListVal([]cty.Value{
				cty.TupleVal([]cty.Value{
					cty.StringVal("135"),
					cty.StringVal("abc"),
				}),
				cty.TupleVal([]cty.Value{
					cty.StringVal("456"),
					cty.StringVal("def"),
				}),
			}),
		},
		{
			cty.StringVal(`^(?:(?P<scheme>[^:/?#]+):)?(?://(?P<authority>[^/?#]*))?(?P<path>[^?#]*)(?:\?(?P<query>[^#]*))?(?:#(?P<fragment>.*))?`),
			cty.StringVal("http://www.ics.uci.edu/pub/ietf/uri/#Related"),
			cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"scheme":    cty.StringVal("http"),
					"authority": cty.StringVal("www.ics.uci.edu"),
					"path":      cty.StringVal("/pub/ietf/uri/"),
					"query":     cty.NullVal(cty.String), // query portion isn't present at all, because there's no ?
					"fragment":  cty.StringVal("Related"),
				}),
			}),
		},
		{
			cty.StringVal("([0-9]*)([a-z]*)"),
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.List(cty.Tuple([]cty.Type{
				cty.String,
				cty.String,
			}))),
		},
		{
			cty.StringVal("(?P<num>[0-9]*)"),
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.List(cty.Object(map[string]cty.Type{
				"num": cty.String,
			}))),
		},
		{
			cty.UnknownVal(cty.String),
			cty.StringVal("135abc456def"),
			cty.UnknownVal(cty.List(cty.DynamicPseudoType)),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("RegexAll(%#v, %#v)", test.Pattern, test.String), func(t *testing.T) {
			got, err := RegexAll(test.Pattern, test.String)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf(
					"wrong result\npattern: %#v\nstring:  %#v\ngot:     %#v\nwant:    %#v",
					test.Pattern, test.String, got, test.Want,
				)
			}
		})
	}
}
