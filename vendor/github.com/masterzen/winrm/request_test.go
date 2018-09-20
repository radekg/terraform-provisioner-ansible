package winrm

import (
	"strings"
	"testing"

	"github.com/ChrisTrenkamp/goxpath"
	"github.com/ChrisTrenkamp/goxpath/tree"
	"github.com/ChrisTrenkamp/goxpath/tree/xmltree"
	"github.com/masterzen/winrm/soap"
	"github.com/masterzen/simplexml/dom"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type WinRMSuite struct{}

var _ = Suite(&WinRMSuite{})

func (s *WinRMSuite) TestOpenShellRequest(c *C) {
	openShell := NewOpenShellRequest("http://localhost", nil)
	defer openShell.Free()

	assertXPath(c, openShell.Doc(), "//a:Action", "http://schemas.xmlsoap.org/ws/2004/09/transfer/Create")
	assertXPath(c, openShell.Doc(), "//a:To", "http://localhost")
	assertXPath(c, openShell.Doc(), "//env:Body/rsp:Shell/rsp:InputStreams", "stdin")
	assertXPath(c, openShell.Doc(), "//env:Body/rsp:Shell/rsp:OutputStreams", "stdout stderr")
}

func (s *WinRMSuite) TestDeleteShellRequest(c *C) {
	request := NewDeleteShellRequest("http://localhost", "SHELLID", nil)
	defer request.Free()

	assertXPath(c, request.Doc(), "//a:Action", "http://schemas.xmlsoap.org/ws/2004/09/transfer/Delete")
	assertXPath(c, request.Doc(), "//a:To", "http://localhost")
	assertXPath(c, request.Doc(), "//w:Selector[@Name=\"ShellId\"]", "SHELLID")
}

func (s *WinRMSuite) TestExecuteCommandRequest(c *C) {
	request := NewExecuteCommandRequest("http://localhost", "SHELLID", "ipconfig /all", []string{}, nil)
	defer request.Free()

	assertXPath(c, request.Doc(), "//a:Action", "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Command")
	assertXPath(c, request.Doc(), "//a:To", "http://localhost")
	assertXPath(c, request.Doc(), "//w:Selector[@Name=\"ShellId\"]", "SHELLID")
	assertXPath(c, request.Doc(), "//w:Option[@Name=\"WINRS_CONSOLEMODE_STDIN\"]", "TRUE")
	assertXPath(c, request.Doc(), "//rsp:CommandLine/rsp:Command", "ipconfig /all")
	assertXPathNil(c, request.Doc(), "//rsp:CommandLine/rsp:Arguments")
}

func (s *WinRMSuite) TestExecuteCommandWithArgumentsRequest(c *C) {
	args := []string{"/p", "C:\\test.txt"}
	request := NewExecuteCommandRequest("http://localhost", "SHELLID", "del", args, nil)
	defer request.Free()

	assertXPath(c, request.Doc(), "//a:Action", "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Command")
	assertXPath(c, request.Doc(), "//a:To", "http://localhost")
	assertXPath(c, request.Doc(), "//w:Selector[@Name=\"ShellId\"]", "SHELLID")
	assertXPath(c, request.Doc(), "//w:Option[@Name=\"WINRS_CONSOLEMODE_STDIN\"]", "TRUE")
	assertXPath(c, request.Doc(), "//rsp:CommandLine/rsp:Command", "del")
	assertXPath(c, request.Doc(), "//rsp:CommandLine/rsp:Arguments", "/p")
	assertXPath(c, request.Doc(), "//rsp:CommandLine/rsp:Arguments", "C:\\test.txt")
}

func (s *WinRMSuite) TestGetOutputRequest(c *C) {
	request := NewGetOutputRequest("http://localhost", "SHELLID", "COMMANDID", "stdout stderr", nil)
	defer request.Free()

	assertXPath(c, request.Doc(), "//a:Action", "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Receive")
	assertXPath(c, request.Doc(), "//a:To", "http://localhost")
	assertXPath(c, request.Doc(), "//w:Selector[@Name=\"ShellId\"]", "SHELLID")
	assertXPath(c, request.Doc(), "//rsp:Receive/rsp:DesiredStream[@CommandId=\"COMMANDID\"]", "stdout stderr")
}

func (s *WinRMSuite) TestSendInputRequest(c *C) {
	request := NewSendInputRequest("http://localhost", "SHELLID", "COMMANDID", []byte{31, 32}, nil)
	defer request.Free()

	assertXPath(c, request.Doc(), "//a:Action", "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Send")
	assertXPath(c, request.Doc(), "//a:To", "http://localhost")
	assertXPath(c, request.Doc(), "//w:Selector[@Name=\"ShellId\"]", "SHELLID")
	assertXPath(c, request.Doc(), "//rsp:Send/rsp:Stream[@CommandId=\"COMMANDID\"]", "HyA=")
}

func (s *WinRMSuite) TestSignalRequest(c *C) {
	request := NewSignalRequest("http://localhost", "SHELLID", "COMMANDID", nil)
	defer request.Free()

	assertXPath(c, request.Doc(), "//a:Action", "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Signal")
	assertXPath(c, request.Doc(), "//a:To", "http://localhost")
	assertXPath(c, request.Doc(), "//w:Selector[@Name=\"ShellId\"]", "SHELLID")
	assertXPath(c, request.Doc(), "//rsp:Signal[@CommandId=\"COMMANDID\"]/rsp:Code", "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/signal/terminate")
}

func assertXPath(c *C, doc *dom.Document, request string, expected string) {
	nodes, err := parseXPath(doc, request)

	if err != nil {
		c.Fatalf("Xpath %s gives error %s", request, err)
	}

	c.Assert(len(nodes), Not(Equals), 0)

	var foundValue string
	for _, i := range nodes {
		foundValue = i.ResValue()
		if foundValue == expected {
			break
		}
	}

	if foundValue != expected {
		c.Errorf("Should have found '%s', but found '%s' instead", expected, foundValue)
	}
}

func assertXPathNil(c *C, doc *dom.Document, request string) {
	nodes, err := parseXPath(doc, request)

	if err != nil {
		c.Fatalf("Xpath %s gives error %s", request, err)
	}

	c.Assert(len(nodes), Equals, 0)
}

func parseXPath(doc *dom.Document, request string) (tree.NodeSet, error) {
	content := strings.NewReader(doc.String())
	body, err := xmltree.ParseXML(content)
	if err != nil {
		return nil, err
	}

	xpExec := goxpath.MustParse(request)
	nodes, err := xpExec.ExecNode(body, soap.GetAllXPathNamespaces())
	if err != nil {
		return nil, err
	}
	return nodes, nil
}
