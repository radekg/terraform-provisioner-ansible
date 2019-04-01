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

func TestLocalInventoryTemplateGeneratesWithAlias(t *testing.T) {

	templateData := inventoryTemplateLocalData{
		Hosts: []inventoryTemplateLocalDataHost{
			inventoryTemplateLocalDataHost{
				Alias:       "testBox",
				AnsibleHost: "10.1.100.34",
			},
		},
		Groups: []string{"group1"},
	}

	tpl := template.Must(template.New("hosts").Parse(inventoryTemplateLocal))
	var buf bytes.Buffer
	err := tpl.Execute(&buf, templateData)
	if err != nil {
		t.Fatalf("Expected template to generate correctly but received: %v", err)
	}
	templateBody := buf.String()
	if strings.Index(templateBody, fmt.Sprintf("[%s]\n%s ansible_host",
		templateData.Groups[0],
		templateData.Hosts[0].Alias)) < 0 {
		t.Fatalf("Expected a group with alias in generated template but got:\n%s", templateBody)
	}
}

func TestLocalInventoryTemplateGeneratesWithoutAlias(t *testing.T) {

	// please refer to mode_local.go writeInventory for details:
	templateData := inventoryTemplateLocalData{
		Hosts: []inventoryTemplateLocalDataHost{
			inventoryTemplateLocalDataHost{
				Alias: "10.1.100.34",
			},
		},
		Groups: []string{"group1"},
	}

	tpl := template.Must(template.New("hosts").Parse(inventoryTemplateLocal))
	var buf bytes.Buffer
	err := tpl.Execute(&buf, templateData)
	if err != nil {
		t.Fatalf("Expected template to generate correctly but received: %v", err)
	}

	templateBody := buf.String()
	if strings.Index(templateBody, fmt.Sprintf("[%s]\n%s",
		templateData.Groups[0],
		templateData.Hosts[0].Alias)) < 0 {
		t.Fatalf("Expected a group with alias in generated template but got: %s", templateBody)
	}
	if strings.Index(templateBody, "ansible_host") > -1 {
		t.Fatalf("Did not expect ansible_host in generated template but got: %s", templateBody)
	}

}

func TestIntegrationLocalModeProvisioning(t *testing.T) {

	testModuleName := "ping"
	sshUsername := "integration-test"
	output := new(terraform.MockUIOutput)
	user := test.GetCurrentUser(t)

	instanceState := test.GetNewSSHInstanceState(t, sshUsername)
	sshServer := test.GetConfiguredAndRunningSSHServer(t, "local-provisioning", true, instanceState, output)
	defer sshServer.Stop()

	// temp vault-id:
	tempVaultIDFilePath := test.WriteTempVaultIDFile(t, "test-password")
	defer os.Remove(tempVaultIDFilePath)

	// temp playbook:
	tempAnsibleDataDir := test.CreateTempAnsibleDataDirectory(t)
	defer os.RemoveAll(tempAnsibleDataDir)

	playbookFilePath := test.WriteTempPlaybook(t, tempAnsibleDataDir)

	// temp remote_tmp:
	tempRemoteTmp := test.CreateTempAnsibleRemoteTmpDir(t)
	defer os.RemoveAll(tempRemoteTmp)
	os.Setenv("ANSIBLE_REMOTE_TMP", tempRemoteTmp)
	defer os.Unsetenv("ANSIBLE_REMOTE_TMP")

	defaultSettings := test.GetDefaultSettingsForUser(t, user)
	playModuleRawConfigs := test.GetPlayModuleSchema(t, testModuleName)
	playPlaybookRawConfigs := test.GetPlayPlaybookSchema(t, playbookFilePath)

	playModule := map[string]interface{}{
		"hosts":               []interface{}{"integrationTest"},
		"enabled":             true,
		"become":              false,
		"become_method":       defaultSettings.BecomeMethod(),
		"become_user":         defaultSettings.BecomeUser(),
		"diff":                false,
		"check":               false,
		"forks":               5,
		"inventory_file":      "",
		"limit":               "",
		"vault_id":            []interface{}{tempVaultIDFilePath},
		"vault_password_file": "",
		"verbose":             true,
		"extra_vars": map[string]interface{}{
			"var1": "value1",
			"var2": 100,
		},
		"module":   playModuleRawConfigs.Get("module").(*schema.Set),
		"playbook": playModuleRawConfigs.Get("playbook").(*schema.Set),
	}

	playPlaybook := map[string]interface{}{
		"hosts":               []interface{}{"integrationTest"},
		"enabled":             true,
		"become":              false,
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

	modeLocal, err := NewLocalMode(output, instanceState)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		runErr := modeLocal.Run([]*types.Play{
			test.GetNewPlay(t, playModule, defaultSettings),
			test.GetNewPlay(t, playPlaybook, defaultSettings),
		}, types.NewAnsibleSSHSettingsFromInterface("", false /* just take defaults */))
		if runErr != nil {
			t.Fatalf("Unexpected error: %v", runErr)
		}
	}()

	// We can only test the workflow with ANSIBLE_REMOTE_TMP set and without become:

	// Module:
	// Ansible creates a directory in ANSIBLE_REMOTE_TMP dirctory:
	test.CommandTest(t, sshServer, fmt.Sprintf("/bin/sh -c '( umask 77 && mkdir -p \"` echo %s", tempRemoteTmp))
	// ... then it chmod u+x it ...
	test.CommandTest(t, sshServer, fmt.Sprintf("/bin/sh -c 'chmod u+x %s", tempRemoteTmp))
	// ... the module is executed:
	test.CommandTest(t, sshServer, fmt.Sprintf("/bin/sh -c '/usr/bin/python %s", tempRemoteTmp))
	// ... and Ansible cleans up after module execution.
	test.CommandTest(t, sshServer, fmt.Sprintf("/bin/sh -c 'rm -f -r %s", tempRemoteTmp))

	// Playbook:
	test.CommandTest(t, sshServer, fmt.Sprintf("/bin/sh -c '( umask 77 && mkdir -p \"` echo %s", tempRemoteTmp))
	// ... then it chmod u+x on the setup.py ...
	test.CommandTest(t, sshServer, fmt.Sprintf("/bin/sh -c 'chmod u+x %s", tempRemoteTmp))
	// ... the setup.py is executed ...
	test.CommandTest(t, sshServer, fmt.Sprintf("/bin/sh -c '/usr/bin/python %s", tempRemoteTmp))
	// ... and Ansible cleans up after playbook execution.
	test.CommandTest(t, sshServer, fmt.Sprintf("/bin/sh -c 'rm -f -r %s", tempRemoteTmp))

	wg.Wait()

}
