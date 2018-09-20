package winrmcp

import (
	"testing"
	"time"
)

func Test_parsing_an_addr_to_a_winrm_endpoint(t *testing.T) {
	timeout, _ := time.ParseDuration("1s")
	endpoint, err := parseEndpoint("1.2.3.4:1234", false, false, "foo", nil, timeout)

	if err != nil {
		t.Fatalf("Should not have been an error: %v", err)
	}
	if endpoint == nil {
		t.Error("Endpoint should not be nil")
	}
	if endpoint.Host != "1.2.3.4" {
		t.Error("Host should be 1.2.3.4")
	}
	if endpoint.Port != 1234 {
		t.Error("Port should be 1234")
	}
	if endpoint.Insecure {
		t.Error("Endpoint should be insecure")
	}
	if endpoint.HTTPS {
		t.Error("Endpoint should be HTTP not HTTPS")
	}
	if endpoint.Timeout != 1*time.Second {
		t.Error("Timeout should be 1s")
	}
}

func Test_parsing_an_addr_without_a_port_to_a_winrm_endpoint(t *testing.T) {
	certBytes := []byte{1, 2, 3, 4, 5, 6}
	endpoint, err := parseEndpoint("1.2.3.4", true, true, "foo", certBytes, 0)

	if err != nil {
		t.Fatalf("Should not have been an error: %v", err)
	}
	if endpoint == nil {
		t.Error("Endpoint should not be nil")
	}
	if endpoint.Host != "1.2.3.4" {
		t.Error("Host should be 1.2.3.4")
	}
	if endpoint.Port != 5985 {
		t.Error("Port should be 5985")
	}
	if endpoint.Insecure != true {
		t.Error("Endpoint should be insecure")
	}
	if endpoint.HTTPS != true {
		t.Error("Endpoint should be HTTPS")
	}

	if len(endpoint.CACert) != len(certBytes) {
		t.Error("Length of CACert is wrong")
	}
	for i := 0; i < len(certBytes); i++ {
		if (endpoint.CACert)[i] != certBytes[i] {
			t.Error("CACert is not set correctly")
		}
	}
}

func Test_parsing_a_hostname_to_a_winrm_endpoint(t *testing.T) {
	timeout, _ := time.ParseDuration("1s")
	endpoint, err := parseEndpoint("windows01:1234", false, false, "foo", nil, timeout)

	if err != nil {
		t.Fatalf("Should not have been an error: %v", err)
	}
	if endpoint == nil {
		t.Error("Endpoint should not be nil")
	}
	if endpoint.Host != "windows01" {
		t.Error("Host should be windows01")
	}
	if endpoint.Port != 1234 {
		t.Error("Port should be 1234")
	}
	if endpoint.Insecure {
		t.Error("Endpoint should be insecure")
	}
	if endpoint.HTTPS {
		t.Error("Endpoint should be HTTP not HTTPS")
	}
	if endpoint.Timeout != 1*time.Second {
		t.Error("Timeout should be 1s")
	}
}

func Test_parsing_a_hostname_without_a_port_to_a_winrm_endpoint(t *testing.T) {
	certBytes := []byte{1, 2, 3, 4, 5, 6}
	endpoint, err := parseEndpoint("windows01.microsoft.com", true, true, "foo", certBytes, 0)

	if err != nil {
		t.Fatalf("Should not have been an error: %v", err)
	}
	if endpoint == nil {
		t.Error("Endpoint should not be nil")
	}
	if endpoint.Host != "windows01.microsoft.com" {
		t.Error("Host should be windows01.microsoft.com")
	}
	if endpoint.Port != 5985 {
		t.Error("Port should be 5985")
	}
	if endpoint.Insecure != true {
		t.Error("Endpoint should be insecure")
	}
	if endpoint.HTTPS != true {
		t.Error("Endpoint should be HTTPS")
	}

	if len(endpoint.CACert) != len(certBytes) {
		t.Error("Length of CACert is wrong")
	}
	for i := 0; i < len(certBytes); i++ {
		if (endpoint.CACert)[i] != certBytes[i] {
			t.Error("CACert is not set correctly")
		}
	}
}

func Test_parsing_an_ipv6_addr_to_a_winrm_endpoint(t *testing.T) {
	timeout, _ := time.ParseDuration("1s")
	endpoint, err := parseEndpoint("[2402:9900:111:1373:ae5:1c:4cb8:dae0]:1234", false, false, "foo", nil, timeout)

	if err != nil {
		t.Fatalf("Should not have been an error: %v", err)
	}
	if endpoint == nil {
		t.Error("Endpoint should not be nil")
	}
	if endpoint.Host != "[2402:9900:111:1373:ae5:1c:4cb8:dae0]" {
		t.Error("Host should be [2402:9900:111:1373:ae5:1c:4cb8:dae0]")
	}
	if endpoint.Port != 1234 {
		t.Error("Port should be 1234")
	}
	if endpoint.Insecure {
		t.Error("Endpoint should be insecure")
	}
	if endpoint.HTTPS {
		t.Error("Endpoint should be HTTP not HTTPS")
	}
	if endpoint.Timeout != 1*time.Second {
		t.Error("Timeout should be 1s")
	}
}

func Test_parsing_an_ipv6_addr_without_a_port_to_a_winrm_endpoint(t *testing.T) {
	certBytes := []byte{1, 2, 3, 4, 5, 6}
	endpoint, err := parseEndpoint("[2402:9900:111:1373:192a:6b0e:7c46:2563]", true, true, "foo", certBytes, 0)

	if err != nil {
		t.Fatalf("Should not have been an error: %v", err)
	}
	if endpoint == nil {
		t.Error("Endpoint should not be nil")
	}
	if endpoint.Host != "[2402:9900:111:1373:192a:6b0e:7c46:2563]" {
		t.Error("Host should be [2402:9900:111:1373:192a:6b0e:7c46:2563]")
	}
	if endpoint.Port != 5985 {
		t.Error("Port should be 5985")
	}
	if endpoint.Insecure != true {
		t.Error("Endpoint should be insecure")
	}
	if endpoint.HTTPS != true {
		t.Error("Endpoint should be HTTPS")
	}

	if len(endpoint.CACert) != len(certBytes) {
		t.Error("Length of CACert is wrong")
	}
	for i := 0; i < len(certBytes); i++ {
		if (endpoint.CACert)[i] != certBytes[i] {
			t.Error("CACert is not set correctly")
		}
	}
}

func Test_parsing_an_empty_addr_to_a_winrm_endpoint(t *testing.T) {
	endpoint, err := parseEndpoint("", false, false, "foo", nil, 0)

	if endpoint != nil {
		t.Error("Endpoint should be nil")
	}
	if err == nil {
		t.Error("Expected an error")
	}
}

func Test_parsing_an_addr_with_a_bad_port(t *testing.T) {
	endpoint, err := parseEndpoint("1.2.3.4:ABCD", false, false, "foo", nil, 0)

	if endpoint != nil {
		t.Error("Endpoint should be nil")
	}
	if err == nil {
		t.Error("Expected an error")
	}
}
