package mode

import (
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/radekg/terraform-provisioner-ansible/test"
)

func TestBastionHostConfiguration(t *testing.T) {
	instanceState := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":                "ssh",
				"user":                "test-username",
				"password":            "test-password",
				"private_key":         test.TestSSHUserKeyPrivate,
				"host":                "127.0.0.1",
				"host_key":            test.TestSSHHostKeyPublic,
				"port":                "0",
				"agent":               "true",
				"agent_identity":      "identity",
				"timeout":             "10m",
				"script_path":         "/tmp/script-path-%RAND%",
				"bastion_user":        "test-username", // must be the same as user for this test
				"bastion_password":    "test-bastion-password",
				"bastion_private_key": test.TestSSHUserKeyPrivate,
				"bastion_host":        "127.0.0.1", // must be the same as host for this test
				"bastion_host_key":    "",
				"bastion_port":        "0", // must be the same as port for this test
			},
		},
	}

	output := new(terraform.MockUIOutput)
	sshServer := test.GetConfiguredAndRunningSSHServer(t, "ssh-bastion-host", false, instanceState, output)
	defer sshServer.Stop()

	_, p, err := sshServer.ListeningHostPort()
	if err != nil {
		t.Fatal("Expected a port from SSH server")
	}

	instanceState.Ephemeral.ConnInfo["port"] = p
	instanceState.Ephemeral.ConnInfo["bastion_port"] = p

	connInfo, err := parseConnectionInfo(instanceState)
	if err != nil {
		t.Fatal("Expected connection info but got an error", err)
	}

	bh := newBastionHostFromConnectionInfo(connInfo)
	if bh.agent() != connInfo.Agent {
		t.Fatal("Expected values to match", bh.agent(), connInfo.Agent)
	}
	if bh.host() != connInfo.BastionHost {
		t.Fatal("Expected values to match", bh.host(), connInfo.BastionHost)
	}
	if bh.port() != connInfo.BastionPort {
		t.Fatal("Expected values to match", bh.port(), connInfo.BastionPort)
	}
	if flatString(bh.pemFile()) != flatString(connInfo.BastionPrivateKey) {
		t.Fatal("Expected values to match", bh.pemFile(), connInfo.BastionPrivateKey)
	}
	if bh.hostKey() != connInfo.BastionHostKey {
		t.Fatal("Expected values to match", bh.hostKey(), connInfo.BastionHostKey)
	}

	sshClient, err := bh.connect()
	if err != nil {
		t.Fatal("Expected sshClient but reeceived an error", err)
	}
	sshClient.Close()
}
