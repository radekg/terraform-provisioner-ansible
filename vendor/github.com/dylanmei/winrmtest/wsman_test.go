package winrmtest

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/antchfx/xquery/xml"
	"github.com/satori/go.uuid"
)

func Test_creating_a_shell(t *testing.T) {
	w := &wsman{}

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "", strings.NewReader(`
		<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing">
			<env:Header>
				<a:Action mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/transfer/Create</a:Action>
			</env:Header>
			<env:Body>
				<rsp:Shell>
					<rsp:InputStream>stdin</rsp:InputStream>
					<rsp:OutputStreams>stdout stderr</rsp:OutputStreams>
				</rsp:Shell>
			</env:Body>
		</env:Envelope>`))

	w.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Errorf("Expected 200 OK but was %d.\n", res.Code)
	}

	if contentType := res.HeaderMap.Get("Content-Type"); contentType != "application/soap+xml" {
		t.Errorf("Expected ContentType application/soap+xml was %s.\n", contentType)
	}

	doc, err := xmlquery.Parse(res.Body)
	if err != nil {
		t.Errorf("Couldn't parse XML: %s", err)
	}
	result := xmlquery.FindOne(doc, "//rsp:ShellId").InnerText()
	if result == "" {
		t.Error("Expected a Shell identifier.")
	}
}

func Test_executing_a_command(t *testing.T) {
	w := &wsman{}
	id := w.HandleCommand(MatchText("echo tacos"), func(out, err io.Writer) int {
		return 0
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "", strings.NewReader(`
		<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell">
			<env:Header>
				<a:Action mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/shell/Command</a:Action>
			</env:Header>
			<env:Body>
				<rsp:CommandLine><rsp:Command>"echo tacos"</rsp:Command></rsp:CommandLine>
			</env:Body>
		</env:Envelope>`))

	w.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK but was %d.\n", res.Code)
	}

	doc, err := xmlquery.Parse(res.Body)
	if err != nil {
		t.Errorf("Couldn't compile the SOAP response: %s", err)
	}
	result := xmlquery.FindOne(doc, "//rsp:CommandId").InnerText()

	if result != id {
		t.Errorf("Expected CommandId=%s but was %q", id, result)
	}
}

func Test_executing_a_regex_command(t *testing.T) {
	w := &wsman{}
	id := w.HandleCommand(MatchPattern(`echo .* >> C:\file.cmd`), func(out, err io.Writer) int {
		return 0
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "", strings.NewReader(fmt.Sprintf(`
		<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell">
			<env:Header>
				<a:Action mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/shell/Command</a:Action>
			</env:Header>
			<env:Body>
				<rsp:CommandLine><rsp:Command>"echo %d >> C:\file.cmd"</rsp:Command></rsp:CommandLine>
			</env:Body>
		</env:Envelope>`, uuid.NewV4().String())))

	w.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK but was %d.\n", res.Code)
	}

	doc, err := xmlquery.Parse(res.Body)
	if err != nil {
		t.Errorf("Couldn't compile the SOAP response.")
	}
	result := xmlquery.FindOne(doc, "//rsp:CommandId").InnerText()

	if result != id {
		t.Errorf("Expected CommandId=%s, but was %q", id, result)
	}
}

func Test_receiving_command_results(t *testing.T) {
	w := &wsman{}
	id := w.HandleCommand(MatchText("echo tacos"), func(out, err io.Writer) int {
		out.Write([]byte("tacos"))
		return 0
	})

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "", strings.NewReader(fmt.Sprintf(`
		<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell">
			<env:Header>
				<a:Action mustUnderstand="true">http://schemas.microsoft.com/wbem/wsman/1/windows/shell/Receive</a:Action>
			</env:Header>
			<env:Body>
				<rsp:Receive><rsp:DesiredStream CommandId="%s">stdout stderr</rsp:DesiredStream></rsp:Receive>
			</env:Body>
		</env:Envelope>`, id)))

	w.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK but was %d.\n", res.Code)
	}

	doc, err := xmlquery.Parse(res.Body)
	if err != nil {
		t.Errorf("Couldn't compile the SOAP response: %s", err)
	}

	result := xmlquery.FindOne(doc, "//rsp:ReceiveResponse")
	iter := xmlquery.CreateXPathNavigator(result)
	if !iter.MoveToChild() {
		t.Error("Expected a ReceiveResponse element.")
	}

	xresp := iter.Current()
	result = xmlquery.FindOne(xresp, fmt.Sprintf("rsp:Stream[@CommandId='%s']", id))
	iter = xmlquery.CreateXPathNavigator(xresp)

	testText := "dGFjb3M="
	if !iter.MoveToNext() ||
		iter.Current().SelectAttr("Name") != "stdout" ||
		iter.Current().InnerText() != testText {
		t.Errorf("Expected an stdout Stream with the text %q", testText)
	}

	if !iter.MoveToNext() ||
		iter.Current().SelectAttr("Name") != "stdout" ||
		iter.Current().SelectAttr("End") != "true" {
		t.Errorf("Expected an stdout Stream with an %q attribute", "end")
	}

	if !iter.MoveToNext() ||
		iter.Current().SelectAttr("Name") != "stderr" ||
		iter.Current().SelectAttr("End") != "true" {
		t.Errorf("Expected an stderr Stream with an %q attribute", "end")
	}

	result = xmlquery.FindOne(doc, "//rsp:CommandState")
	if result == nil {
		t.Errorf("Expected CommandState=%q, got: nil", "Done")
	}

	if result.SelectAttr("State") != "http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandState/Done" {
		t.Errorf("Expected CommandState=%q", "Done")
	}

	result = xmlquery.FindOne(doc,
		"//rsp:CommandState/rsp:ExitCode")
	if result.InnerText() != "0" {
		t.Errorf("Expected ExitCode=0 but found %q", result.InnerText())
	}
}

func Test_deleting_a_shell(t *testing.T) {
	w := &wsman{}

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "", strings.NewReader(`
		<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing">
			<env:Header>
				<a:Action mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/transfer/Delete</a:Action>
			</env:Header>
		</env:Envelope>`))

	w.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK but was %d.\n", res.Code)
	}

	if res.Body.Len() != 0 {
		t.Errorf("Expected body to be empty but was \"%v\".", res.Body)
	}
}
