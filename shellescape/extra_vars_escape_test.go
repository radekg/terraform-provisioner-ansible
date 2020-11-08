package shellescape

import "testing"

func TestEscape(t *testing.T) {

	escape := NewSingleQuoteEscape("\\'")
	if escape.Safe() != "'\\''" {
		t.Fatal("Expected the sequence to be escaped")
	}

	escape = NewSingleQuoteEscape("'")
	if escape.Safe() != "'\\''" {
		t.Fatal("Expected the sequence to be escaped")
	}

	escape = NewSingleQuoteEscape("'\\")
	if escape.Safe() != "'\\''\\" {
		t.Fatal("Expected the sequence to be escaped")
	}

	escape = NewSingleQuoteEscape("'\\'")
	if escape.Safe() != "'\\''" {
		t.Fatal("Expected the sequence to be escaped")
	}

	escape = NewSingleQuoteEscape("'\\''")
	if escape.Safe() != "'\\''" {
		t.Fatal("Expected the sequence to be escaped")
	}

	escape = NewSingleQuoteEscape("''")
	if escape.Safe() != "'\\'''\\''" {
		t.Fatal("Expected the sequence to be escaped")
	}

	escape = NewSingleQuoteEscape("'\\n'")
	if escape.Safe() != "'\\''\\n'\\''" {
		t.Fatal("Expected the sequence to be escaped")
	}

	escape = NewSingleQuoteEscape(`{"field": "'", "field2": "'\'", "field3": "\''", "field4": "'\''"}`)
	if escape.Safe() != `{"field": "'\''", "field2": "'\''", "field3": "'\'''\''", "field4": "'\''"}` {
		t.Fatal("Expected the sequence to be escaped")
	}

}
