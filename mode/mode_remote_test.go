package mode

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"text/template"

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

	remoteTempDirectory := test.CreateTempAnsibleRemoteTmpDir(t)
	defer os.RemoveAll(remoteTempDirectory)
	bootstrapDirectory := test.CreateTempAnsibleBootstrapDir(t)
	defer os.RemoveAll(bootstrapDirectory)
	sshUsername := "integration-test"
	testModuleName := "ping"

	remoteSettingsRaw := map[string]interface{}{
		"skip_install":               false,
		"use_sudo":                   true,
		"skip_cleanup":               false,
		"install_version":            "ansible@integration-test",
		"local_installer_path":       "",
		"remote_installer_directory": remoteTempDirectory,
		"bootstrap_directory":        bootstrapDirectory,
	}

	output := new(terraform.MockUIOutput)
	user := test.GetCurrentUser(t)

	instanceState := test.GetNewSSHInstanceState(t, sshUsername)
	sshServer := test.GetConfiguredAndRunningSSHServer(t, "remote-provisioning", false, instanceState, output)
	defer sshServer.Stop()

	// temp vault-id:
	tempVaultIDFilePath := test.WriteTempVaultIDFile(t, "test-password")
	defer os.Remove(tempVaultIDFilePath)

	// temp playbook:
	tempAnsibleDataDir := test.CreateTempAnsibleDataDirectory(t)
	defer os.RemoveAll(tempAnsibleDataDir)
	playbookFilePath := test.WriteTempPlaybook(t, tempAnsibleDataDir)

	remoteSettings := test.GetNewRemoteSettings(t, remoteSettingsRaw)
	defaultSettings := test.GetDefaultSettingsForUser(t, user)
	playModuleRawConfigs := test.GetPlayModuleSchema(t, testModuleName)
	playPlaybookRawConfigs := test.GetPlayPlaybookSchema(t, playbookFilePath)

	playModule := map[string]interface{}{
		"enabled":             true,
		"become":              true,
		"become_method":       defaultSettings.BecomeMethod(),
		"become_user":         defaultSettings.BecomeUser(),
		"diff":                false,
		"check":               false,
		"forks":               5,
		"inventory_file":      "",
		"limit":               "",
		"vault_id":            []interface{}{tempVaultIDFilePath},
		"vault_password_file": "",
		"verbose":             false,
		"extra_vars": map[string]interface{}{
			"var1": "value1",
			"var2": 100,
		},
		"module":   playModuleRawConfigs.Get("module").(*schema.Set),
		"playbook": playModuleRawConfigs.Get("playbook").(*schema.Set),
	}

	playPlaybook := map[string]interface{}{
		"enabled":             true,
		"become":              true,
		"become_method":       defaultSettings.BecomeMethod(),
		"become_user":         defaultSettings.BecomeUser(),
		"diff":                false,
		"check":               false,
		"forks":               5,
		"inventory_file":      "",
		"limit":               "",
		"vault_id":            []interface{}{tempVaultIDFilePath},
		"vault_password_file": "",
		"verbose":             false,
		"extra_vars": map[string]interface{}{
			"var1": "value1",
			"var2": 100,
		},
		"module":   playModuleRawConfigs.Get("module").(*schema.Set),
		"playbook": playPlaybookRawConfigs.Get("playbook").(*schema.Set),
	}

	modeRemote, err := NewRemoteMode(output, instanceState, remoteSettings)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		runErr := modeRemote.Run([]*types.Play{
			types.NewPlayFromMapInterface(playModule, defaultSettings),
			types.NewPlayFromMapInterface(playPlaybook, defaultSettings),
		})
		if runErr != nil {
			t.Fatalf("Unexpected error: %v", runErr)
		}
	}()

	// upload ansible data for th first play:
	test.CommandTest(t, sshServer, fmt.Sprintf("mkdir -p \"%s", bootstrapDirectory))
	test.CommandTest(t, sshServer, fmt.Sprintf("scp -vt %s", bootstrapDirectory))
	// upload vault ID for the first play:
	test.CommandTest(t, sshServer, fmt.Sprintf("scp -vt %s", bootstrapDirectory))

	// upload ansible data for the second play:
	test.CommandTest(t, sshServer, fmt.Sprintf("mkdir -p \"%s", bootstrapDirectory))
	test.CommandTest(t, sshServer, "/bin/sh -c 'if [ -d") // playbook always checks if we have the source playbook dir uploaded
	test.CommandTest(t, sshServer, fmt.Sprintf("scp -rvt %s", bootstrapDirectory))
	test.CommandTest(t, sshServer, fmt.Sprintf("scp -vt %s", bootstrapDirectory)) // an inventory is written
	// upload vault ID for the second play:
	test.CommandTest(t, sshServer, fmt.Sprintf("scp -vt %s", bootstrapDirectory))

	// upload installer:
	test.CommandTest(t, sshServer, fmt.Sprintf("mkdir -p \"%s", remoteTempDirectory))
	test.CommandTest(t, sshServer, fmt.Sprintf("scp -vt %s", remoteTempDirectory))
	// make the installer executable:
	test.CommandTest(t, sshServer, "chmod 0777")
	// run and cleanup ansible installer:
	test.CommandTest(t, sshServer, "sudo /bin/sh -c")

	// run ansible module:
	test.CommandTest(t, sshServer, fmt.Sprintf("sudo ANSIBLE_FORCE_COLOR=true ansible all --module-name='%s'", testModuleName))
	test.CommandTest(t, sshServer, "sudo ANSIBLE_FORCE_COLOR=true ansible-playbook")

	// cleanup ansible data:
	test.CommandTest(t, sshServer, fmt.Sprintf("rm -rf \"%s", bootstrapDirectory))

	wg.Wait()

}
