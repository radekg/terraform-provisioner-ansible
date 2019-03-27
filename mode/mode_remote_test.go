package mode

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/hashicorp/terraform/terraform"
	"github.com/radekg/terraform-provisioner-ansible/test"
	"github.com/radekg/terraform-provisioner-ansible/types"
)

func TestRemoteInventTemplateGenerates(t *testing.T) {
	originalHosts := []string{"host1", "host2"}
	templateData := inventoryTemplateRemoteData{
		Hosts:  ensureLocalhostInHosts(originalHosts),
		Groups: []string{"group1", "group2"},
	}

	tpl := template.Must(template.New("hosts").Parse(inventoryTemplateRemote))
	var buf bytes.Buffer
	err := tpl.Execute(&buf, templateData)
	if err != nil {
		t.Fatalf("Expected template to generate correctly but received: %v", err)
	}
	templateBody := buf.String()
	if strings.Index(templateBody, fmt.Sprintf("[%s]\nlocalhost ansible_connection=local\n%s",
		templateData.Groups[0],
		originalHosts[0])) < 0 {
		t.Fatalf("Expected a group with alias in generated template but got: %s", templateBody)
	}
}

func TestIntegrationRemoteModeProvisioning(t *testing.T) {

	remoteTempDirectory := "/remote-temp"
	bootstrapDirectory := "/bootstrap"
	testModuleName := "module-name"

	output := new(terraform.MockUIOutput)

	instanceState := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":        "ssh",
				"user":        "integration-test",
				"host":        "127.0.0.1",
				"port":        "22022",
				"private_key": test.TestSSHUserKeyPrivate,
				"host_key":    test.TestSSHHostKeyPublic,
			},
		},
	}

	authUser := &test.TestingSSHUser{
		Username:  instanceState.Ephemeral.ConnInfo["user"],
		PublicKey: test.TestSSHUserKeyPublic,
	}
	sshConfig := &test.TestingSSHServerConfig{
		ServerID:           "remote-provisioning",
		HostKey:            test.TestSSHHostKeyPrivate,
		HostPort:           fmt.Sprintf("%s:%s", instanceState.Ephemeral.ConnInfo["host"], instanceState.Ephemeral.ConnInfo["port"]),
		AuthenticatedUsers: []*test.TestingSSHUser{authUser},
		Listeners:          5,
		Output:             output,
		LogPrintln:         true,
	}
	sshServer := test.NewTestingSSHServer(t, sshConfig)
	go sshServer.Start()
	defer sshServer.Stop()

	select {
	case <-sshServer.ReadyNotify():
	case <-time.After(5 * time.Second):
		t.Fatal("Expected the TestingSSHServer to be running.")
	}

	remoteSettings := map[string]interface{}{
		"skip_install":               false,
		"use_sudo":                   true,
		"skip_cleanup":               false,
		"install_version":            "ansible@integration-test",
		"local_installer_path":       "",
		"remote_installer_directory": remoteTempDirectory,
		"bootstrap_directory":        bootstrapDirectory,
	}

	defaultSettings := map[string]interface{}{
		"become_method": "sudo",
		"become_user":   "test-user",
	}

	playModuleEntity := map[string]interface{}{
		"module": []map[string]interface{}{
			map[string]interface{}{
				"module": testModuleName,
			},
		},
		"playbook": []map[string]interface{}{},
	}
	playModuleSchemas := map[string]*schema.Schema{
		"module":   types.NewModuleSchema(),
		"playbook": types.NewPlaybookSchema(),
	}
	playModuleRawConfigs := schema.TestResourceDataRaw(t, playModuleSchemas, playModuleEntity)

	playModule := map[string]interface{}{
		"enabled":             true,
		"become":              true,
		"become_method":       "sudo",
		"become_user":         "test-user",
		"diff":                false,
		"check":               false,
		"forks":               5,
		"inventory_file":      "",
		"limit":               "",
		"vault_id":            []interface{}{}, // TODO: add testing with vault id
		"vault_password_file": "",
		"verbose":             false,
		"extra_vars": map[string]interface{}{
			"var1": "value1",
			"var2": 100,
		},
		"module":   playModuleRawConfigs.Get("module").(*schema.Set),
		"playbook": playModuleRawConfigs.Get("playbook").(*schema.Set),
	}

	modeRemote, err := NewRemoteMode(output, instanceState, types.NewRemoteSettingsFromMapInterface(remoteSettings, true))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		runErr := modeRemote.Run([]*types.Play{
			types.NewPlayFromMapInterface(playModule, types.NewDefaultsFromMapInterface(defaultSettings, true)),
		})
		if runErr != nil {
			t.Fatalf("Unexpected error: %v", runErr)
		}
	}()

	// upload ansible data:
	testForCommand(t, sshServer, fmt.Sprintf("mkdir -p \"%s", bootstrapDirectory))
	testForCommand(t, sshServer, fmt.Sprintf("scp -vt %s", bootstrapDirectory))
	// upload installer:
	testForCommand(t, sshServer, fmt.Sprintf("mkdir -p \"%s", remoteTempDirectory))
	testForCommand(t, sshServer, fmt.Sprintf("scp -vt %s", remoteTempDirectory))
	// make the installer executable:
	testForCommand(t, sshServer, "chmod 0777")
	// run and cleanup ansible installer:
	testForCommand(t, sshServer, "sudo /bin/sh -c")
	// run ansible module:
	testForCommand(t, sshServer, fmt.Sprintf("sudo ANSIBLE_FORCE_COLOR=true ansible all --module-name='%s'", testModuleName))
	// cleanup ansible data:
	testForCommand(t, sshServer, fmt.Sprintf("rm -rf \"%s", bootstrapDirectory))

	wg.Wait()

}

func testForCommand(t *testing.T, sshServer *test.TestingSSHServer, commandPrefix string) {
	select {
	case event := <-sshServer.Notifications():
		switch tevent := event.(type) {
		case test.NotificationCommandExecuted:
			if !strings.HasPrefix(tevent.Command, commandPrefix) {
				t.Fatalf("Expected a command starting with '%s' received: '%s'", commandPrefix, tevent.Command)
			}
		default:
			t.Fatal("Expected a command execution but received", tevent)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Excepted a notification from the SSH server.")
	}
}
