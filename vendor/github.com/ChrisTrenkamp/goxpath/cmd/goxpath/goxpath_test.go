package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"os"
	"strings"
	"testing"
)

func setup(in string, args ...string) (*bytes.Buffer, *bytes.Buffer) {
	retCode = 0
	os.Args = append([]string{"test"}, args...)
	flag.CommandLine = flag.NewFlagSet("test", flag.ExitOnError)
	out := &bytes.Buffer{}
	err := &bytes.Buffer{}
	stdout = out
	stderr = err
	stdin = strings.NewReader(in)
	exec()
	return out, err
}

func TestStdinVal(t *testing.T) {
	out, _ := setup(xml.Header+"<root><tag>test</tag></root>", "-v", "/root/tag")
	if out.String() != "test\n" {
		t.Error("Expecting 'test' for the result.  Recieved: ", out.String())
	}
	if retCode != 0 {
		t.Error("Incorrect return value")
	}
}

func TestStdinNonVal(t *testing.T) {
	out, _ := setup(xml.Header+"<root><tag>test</tag></root>", "/root/tag")
	if out.String() != "<tag>test</tag>\n" {
		t.Error("Expecting '<tag>test</tag>' for the result.  Recieved: ", out.String())
	}
	if retCode != 0 {
		t.Error("Incorrect return value")
	}
}

func TestFile(t *testing.T) {
	out, _ := setup("", "-ns", "foo=http://foo.bar", "/foo:test/foo:path", "test/1.xml")
	if out.String() != `<path xmlns="http://foo.bar">path</path>`+"\n" {
		t.Error(`Expecting '<path xmlns="http://foo.bar">path</path>' for the result.  Recieved: `, out.String())
	}
	if retCode != 0 {
		t.Error("Incorrect return value")
	}
}

func TestDir(t *testing.T) {
	out, _ := setup("", "-r", "/foo", "test/subdir")
	val := strings.Replace(out.String(), "test\\subdir\\", "test/subdir/", -1)
	if val != `test/subdir/2.xml:<foo>bar</foo>`+"\n"+`test/subdir/3.xml:<foo>bar2</foo>`+"\n" {
		t.Error(`Incorrect result.  Recieved: `, val)
	}
	if retCode != 0 {
		t.Error("Incorrect return value")
	}
}

func TestDirNonRec(t *testing.T) {
	_, err := setup("", "/foo", "test/subdir")
	val := strings.Replace(err.String(), "test\\subdir\\", "test/subdir/", -1)
	if val != `test/subdir: Is a directory`+"\n" {
		t.Error(`Incorrect result.  Recieved: `, val)
	}
	if retCode != 1 {
		t.Error("Incorrect return value")
	}
}

func TestNoXPath(t *testing.T) {
	_, err := setup("")
	if err.String() != "Specify an XPath expression with one or more files, or pipe the XML from stdin.  Run 'goxpath --help' for more information.\n" {
		t.Error("No XPath error")
	}
	if retCode != 1 {
		t.Error("Incorrect return value")
	}
}

func TestInvalidXPathExpr(t *testing.T) {
	_, err := setup("", "/foo()", "test/1.xml")
	if err.String() != "Invalid node-type foo\n" {
		t.Error("Invalid XPath error")
	}
	if retCode != 1 {
		t.Error("Incorrect return value")
	}
}

func TestInvalidFilePath(t *testing.T) {
	_, err := setup("", "/foo", "foo.xml")
	if err.String() != "Could not open file: foo.xml\n" {
		t.Error("Invalid error")
	}
	if retCode != 1 {
		t.Error("Incorrect return value")
	}
}

func TestXPathExecErr(t *testing.T) {
	_, err := setup("", "foobar()", "test/1.xml")
	if err.String() != "test/1.xml: Unknown function: foobar\n" {
		t.Error("Invalid error", err.String())
	}
	if retCode != 1 {
		t.Error("Incorrect return value")
	}
}

func TestXPathExecErrStdin(t *testing.T) {
	_, err := setup(xml.Header+"<root><tag>test</tag></root>", "foobar()")
	if err.String() != "Unknown function: foobar\n" {
		t.Error("Invalid error", err.String())
	}
	if retCode != 1 {
		t.Error("Incorrect return value")
	}
}

func TestInvalidXML(t *testing.T) {
	_, err := setup("<root>", "/root")
	if err.String() != "XML syntax error on line 1: unexpected EOF\n" {
		t.Error("Invalid error", err.String())
	}
	if retCode != 1 {
		t.Error("Incorrect return value")
	}
}

func TestVarRef(t *testing.T) {
	out, _ := setup(xml.Header+"<root><tag>test</tag></root>", "-var=foo=test", "/root/tag = $foo")
	if out.String() != "true\n" {
		t.Error("Expecting 'true' for the result.  Recieved: ", out.String())
	}
	if retCode != 0 {
		t.Error("Incorrect return value")
	}
}

func TestInvalidNSMap(t *testing.T) {
	_, err := setup(xml.Header+"<root/>", "-ns=foo=http://foo=bar", "/root")
	if err.String() != "Invalid namespace mapping: foo=http://foo=bar\n" {
		t.Error("Invalid error", err.String())
	}
	if retCode != 1 {
		t.Error("Incorrect return value")
	}
}

func TestInvalidVarMap(t *testing.T) {
	_, err := setup(xml.Header+"<root/>", "-var=test=blag=foo", "/root")
	if err.String() != "Invalid variable mapping: test=blag=foo\n" {
		t.Error("Invalid error", err.String())
	}
	if retCode != 1 {
		t.Error("Incorrect return value")
	}
}
