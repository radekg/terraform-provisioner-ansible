package xmlstruct

import (
	"encoding/xml"
	"testing"

	"github.com/ChrisTrenkamp/goxpath"
)

func TestSingleFields(t *testing.T) {
	str := struct {
		XMLName xml.Name `xml:"struct"`
		Elem    string   `xml:"elem"`
		Attr    string   `xml:"attr,attr"`
		Attr2   string   `xml:",attr"`
		Comm    string   `xml:",comment"`
		CD      string   `xml:",chardata"`
		Test    interface{}
	}{
		Elem:  "foo",
		Attr:  "bar",
		Attr2: "baz",
		Comm:  "steak",
		CD:    "eggs",
		Test: struct {
			Elem2 string `xml:"elem2"`
			Attr3 string `xml:",attr"`
		}{
			Elem2: "elem2",
			Attr3: "attr3",
		},
	}

	x := MustParseStruct(&str)
	x1, err := xml.Marshal(str)
	str1 := string(x1)
	if err != nil {
		t.Error(err)
	}
	str2, err := goxpath.MarshalStr(x)
	if err != nil {
		t.Error(err)
	}
	if str1 != str2 {
		t.Error("Strings not equal")
		t.Error(str1)
		t.Error(str2)
	}
}
