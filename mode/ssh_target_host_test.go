package mode

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/radekg/terraform-provisioner-ansible/test"
)

func TestTargetHostConfiguration(t *testing.T) {
	instanceState := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":           "ssh",
				"user":           "test-username",
				"password":       "test-password",
				"private_key":    test.TestSSHUserKeyPrivate,
				"host":           "127.0.0.1",
				"host_key":       test.TestSSHHostKeyPublic,
				"port":           "0",
				"agent":          "true",
				"agent_identity": "identity",
				"timeout":        "10m",
				"script_path":    "/tmp/script-path-%RAND%",
				"bastion_host":   "127.0.0.2",
			},
		},
	}

	output := new(terraform.MockUIOutput)
	sshServer := test.GetConfiguredAndRunningSSHServer(t, "ssh-target-host", false, instanceState, output)
	defer sshServer.Stop()

	_, p, err := sshServer.ListeningHostPort()
	if err != nil {
		t.Fatal("Expected a port from SSH server")
	}

	instanceState.Ephemeral.ConnInfo["port"] = p
	instanceState.Ephemeral.ConnInfo["host_key"] = ""

	connInfo, err := parseConnectionInfo(instanceState)
	if err != nil {
		t.Fatal("Expected connection info but got an error", err)
	}
	th := newTargetHostFromConnectionInfo(connInfo)
	if th.agent() != connInfo.Agent {
		t.Fatal("Expected values to match", th.agent(), connInfo.Agent)
	}
	if th.host() != connInfo.Host {
		t.Fatal("Expected values to match", th.host(), connInfo.Host)
	}
	if th.port() != connInfo.Port {
		t.Fatal("Expected values to match", th.port(), connInfo.Port)
	}
	if flatString(th.pemFile()) != flatString(connInfo.PrivateKey) {
		t.Fatal("Expected values to match", th.pemFile(), connInfo.PrivateKey)
	}
	if th.hostKey() != connInfo.HostKey {
		t.Fatal("Expected values to match", th.hostKey(), connInfo.HostKey)
	}

	fetchErr := th.fetchHostKey()
	if fetchErr != nil {
		t.Fatal("Expected fetchHostKey to succeed.", fetchErr)
	}

}
