package cty

import (
	"fmt"
	"math/big"
	"testing"
)

func TestSetHashBytes(t *testing.T) {
	tests := []struct {
		value Value
		want  string
	}{
		{
			UnknownVal(Number),
			"?",
		},
		{
			UnknownVal(String),
			"?",
		},
		{
			NullVal(Number),
			"~",
		},
		{
			NullVal(String),
			"~",
		},
		{
			DynamicVal,
			"?",
		},
		{
			NumberVal(big.NewFloat(12)),
			"12",
		},
		{
			StringVal(""),
			`""`,
		},
		{
			StringVal("pizza"),
			`"pizza"`,
		},
		{
			True,
			"T",
		},
		{
			False,
			"F",
		},
		{
			ListValEmpty(Bool),
			"[]",
		},
		{
			ListValEmpty(DynamicPseudoType),
			"[]",
		},
		{
			ListVal([]Value{True, False}),
			"[T;F;]",
		},
		{
			ListVal([]Value{UnknownVal(Bool)}),
			"[?;]",
		},
		{
			ListVal([]Value{ListValEmpty(Bool)}),
			"[[];]",
		},
		{
			MapValEmpty(Bool),
			"{}",
		},
		{
			MapVal(map[string]Value{"true": True, "false": False}),
			`{"false":F;"true":T;}`,
		},
		{
			MapVal(map[string]Value{"true": True, "unknown": UnknownVal(Bool), "dynamic": DynamicVal}),
			`{"dynamic":?;"true":T;"unknown":?;}`,
		},
		{
			SetValEmpty(Bool),
			"[]",
		},
		{
			SetVal([]Value{True, True, False}),
			"[F;T;]",
		},
		{
			SetVal([]Value{UnknownVal(Bool), UnknownVal(Bool)}),
			"[?;?;]", // unknowns are never equal, so we can have multiple of them
		},
		{
			EmptyObjectVal,
			"<>",
		},
		{
			ObjectVal(map[string]Value{
				"name": StringVal("ermintrude"),
				"age":  NumberVal(big.NewFloat(54)),
			}),
			`<54;"ermintrude";>`,
		},
		{
			EmptyTupleVal,
			"<>",
		},
		{
			TupleVal([]Value{
				StringVal("ermintrude"),
				NumberVal(big.NewFloat(54)),
			}),
			`<"ermintrude";54;>`,
		},
	}

	for _, test := range tests {
		t.Run(test.value.GoString(), func(t *testing.T) {
			got := string(makeSetHashBytes(test.value))
			if got != test.want {
				t.Errorf("wrong result\ngot:  %s\nwant: %s", got, test.want)
			}
		})
	}
}

func TestSetOrder(t *testing.T) {
	tests := []struct {
		a, b Value
		want bool
	}{
		// Strings sort lexicographically (this is a compatibility constraint)
		{
			StringVal("a"),
			StringVal("b"),
			true,
		},
		{
			StringVal("b"),
			StringVal("a"),
			false,
		},
		{
			UnknownVal(String),
			StringVal("a"),
			false,
		},
		{
			StringVal("a"),
			UnknownVal(String),
			true,
		},

		// Numbers sort numerically (this is a compatibility constraint)
		{
			Zero,
			NumberIntVal(1),
			true,
		},
		{
			NumberIntVal(1),
			Zero,
			false,
		},

		// Booleans sort false before true (this is a compatibility constraint)
		{
			False,
			True,
			true,
		},
		{
			True,
			False,
			false,
		},

		// Unknown and Null values push to the end of a sort (this is a compatibility constraint)
		{
			UnknownVal(String),
			UnknownVal(String),
			false, // no defined ordering
		},
		{
			NullVal(String),
			StringVal("a"),
			false,
		},
		{
			StringVal("a"),
			NullVal(String),
			true,
		},
		{
			UnknownVal(String),
			NullVal(String),
			true,
		},
		{
			NullVal(String),
			UnknownVal(String),
			false,
		},

		// All other types just use an arbitrary fallback sort. These results
		// are _not_ compatibility constraints but we are testing them here
		// to verify that the result is consistent between runs for a
		// specific version of cty.
		{
			ListValEmpty(String),
			ListVal([]Value{StringVal("boop")}),
			false,
		},
		{
			ListVal([]Value{StringVal("boop")}),
			ListValEmpty(String),
			true,
		},
		{
			SetValEmpty(String),
			SetVal([]Value{StringVal("boop")}),
			false,
		},
		{
			SetVal([]Value{StringVal("boop")}),
			SetValEmpty(String),
			true,
		},
		{
			MapValEmpty(String),
			MapVal(map[string]Value{"blah": StringVal("boop")}),
			false,
		},
		{
			MapVal(map[string]Value{"blah": StringVal("boop")}),
			MapValEmpty(String),
			true,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v < %#v", test.a, test.b), func(t *testing.T) {
			rules := setRules{test.a.Type()} // both values are assumed to have the same type
			got := rules.Less(test.a.v, test.b.v)
			if got != test.want {
				t.Errorf("wrong result\na: %#v\nb: %#v\ngot:  %#v\nwant: %#v", test.a, test.b, got, test.want)
			}
		})
	}
}
