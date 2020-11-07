package types

import "testing"

func TestEscapeExtraVars(t *testing.T) {
	escaped := escapeExtraVars("this is ' a string '\\'' to ' escape")
	if escaped != "this is '\\'' a string '\\'' to '\\'' escape" {
		t.Fatal("string not escaped properly")
	}
}
