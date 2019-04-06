package mode

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/terraform"
	"github.com/radekg/terraform-provisioner-ansible/test"
)

func TestValidPrivateKeyWithExtraBytesDecrypts(t *testing.T) {
	key := `-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQCqGKukO1De7zhZj6+H0qtjTkVxwTCpvKe4eCZ0FPqri0cb2JZfXJ/DgYSF6vUp
wmJG8wVQZKjeGcjDOL5UlsuusFncCzWBQ7RKNUSesmQRMSGkVb1/3j+skZ6UtW+5u09lHNsj6tQ5
1s1SPrCBkedbNf0Tp0GbMJDyR4e9T04ZZwIDAQABAoGAFijko56+qGyN8M0RVyaRAXz++xTqHBLh
3tx4VgMtrQ+WEgCjhoTwo23KMBAuJGSYnRmoBZM3lMfTKevIkAidPExvYCdm5dYq3XToLkkLv5L2
pIIVOFMDG+KESnAFV7l2c+cnzRMW0+b6f8mR1CJzZuxVLL6Q02fvLi55/mbSYxECQQDeAw6fiIQX
GukBI4eMZZt4nscy2o12KyYner3VpoeE+Np2q+Z3pvAMd/aNzQ/W9WaI+NRfcxUJrmfPwIGm63il
AkEAxCL5HQb2bQr4ByorcMWm/hEP2MZzROV73yF41hPsRC9m66KrheO9HPTJuo3/9s5p+sqGxOlF
L0NDt4SkosjgGwJAFklyR1uZ/wPJjj611cdBcztlPdqoxssQGnh85BzCj/u3WqBpE2vjvyyvyI5k
X6zk7S0ljKtt2jny2+00VsBerQJBAJGC1Mg5Oydo5NwD6BiROrPxGo2bpTbu/fhrT8ebHkTz2epl
U9VQQSQzY1oZMVX8i1m5WUTLPz2yLJIBQVdXqhMCQBGoiuSoSjafUhV7i1cEGpb88h5NBYZzWXGZ
37sJ5QsW+sJyoNde3xH8vdXhzU7eT82D6X/scw9RZz+/6rCJ4p0=
-----END RSA PRIVATE KEY-----
with exytra bytes`
	err := validatePrivateKey(&key)
	if err != nil {
		t.Fatalf("Expected no error.")
	}
}

func TestValidPrivateKeyWithoutExtraBytesDecrypts(t *testing.T) {
	key := `-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQCqGKukO1De7zhZj6+H0qtjTkVxwTCpvKe4eCZ0FPqri0cb2JZfXJ/DgYSF6vUp
wmJG8wVQZKjeGcjDOL5UlsuusFncCzWBQ7RKNUSesmQRMSGkVb1/3j+skZ6UtW+5u09lHNsj6tQ5
1s1SPrCBkedbNf0Tp0GbMJDyR4e9T04ZZwIDAQABAoGAFijko56+qGyN8M0RVyaRAXz++xTqHBLh
3tx4VgMtrQ+WEgCjhoTwo23KMBAuJGSYnRmoBZM3lMfTKevIkAidPExvYCdm5dYq3XToLkkLv5L2
pIIVOFMDG+KESnAFV7l2c+cnzRMW0+b6f8mR1CJzZuxVLL6Q02fvLi55/mbSYxECQQDeAw6fiIQX
GukBI4eMZZt4nscy2o12KyYner3VpoeE+Np2q+Z3pvAMd/aNzQ/W9WaI+NRfcxUJrmfPwIGm63il
AkEAxCL5HQb2bQr4ByorcMWm/hEP2MZzROV73yF41hPsRC9m66KrheO9HPTJuo3/9s5p+sqGxOlF
L0NDt4SkosjgGwJAFklyR1uZ/wPJjj611cdBcztlPdqoxssQGnh85BzCj/u3WqBpE2vjvyyvyI5k
X6zk7S0ljKtt2jny2+00VsBerQJBAJGC1Mg5Oydo5NwD6BiROrPxGo2bpTbu/fhrT8ebHkTz2epl
U9VQQSQzY1oZMVX8i1m5WUTLPz2yLJIBQVdXqhMCQBGoiuSoSjafUhV7i1cEGpb88h5NBYZzWXGZ
37sJ5QsW+sJyoNde3xH8vdXhzU7eT82D6X/scw9RZz+/6rCJ4p0=
-----END RSA PRIVATE KEY-----`
	err := validatePrivateKey(&key)
	if err != nil {
		t.Fatalf("Expected no error.")
	}
}

func TestNotPrivateKeyFailsToDecrypt(t *testing.T) {
	key := `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCqGKukO1De7zhZj6+H0qtjTkVxwTCpvKe4eCZ0
FPqri0cb2JZfXJ/DgYSF6vUpwmJG8wVQZKjeGcjDOL5UlsuusFncCzWBQ7RKNUSesmQRMSGkVb1/
3j+skZ6UtW+5u09lHNsj6tQ51s1SPrCBkedbNf0Tp0GbMJDyR4e9T04ZZwIDAQAB
-----END PUBLIC KEY-----`
	err := validatePrivateKey(&key)
	if err == nil {
		t.Fatalf("Expected an error.")
	}
}

func TestEncryptedPrivateKeyFailsToDecrypt(t *testing.T) {
	key := `-----BEGIN RSA PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-256-CBC,1831C8FC5E508AF10642A9839E58A33E

NGK+djSC6GIVpzg5BilGp818Ec+4a9fR30BCUorEg+T9oBoltWXqJX+78B+7HFlu
OJp0Z7h6HdyojliTqYSXSYjSdjN4BXM1z8kFKWZ2rTl1ab/a+HG8blT7Bhc62aC/
7bfdkCPAPL0Y7lLqeJ/ci6I39+AoJbi13a3xVRPo9ZSuKXD3SYYEcaXO8vDPid0O
I3gXgcxBm7KAlF6KJj7+s5P8BBGBOj5/JWIb00wJSOemCbEstaA/SRg9RsinJJkg
R7eD2PMcb+9kvmUFqtCnoda0bxcwAWppZkeerNr1lva303EnZ1AVu56zwXfY8E1D
WHKJsdjxgjGEosQLy1aSVM9C0OU0Lgqxyq/Ns7j9GCB/SM5odU9VgwT+XYW33E8V
AirztP5cNfNCQH79IqsynWNa+8Lzxk+By/VevOJDlY20AVCk+va8NIE27fuxRdsa
iJSxu87NfnZv1rtrq3T1YmqhOC2rZFWSlQ8zesbasEoM6oEuOGLDAdM1TsxXJGeK
/KGZpF0Y/3Tv0EZvcVOldxUMOd+TOTSQPhoReo68UrZTuiCHyn/xKHX3YAB/Wzn9
GWBU1a2PrW4WyuE4Wy/+P3sG4jFnkM1W/Jg6DZp8Qi7eSk/DkbEASIleLM18ze2N
kYR69ClQcEw0fcvr4ZDrJXehEaGNbjoSqZxqoVbe6RXFdF4Fb4qp3eETu3FW/m/T
L+q7nOQiI49VQC2lOM/5RvxSuH7cp0ZD6QEvBTnkQtbtoDoRuaK3TgRgsMS4EhZS
lRcaj/CLlXBZEXCFWfp6/N5X1W+/9rmJoSiItqYVYQ0xdC5lCK6aE/HyyRJceR9Q
-----END RSA PRIVATE KEY-----`
	err := validatePrivateKey(&key)
	if err == nil {
		t.Fatalf("Expected an error.")
	}
}

func TestLocalConnectionExtractor(t *testing.T) {
	instanceState := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":                "ssh",
				"user":                "test-username",
				"password":            "test-password",
				"private_key":         test.TestSSHUserKeyPrivate,
				"host":                "127.0.0.1",
				"host_key":            test.TestSSHHostKeyPublic,
				"port":                "2022",
				"agent":               "true",
				"agent_identity":      "identity",
				"timeout":             "10m",
				"script_path":         "/tmp/script-path-%RAND%",
				"bastion_user":        "test-bastion-username",
				"bastion_password":    "test-bastion-password",
				"bastion_private_key": test.TestSSHUserKeyPrivate,
				"bastion_host":        "127.0.0.2",
				"bastion_host_key":    test.TestSSHHostKeyPublic,
				"bastion_port":        "2021",
			},
		},
	}

	connInfo, err := parseConnectionInfo(instanceState)
	if err != nil {
		t.Fatal("Expected connection info but received an error", err)
	}
	// agent:
	if !connInfo.Agent {
		t.Fatal("Expected connection info Agent true but got", connInfo.Agent)
	}
	if connInfo.AgentIdentity != instanceState.Ephemeral.ConnInfo["agent_identity"] {
		t.Fatalf("Expected connection info AgentIdentity %s but got %s", instanceState.Ephemeral.ConnInfo["agent_identity"], connInfo.AgentIdentity)
	}
	if connInfo.ScriptPath != instanceState.Ephemeral.ConnInfo["script_path"] {
		t.Fatalf("Expected connection info ScriptPath %s but got %s", instanceState.Ephemeral.ConnInfo["script_path"], connInfo.ScriptPath)
	}
	// bastion:
	if connInfo.BastionHost != instanceState.Ephemeral.ConnInfo["bastion_host"] {
		t.Fatalf("Expected connection info BastionHost %s but got %s", instanceState.Ephemeral.ConnInfo["bastion_host"], connInfo.BastionHost)
	}
	if connInfo.BastionHostKey != instanceState.Ephemeral.ConnInfo["bastion_host_key"] {
		t.Fatalf("Expected connection info BastionHostKey %s but got %s", instanceState.Ephemeral.ConnInfo["bastion_host_key"], connInfo.BastionHostKey)
	}
	if connInfo.BastionPassword != instanceState.Ephemeral.ConnInfo["bastion_password"] {
		t.Fatalf("Expected connection info BastionPassword %s but got %s", instanceState.Ephemeral.ConnInfo["bastion_password"], connInfo.BastionPassword)
	}
	if fmt.Sprintf("%d", connInfo.BastionPort) != instanceState.Ephemeral.ConnInfo["bastion_port"] {
		t.Fatalf("Expected connection info BastionPort %s but got %d", instanceState.Ephemeral.ConnInfo["bastion_port"], connInfo.BastionPort)
	}
	if flatString(connInfo.BastionPrivateKey) != flatString(instanceState.Ephemeral.ConnInfo["bastion_private_key"]) {
		t.Fatalf("Expected connection info BastionPrivateKey %s but got %s", instanceState.Ephemeral.ConnInfo["bastion_private_key"], connInfo.BastionPrivateKey)
	}
	if connInfo.BastionUser != instanceState.Ephemeral.ConnInfo["bastion_user"] {
		t.Fatalf("Expected connection info BastionUser %s but got %s", instanceState.Ephemeral.ConnInfo["bastion_user"], connInfo.BastionUser)
	}
	// host:
	if connInfo.Host != instanceState.Ephemeral.ConnInfo["host"] {
		t.Fatalf("Expected connection info Host %s but got %s", instanceState.Ephemeral.ConnInfo["host"], connInfo.Host)
	}
	if connInfo.HostKey != instanceState.Ephemeral.ConnInfo["host_key"] {
		t.Fatalf("Expected connection info HostKey %s but got %s", instanceState.Ephemeral.ConnInfo["host_key"], connInfo.HostKey)
	}
	if connInfo.Password != instanceState.Ephemeral.ConnInfo["password"] {
		t.Fatalf("Expected connection info Password %s but got %s", instanceState.Ephemeral.ConnInfo["password"], connInfo.Password)
	}
	if fmt.Sprintf("%d", connInfo.Port) != instanceState.Ephemeral.ConnInfo["port"] {
		t.Fatalf("Expected connection info Port %s but got %d", instanceState.Ephemeral.ConnInfo["port"], connInfo.Port)
	}
	if flatString(connInfo.PrivateKey) != flatString(instanceState.Ephemeral.ConnInfo["private_key"]) {
		t.Fatalf("Expected connection info PrivateKey %s but got %s", instanceState.Ephemeral.ConnInfo["private_key"], connInfo.PrivateKey)
	}
	if connInfo.User != instanceState.Ephemeral.ConnInfo["user"] {
		t.Fatalf("Expected connection info User %s but got %s", instanceState.Ephemeral.ConnInfo["user"], connInfo.User)
	}
}

func TestLocalConnectionExtractorBastionInherits(t *testing.T) {
	instanceState := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":           "ssh",
				"user":           "test-username",
				"password":       "test-password",
				"private_key":    test.TestSSHUserKeyPrivate,
				"host":           "127.0.0.1",
				"host_key":       test.TestSSHHostKeyPublic,
				"port":           "2022",
				"agent":          "true",
				"agent_identity": "identity",
				"timeout":        "10m",
				"script_path":    "/tmp/script-path-%RAND%",
				"bastion_host":   "127.0.0.2",
			},
		},
	}

	connInfo, err := parseConnectionInfo(instanceState)
	if err != nil {
		t.Fatal("Expected connection info but received an error", err)
	}
	// bastion:
	if connInfo.BastionHost != instanceState.Ephemeral.ConnInfo["bastion_host"] {
		t.Fatalf("Expected connection info BastionHost %s but got %s", instanceState.Ephemeral.ConnInfo["bastion_host"], connInfo.BastionHost)
	}
	if connInfo.BastionHostKey != "" {
		t.Fatalf("Expected connection info BastionHostKey to be empty but got %s", connInfo.BastionHostKey)
	}
	if connInfo.BastionPassword != instanceState.Ephemeral.ConnInfo["password"] {
		t.Fatalf("Expected connection info BastionPassword %s but got %s", instanceState.Ephemeral.ConnInfo["password"], connInfo.BastionPassword)
	}
	if fmt.Sprintf("%d", connInfo.BastionPort) != instanceState.Ephemeral.ConnInfo["port"] {
		t.Fatalf("Expected connection info BastionPort %s but got %d", instanceState.Ephemeral.ConnInfo["port"], connInfo.BastionPort)
	}
	if flatString(connInfo.BastionPrivateKey) != flatString(instanceState.Ephemeral.ConnInfo["private_key"]) {
		t.Fatalf("Expected connection info BastionPrivateKey %s but got %s", instanceState.Ephemeral.ConnInfo["private_key"], connInfo.BastionPrivateKey)
	}
	if connInfo.BastionUser != instanceState.Ephemeral.ConnInfo["user"] {
		t.Fatalf("Expected connection info BastionUser %s but got %s", instanceState.Ephemeral.ConnInfo["user"], connInfo.BastionUser)
	}
}

func TestInvalidDurationResultsInDefaultDuration(t *testing.T) {
	defaultDuration := time.Duration(time.Second * 5)
	returnedDuration := safeDuration("not a duration string", defaultDuration)
	if defaultDuration != returnedDuration {
		t.Fatalf("Expected default duration")
	}
}

func flatString(str string) string {
	return strings.Replace(str, "\n", "", -1)
}
