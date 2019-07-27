package dom

import (
	"testing"

	. "github.com/google/go-cmp/cmp"
)

func TestEmptyDocument(t *testing.T) {
	doc := CreateDocument()
	doc.PrettyPrint = true
	if diff := Diff(doc.String(), "<?xml version=\"1.0\" encoding=\"utf-8\" ?>\n"); diff != "" {
		t.Fatalf("Unexpected output: %s", diff)
	}
}

func TestOneEmptyNode(t *testing.T) {
	doc := CreateDocument()
	doc.PrettyPrint = true
	root := CreateElement("root")
	doc.SetRoot(root)
	if diff := Diff(doc.String(), "<?xml version=\"1.0\" encoding=\"utf-8\" ?>\n<root/>\n"); diff != "" {
		t.Fatalf("Unexpected output: %s", diff)
	}
}

func TestMoreNodes(t *testing.T) {
	doc := CreateDocument()
	doc.PrettyPrint = true
	root := CreateElement("root")
	node1 := CreateElement("node1")
	root.AddChild(node1)
	subnode := CreateElement("sub")
	node1.AddChild(subnode)
	node2 := CreateElement("node2")
	root.AddChild(node2)
	doc.SetRoot(root)

	expected := `<?xml version="1.0" encoding="utf-8" ?>
<root>
  <node1>
    <sub/>
  </node1>
  <node2/>
</root>
`

	if diff := Diff(doc.String(), expected); diff != "" {
		t.Fatalf("Unexpected output: %s", diff)
	}
}

func TestAttributes(t *testing.T) {
	doc := CreateDocument()
	doc.PrettyPrint = true
	root := CreateElement("root")
	node1 := CreateElement("node1")
	node1.SetAttr("attr1", "pouet")
	root.AddChild(node1)
	doc.SetRoot(root)

	expected := `<?xml version="1.0" encoding="utf-8" ?>
<root>
  <node1 attr1="pouet"/>
</root>
`
	if diff := Diff(doc.String(), expected); diff != "" {
		t.Fatalf("Unexpected output: %s", diff)
	}
}

func TestContent(t *testing.T) {
	doc := CreateDocument()
	doc.PrettyPrint = true
	root := CreateElement("root")
	node1 := CreateElement("node1")
	node1.SetContent("this is a text content")
	root.AddChild(node1)
	doc.SetRoot(root)

	expected := `<?xml version="1.0" encoding="utf-8" ?>
<root>
  <node1>this is a text content</node1>
</root>
`
	if diff := Diff(doc.String(), expected); diff != "" {
		t.Fatalf("Unexpected output: %s", diff)
	}
}

func TestNamespace(t *testing.T) {
	doc := CreateDocument()
	doc.PrettyPrint = true
	root := CreateElement("root")
	root.DeclareNamespace(Namespace{Prefix: "a", Uri: "http://schemas.xmlsoap.org/ws/2004/08/addressing"})
	node1 := CreateElement("node1")
	root.AddChild(node1)
	node1.SetNamespace("a", "http://schemas.xmlsoap.org/ws/2004/08/addressing")
	node1.SetContent("this is a text content")
	doc.SetRoot(root)

	expected := `<?xml version="1.0" encoding="utf-8" ?>
<root xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing">
  <a:node1>this is a text content</a:node1>
</root>
`
	if diff := Diff(doc.String(), expected); diff != "" {
		t.Fatalf("Unexpected output: %s", diff)
	}
}
