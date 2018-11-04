package winrm

import (
	"bytes"
	"crypto/tls"
	"net/http"
	"strings"

	"github.com/masterzen/winrm/soap"

	. "gopkg.in/check.v1"
	"net"
	"time"
)

type Requester struct {
	http      func(*Client, *soap.SoapMessage) (string, error)
	transport http.RoundTripper
	dial func(network, addr string) (net.Conn, error)
}

func (r Requester) Post(client *Client, request *soap.SoapMessage) (string, error) {
	return r.http(client, request)
}

func (r Requester) Transport(endpoint *Endpoint) error {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: endpoint.Insecure,
		},
		ResponseHeaderTimeout: endpoint.Timeout,
		Dial: r.dial,
	}

	if endpoint.CACert != nil && len(endpoint.CACert) > 0 {
		certPool, err := readCACerts(endpoint.CACert)
		if err != nil {
			return err
		}

		transport.TLSClientConfig.RootCAs = certPool
	}

	r.transport = transport

	return nil

}

func (s *WinRMSuite) TestNewClient(c *C) {
	endpoint := NewEndpoint("localhost", 5985, false, false, nil, nil, nil, 0)
	client, err := NewClient(endpoint, "Administrator", "v3r1S3cre7")

	c.Assert(err, IsNil)
	c.Assert(client.url, Equals, "http://localhost:5985/wsman")
	c.Assert(client.username, Equals, "Administrator")
	c.Assert(client.password, Equals, "v3r1S3cre7")
}

func (s *WinRMSuite) TestClientCreateShell(c *C) {

	endpoint := NewEndpoint("localhost", 5985, false, false, nil, nil, nil, 0)
	client, err := NewClient(endpoint, "Administrator", "v3r1S3cre7")
	c.Assert(err, IsNil)
	r := Requester{}
	r.http = func(client *Client, message *soap.SoapMessage) (string, error) {
		c.Assert(message.String(), Contains, "http://schemas.xmlsoap.org/ws/2004/09/transfer/Create")
		return createShellResponse, nil
	}
	client.http = r

	shell, _ := client.CreateShell()
	c.Assert(shell.id, Equals, "67A74734-DD32-4F10-89DE-49A060483810")
}

func (s *WinRMSuite) TestRun(c *C) {
	ts, host, port, err := runWinRMFakeServer(c, "no input")
	c.Assert(err, IsNil)
	defer ts.Close()

	endpoint := NewEndpoint(host, port, false, false, nil, nil, nil, 0)
	client, err := NewClient(endpoint, "Administrator", "v3r1S3cre7")
	c.Assert(err, IsNil)

	var stdout, stderr bytes.Buffer
	code, err := client.Run("ipconfig /all", &stdout, &stderr)
	c.Assert(err, IsNil)
	c.Assert(code, Equals, 123)
	c.Assert(stdout.String(), Equals, "That's all folks!!!")
	c.Assert(stderr.String(), Equals, "This is stderr, I'm pretty sure!")
}

func (s *WinRMSuite) TestRunWithString(c *C) {
	ts, host, port, err := runWinRMFakeServer(c, "this is the input")
	c.Assert(err, IsNil)
	defer ts.Close()
	endpoint := NewEndpoint(host, port, false, false, nil, nil, nil, 0)
	client, err := NewClient(endpoint, "Administrator", "v3r1S3cre7")
	c.Assert(err, IsNil)

	stdout, stderr, code, err := client.RunWithString("ipconfig /all", "this is the input")
	c.Assert(err, IsNil)
	c.Assert(code, Equals, 123)
	c.Assert(stdout, Equals, "That's all folks!!!")
	c.Assert(stderr, Equals, "This is stderr, I'm pretty sure!")
}

func (s *WinRMSuite) TestRunWithInput(c *C) {
	ts, host, port, err := runWinRMFakeServer(c, "this is the input")
	c.Assert(err, IsNil)
	defer ts.Close()

	endpoint := NewEndpoint(host, port, false, false, nil, nil, nil, 0)
	client, err := NewClient(endpoint, "Administrator", "v3r1S3cre7")
	c.Assert(err, IsNil)

	var stdout, stderr bytes.Buffer
	code, err := client.RunWithInput("ipconfig /all", &stdout, &stderr, strings.NewReader("this is the input"))
	c.Assert(err, IsNil)
	c.Assert(code, Equals, 123)
	c.Assert(stdout.String(), Equals, "That's all folks!!!")
	c.Assert(stderr.String(), Equals, "This is stderr, I'm pretty sure!")
}

var cert = `
-----BEGIN CERTIFICATE-----
MIIFCzCCAvOgAwIBAgIUYsw8cUIEaFPeEqFg6h+T2xGX3JUwDQYJKoZIhvcNAQEL
BQAwHDEaMBgGA1UEAxMRd2lucm0gY2xpZW50IGNlcnQwHhcNMTYwNjA4MDkyODUy
WhcNMTcwNjA4MDkyODUyWjAcMRowGAYDVQQDExF3aW5ybSBjbGllbnQgY2VydDCC
AiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAJh5zw1Hlm0LRyftzEj9usJp
WlRQvL6U89IwfjPM+ip3J28f47+e7BjjXj2UQQJNKP1GrkzoJGgPVD2nV3AwfkZa
TYUD+ciSpwaAdcVFGWbKofL2YbPbNIyz6Yp3+pu2vxNrA1Q5e+shmn3CSDjYNljD
JpKdo6L8dd8KZqYxyHxvI4GoxGsL6Ij5dgha5ua1hENSXQEEUKMBLDPHTHy+EQCB
A/DvuW7SSaxjfJ2kPy2uW2wePkZub6rt2TRBPjYjv5pPQ97jFYnj4nAMzjv1O/h7
moffc23/gzYQ0w0S0O3ZD7VNCejJpD0O2obto2cAWfOzp7F5B3zH9jFJCfXfVOKN
gWYDr+PsYTg9XN/yd78D7A14z4sN+aANT+1fd+uFdHHgRmF8sAEYLKTAqEvvKbsg
o/eZoM8RkMqDZnl1eDK3r0AGPqLieRUchSYk+wjbNyfLReXmxRqbmFEfgF2izX3/
FscGvFCPLhCE3L4vf6rgDRWUkC8oqdiGBUQrGtHYjlbwyDi9ntzlpK8GHx51PKQy
+3g1We1sTNzuTRy8xqKJPlDhnDjjyqEASXHyjycGxJmTojGpvc9R7yEGFx+7VLDy
9IoDTE0/s4dZlK/FcV9piz7Oy5/NpJ2IkUzLtDioFUasPky5657BUjbnK0pXAMIr
sMK5/c2SoAKN0jt5tCm5AgMBAAGjRTBDMBMGA1UdJQQMMAoGCCsGAQUFBwMCMCwG
A1UdEQQlMCOgIQYKKwYBBAGCNxQCA6ATDBF3aW5ybSBjbGllbnQgY2VydDANBgkq
hkiG9w0BAQsFAAOCAgEASBQYRW0jb2r+TdRd54Zx5O2MiMsejM4/ae7qZELnmwrv
PD+AlvWdl160TBhQOOUjzmcagcThx7FvENoKe0VPg5XXHRysm3I8VozbUBuuIFWz
Ayv8H5zGopS/OzR8nUucet8LBnDxRGxOqN44P2iTrRwXuSnfh+hzm8LRo6qr5DGQ
JvjNvUdGMy4HN+z3Gy4kXUJX3kkQIlUyUglCmyuKdNb4DpOPPJPcL2mZGyjNyKsF
zdVM+r8sO92+fcAn6lYK7FF9WIMyRdnjWUas1qOJpxhN5rW6YjpE6koOiDJfC8M0
gENPCy0x+A6zNI33Gvpk1S0Fr5RIOVB2SQhkBzkESli61ZRpCp8ndNN3T/WSd8oJ
IjEd0q5G3DdtZtEq2kE06gtZmyRpwrVbcrAJoTnlD71DdMlo3e8jjeFOqn2UpvS8
AxZ8nRgA/N+DBD3eXo9aH70Bw3mZaF2Uou48gFqR08aQ+lnYv9J/KQfdsrT1vMpw
zceXla1lJtQpZn/82xxL0ycTwafUf5ClX9F7aBUbSikKBhKYsECKE+GMkhXtKh2D
rnFNh5JNpki1r92WuiOa2PNSprbDZbfACnNm74SXcbJ8TqggxMt1svDO/rYv979K
7jp2G7gwj6fpHYBgR+ZY1kcScIHfyNUhfn1YspjOPRdl14g+tRx6QIeXF2zaq7Q=
-----END CERTIFICATE-----
`[1:]

var key = `
-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
`[1:]

func (s *WinRMSuite) TestReplaceTransportWithDecorator(c *C) {
	params := NewParameters("PT60S", "en-US", 153600)
	params.TransportDecorator = func() Transporter {
		return &ClientAuthRequest{}
	}

	endpoint := NewEndpoint("localhost", 5986, false, false, nil, []byte(cert), []byte(key), 0)
	client, err := NewClientWithParameters(endpoint, "Administrator", "password", params)
	c.Assert(err, IsNil)
	_, ok := client.http.(*ClientAuthRequest)
	c.Assert(ok, Equals, true)
}


func (s *WinRMSuite) TestReplaceDial(c *C) {
	ts, host, port, err := runWinRMFakeServer(c, "this is the input")
	c.Assert(err, IsNil)
	defer ts.Close()

	normalDialer := (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).Dial

	params := NewParameters("PT60S", "en-US", 153600)
	usedCustomDial := false
	params.Dial = func(network, addr string) (net.Conn, error) {
		usedCustomDial = true
		return normalDialer(network, addr)
	}

	endpoint := NewEndpoint(host, port, false, false, nil, nil, nil, 0)
	client, err := NewClientWithParameters(endpoint, "Administrator", "v3r1S3cre7", params)
	var stdout, stderr bytes.Buffer
	_, err = client.Run("ipconfig /all", &stdout, &stderr)
	c.Assert(err, IsNil)
	c.Assert(usedCustomDial, Equals, true)
}
