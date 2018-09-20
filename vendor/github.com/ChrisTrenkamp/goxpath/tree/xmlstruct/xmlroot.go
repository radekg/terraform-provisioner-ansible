package xmlstruct

import (
	"encoding/xml"

	"github.com/ChrisTrenkamp/goxpath/tree"
)

type XMLRoot struct {
	Ele *XMLEle
}

func (x *XMLRoot) ResValue() string {
	return x.Ele.ResValue()
}

func (x *XMLRoot) Pos() int {
	return 0
}

func (x *XMLRoot) GetToken() xml.Token {
	return xml.StartElement{}
}

func (x *XMLRoot) GetParent() tree.Elem {
	return x
}

func (x *XMLRoot) GetNodeType() tree.NodeType {
	return tree.NtRoot
}

func (x *XMLRoot) GetChildren() []tree.Node {
	return []tree.Node{x.Ele}
}

func (x *XMLRoot) GetAttrs() []tree.Node {
	return nil
}
