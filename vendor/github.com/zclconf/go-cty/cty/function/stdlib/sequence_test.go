package stdlib

import (
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestConcat(t *testing.T) {
	tests := []struct {
		Input []cty.Value
		Want  cty.Value
	}{
		{
			[]cty.Value{
				cty.ListValEmpty(cty.Number),
			},
			cty.ListValEmpty(cty.Number),
		},
		{
			[]cty.Value{
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
					cty.NumberIntVal(2),
					cty.NumberIntVal(3),
				}),
			},
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
			}),
		},
		{
			[]cty.Value{
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
				}),
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(2),
					cty.NumberIntVal(3),
				}),
			},
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
			}),
		},
		{
			[]cty.Value{
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("foo"),
				}),
				cty.ListVal([]cty.Value{
					cty.True,
				}),
			},
			cty.ListVal([]cty.Value{
				cty.StringVal("1"),
				cty.StringVal("foo"),
				cty.StringVal("true"),
			}),
		},
		{
			[]cty.Value{
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
				}),
				cty.ListVal([]cty.Value{
					cty.StringVal("foo"),
					cty.StringVal("bar"),
				}),
			},
			cty.ListVal([]cty.Value{
				cty.StringVal("1"),
				cty.StringVal("foo"),
				cty.StringVal("bar"),
			}),
		},
		{
			[]cty.Value{
				cty.EmptyTupleVal,
			},
			cty.EmptyTupleVal,
		},
		{
			[]cty.Value{
				cty.TupleVal([]cty.Value{
					cty.NumberIntVal(1),
					cty.True,
					cty.NumberIntVal(3),
				}),
			},
			cty.TupleVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.True,
				cty.NumberIntVal(3),
			}),
		},
		{
			[]cty.Value{
				cty.TupleVal([]cty.Value{
					cty.NumberIntVal(1),
				}),
				cty.TupleVal([]cty.Value{
					cty.True,
					cty.NumberIntVal(3),
				}),
			},
			cty.TupleVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.True,
				cty.NumberIntVal(3),
			}),
		},
		{
			[]cty.Value{
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
				}),
				cty.TupleVal([]cty.Value{
					cty.True,
					cty.NumberIntVal(3),
				}),
			},
			cty.TupleVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.True,
				cty.NumberIntVal(3),
			}),
		},
		{
			[]cty.Value{
				cty.TupleVal([]cty.Value{
					cty.NumberIntVal(1),
					cty.True,
				}),
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(3),
				}),
			},
			cty.TupleVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.True,
				cty.NumberIntVal(3),
			}),
		},
		{
			// Two lists with unconvertable element types become a tuple.
			[]cty.Value{
				cty.ListVal([]cty.Value{
					cty.NumberIntVal(1),
				}),
				cty.ListVal([]cty.Value{
					cty.ListValEmpty(cty.Bool),
				}),
			},
			cty.TupleVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.ListValEmpty(cty.Bool),
			}),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Concat(%#v...)", test.Input), func(t *testing.T) {
			got, err := Concat(test.Input...)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
			}
		})
	}
}

func TestRange(t *testing.T) {
	tests := []struct {
		Args []cty.Value
		Want cty.Value
	}{
		// One argument
		{
			[]cty.Value{
				cty.NumberIntVal(5),
			},
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(0),
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
				cty.NumberIntVal(4),
			}),
		},
		{
			[]cty.Value{
				cty.NumberIntVal(-5),
			},
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(0),
				cty.NumberIntVal(-1),
				cty.NumberIntVal(-2),
				cty.NumberIntVal(-3),
				cty.NumberIntVal(-4),
			}),
		},
		{
			[]cty.Value{
				cty.NumberIntVal(1),
			},
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(0),
			}),
		},
		{
			[]cty.Value{
				cty.NumberIntVal(0),
			},
			cty.ListValEmpty(cty.Number),
		},
		{
			[]cty.Value{
				cty.MustParseNumberVal("5.5"),
			},
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(0),
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
				cty.NumberIntVal(4),
				cty.NumberIntVal(5), // because 5 < 5.5
			}),
		},

		// Two arguments
		{
			[]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(5),
			},
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
				cty.NumberIntVal(4),
			}),
		},
		{
			[]cty.Value{
				cty.NumberIntVal(5),
				cty.NumberIntVal(1),
			},
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(5),
				cty.NumberIntVal(4),
				cty.NumberIntVal(3),
				cty.NumberIntVal(2),
			}),
		},
		{
			[]cty.Value{
				cty.NumberFloatVal(1.5),
				cty.NumberIntVal(5),
			},
			cty.ListVal([]cty.Value{
				cty.NumberFloatVal(1.5),
				cty.NumberFloatVal(2.5),
				cty.NumberFloatVal(3.5),
				cty.NumberFloatVal(4.5),
			}),
		},
		{
			[]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
			},
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(1),
			}),
		},
		{
			[]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(1),
			},
			cty.ListValEmpty(cty.Number),
		},

		// Three arguments
		{
			[]cty.Value{
				cty.NumberIntVal(0),
				cty.NumberIntVal(5),
				cty.NumberIntVal(2),
			},
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(0),
				cty.NumberIntVal(2),
				cty.NumberIntVal(4),
			}),
		},
		{
			[]cty.Value{
				cty.NumberIntVal(0),
				cty.NumberIntVal(5),
				cty.NumberIntVal(1),
			},
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(0),
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
				cty.NumberIntVal(4),
			}),
		},
		{
			[]cty.Value{
				cty.NumberIntVal(0),
				cty.NumberIntVal(1),
				cty.NumberIntVal(1),
			},
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(0),
			}),
		},
		{
			[]cty.Value{
				cty.NumberIntVal(0),
				cty.NumberIntVal(0),
				cty.NumberIntVal(1),
			},
			cty.ListValEmpty(cty.Number),
		},
		{
			[]cty.Value{
				cty.NumberIntVal(5),
				cty.NumberIntVal(0),
				cty.NumberIntVal(-1),
			},
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(5),
				cty.NumberIntVal(4),
				cty.NumberIntVal(3),
				cty.NumberIntVal(2),
				cty.NumberIntVal(1),
			}),
		},
		{
			[]cty.Value{
				cty.NumberIntVal(0),
				cty.NumberIntVal(5),
				cty.NumberFloatVal(0.5),
			},
			cty.ListVal([]cty.Value{
				cty.NumberIntVal(0),
				cty.NumberFloatVal(0.5),
				cty.NumberIntVal(1),
				cty.NumberFloatVal(1.5),
				cty.NumberIntVal(2),
				cty.NumberFloatVal(2.5),
				cty.NumberIntVal(3),
				cty.NumberFloatVal(3.5),
				cty.NumberIntVal(4),
				cty.NumberFloatVal(4.5),
			}),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("Range(%#v)", test.Args), func(t *testing.T) {
			got, err := Range(test.Args...)

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !got.RawEquals(test.Want) {
				t.Errorf(
					"wrong result\nargs: %#v\ngot:  %#v\nwant: %#v",
					test.Args, got, test.Want,
				)
			}
		})
	}
}
