package xmlstruct

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"strings"

	"github.com/ChrisTrenkamp/goxpath/tree"
)

type XMLEle struct {
	Val     reflect.Value
	pos     int
	prnt    tree.Elem
	prntTag string
}

func (x *XMLEle) ResValue() string {
	if x.Val.Kind() != reflect.Struct {
		return fmt.Sprintf("%v", x.Val.Interface())
	}

	ret := ""
	for _, i := range x.GetChildren() {
		switch i.GetNodeType() {
		case tree.NtChd, tree.NtElem, tree.NtRoot:
			ret += i.ResValue()
		}
	}
	return ret
}

func (x *XMLEle) Pos() int {
	return x.pos
}

func (x *XMLEle) GetToken() xml.Token {
	ret := xml.StartElement{}

	if x.prntTag != "" {
		ret.Name = getTagInfo(x.prntTag).name
	} else {
		ret.Name = getXMLName(x.Val)
	}

	return ret
}

func (x *XMLEle) GetParent() tree.Elem {
	return x.prnt
}

func (x *XMLEle) GetNodeType() tree.NodeType {
	return tree.NtElem
}

func (x *XMLEle) GetChildren() []tree.Node {
	n, _ := getChildren(x, x.Val, x.pos, false)
	return n
}

func (x *XMLEle) GetAttrs() []tree.Node {
	n, _ := getChildren(x, x.Val, x.pos, true)
	return n
}

type tagInfo struct {
	name    xml.Name
	attr    bool
	cdata   bool
	comment bool
}

func getTagInfo(tag string) (ret tagInfo) {
	spl := strings.Split(tag, ",")
	name := strings.Split(spl[0], " ")
	if len(spl) >= 2 {
		for i := 1; i < len(spl); i++ {
			if spl[i] == "chardata" || spl[i] == "cdata" {
				ret.cdata = true
			} else if spl[i] == "attr" {
				ret.attr = true
			} else if spl[i] == "comment" {
				ret.comment = true
			}
		}
	}

	if len(name) == 2 {
		ret.name.Space = name[1]
	}

	ret.name.Local = name[0]
	return
}

func getXMLName(val reflect.Value) xml.Name {
	n := val.FieldByName("XMLName")
	zero := reflect.Value{}

	if zero != n {
		field, _ := val.Type().FieldByName("XMLName")
		tagInfo := getTagInfo(field.Tag.Get("xml"))
		if tagInfo.name.Local != "" {
			return tagInfo.name
		}

		if name, ok := n.Interface().(xml.Name); ok {
			return name
		}
	}

	return xml.Name{Local: val.Type().Name()}
}

func getChildren(x *XMLEle, val reflect.Value, pos int, getAttrs bool) ([]tree.Node, int) {
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() == reflect.Interface {
		val = reflect.ValueOf(val.Interface())
	}

	if val.Kind() != reflect.Struct {
		if getAttrs {
			return []tree.Node{}, x.pos + 1
		}

		return []tree.Node{&XMLNode{
			Val:      x.Val,
			pos:      x.pos + 1,
			prnt:     x,
			nodeType: tree.NtChd,
			prntTag:  "",
		}}, x.pos + 2
	}

	ret := make([]tree.Node, 0, val.NumField())

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		name := val.Type().Field(i).Name

		if val.Type().Field(i).Anonymous {
			nodes, newPos := getChildren(x, field, pos, getAttrs)
			ret = append(ret, nodes...)
			pos = newPos
			continue
		}

		tag := val.Type().Field(i).Tag.Get("xml")

		if tag == "-" || name == "XMLName" {
			continue
		}

		tagInfo := getTagInfo(tag)

		pos++

		if tagInfo.attr || tagInfo.cdata || tagInfo.comment {
			if !getAttrs && tagInfo.attr || getAttrs && !tagInfo.attr {
				continue
			}

			child := &XMLNode{
				Val:     field,
				pos:     pos,
				prnt:    x,
				prntTag: tag,
			}

			if tagInfo.attr {
				child.nodeType = tree.NtAttr
			} else if tagInfo.cdata {
				child.nodeType = tree.NtChd
			} else {
				child.nodeType = tree.NtComm
			}

			if tagInfo.name.Local == "" {
				child.prntTag = name
			}

			ret = append(ret, child)

			continue
		}

		if getAttrs {
			continue
		}

		child := &XMLEle{Val: field, pos: pos}

		if tag == "" {
			tag = name
		}

		child.prntTag = tag
		ret = append(ret, child)
	}

	return ret, pos + 1
}
