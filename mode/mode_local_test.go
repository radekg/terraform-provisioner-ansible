package mode

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
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

	remoteTempDirectory := "/remote-temp"
	bootstrapDirectory := "/bootstrap"
	testModuleName := "ping"

	output := new(terraform.MockUIOutput)

	instanceState := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":        "ssh",
				"user":        "integration-test",
				"host":        "127.0.0.1",
				"port":        "0", // will be set later
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
		ServerID:           "local-provisioning",
		HostKey:            test.TestSSHHostKeyPrivate,
		HostPort:           fmt.Sprintf("%s:%s", instanceState.Ephemeral.ConnInfo["host"], instanceState.Ephemeral.ConnInfo["port"]),
		AuthenticatedUsers: []*test.TestingSSHUser{authUser},
		Listeners:          5,
		Output:             output,
		LogPrintln:         true,
		LocalMode:          true,
	}
	sshServer := test.NewTestingSSHServer(t, sshConfig)
	go sshServer.Start()
	defer sshServer.Stop()

	select {
	case <-sshServer.ReadyNotify():
	case <-time.After(5 * time.Second):
		t.Fatal("Expected the TestingSSHServer to be running.")
	}

	// we need to update the instance info with the address the SSH server is bound on:
	h, p, err := sshServer.ListeningHostPort()
	if err != nil {
		t.Fatal("Expected the SSH server to return an address it is bound on but got an error", err)
	}

	// set connection details based on where the SSH server is bound:
	instanceState.Ephemeral.ConnInfo["host"] = h
	instanceState.Ephemeral.ConnInfo["port"] = p

	// temp vault-id:
	tempVaultIDFile, err := ioutil.TempFile("", ".temp-vault-id")
	if err != nil {
		t.Fatal("Expected a temp vault id file to be crated", err)
	}
	defer os.Remove(tempVaultIDFile.Name())
	tempVaultIDFileToWrite, err := os.OpenFile(tempVaultIDFile.Name(), os.O_RDWR, 0644)
	if err != nil {
		t.Fatal("Expected a temp vault id file to be writable", err)
	}
	tempVaultIDFileToWrite.WriteString("test-password")
	tempVaultIDFileToWrite.Close()

	// temp playbook:
	tempAnsibleDataDir := test.CreateTempAnsibleDataDirectory(t)
	defer os.RemoveAll(tempAnsibleDataDir)
	playbookFilePath := test.WriteTempPlaybookFile(t, tempAnsibleDataDir)

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

	playPlaybookEntity := map[string]interface{}{
		"playbook": []map[string]interface{}{
			map[string]interface{}{
				"file_path": playbookFilePath,
			},
		},
		"module": []map[string]interface{}{},
	}

	playEntitySchemas := map[string]*schema.Schema{
		"module":   types.NewModuleSchema(),
		"playbook": types.NewPlaybookSchema(),
	}

	playModuleRawConfigs := schema.TestResourceDataRaw(t, playEntitySchemas, playModuleEntity)
	playPlaybookRawConfigs := schema.TestResourceDataRaw(t, playEntitySchemas, playPlaybookEntity)

	playModule := map[string]interface{}{
		"hosts":               []interface{}{"integrationTest"},
		"enabled":             true,
		"become":              true,
		"become_method":       "sudo",
		"become_user":         "test-user",
		"diff":                false,
		"check":               false,
		"forks":               5,
		"inventory_file":      "",
		"limit":               "",
		"vault_id":            []interface{}{tempVaultIDFile.Name()},
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
		"become":              true,
		"become_method":       "sudo",
		"become_user":         "test-user",
		"diff":                false,
		"check":               false,
		"forks":               5,
		"inventory_file":      "",
		"limit":               "",
		"vault_id":            []interface{}{tempVaultIDFile.Name()},
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
			types.NewPlayFromMapInterface(playModule, types.NewDefaultsFromMapInterface(defaultSettings, true)),
			types.NewPlayFromMapInterface(playPlaybook, types.NewDefaultsFromMapInterface(defaultSettings, true)),
		}, types.NewAnsibleSSHSettingsFromInterface("", false /* just take defaults */))
		if runErr != nil {
			t.Fatalf("Unexpected error: %v", runErr)
		}
	}()

	// Ansible starts with a handshake:
	test.CommandTest(t, sshServer, fmt.Sprintf("/bin/sh -c 'echo ~%s && sleep 0'", instanceState.Ephemeral.ConnInfo["user"]))
	// then it creates a temp directory:
	test.CommandTest(t, sshServer, "/bin/sh -c '( umask 77 && mkdir -p \"` echo /var/tmp/ansible-tmp-")
	// Ansible writes a module file:
	test.CommandTest(t, sshServer, "scp -t /var/tmp/ansible-tmp-")

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
