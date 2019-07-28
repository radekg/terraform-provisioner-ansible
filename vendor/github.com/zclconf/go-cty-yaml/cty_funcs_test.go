package yaml

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestYAMLDecodeFunc(t *testing.T) {
	// FIXME: This is not a very extensive test.
	got, err := YAMLDecodeFunc.Call([]cty.Value{
		cty.StringVal("true"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if want := cty.True; !want.RawEquals(got) {
		t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestYAMLEncodeFunc(t *testing.T) {
	// FIXME: This is not a very extensive test.
	got, err := YAMLEncodeFunc.Call([]cty.Value{
		cty.StringVal("true"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if want := cty.StringVal("\"true\"\n"); !want.RawEquals(got) {
		t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", got, want)
	}
}
