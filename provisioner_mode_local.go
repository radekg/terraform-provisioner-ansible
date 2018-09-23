package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
	"time"

	localExec "github.com/hashicorp/terraform/builtin/provisioners/local-exec"
	"github.com/hashicorp/terraform/terraform"
	uuid "github.com/satori/go.uuid"
)

func (p *provisioner) localEnsureKnownHosts(o terraform.UIOutput, connInfo *connectionInfo) (string, error) {

	if connInfo.Host == "" {
		return "", fmt.Errorf("Host could not be established from the connection info")
	}
	u1 := uuid.Must(uuid.NewV4())
	targetPath := filepath.Join(os.TempDir(), u1.String())

	startedAt := time.Now().Unix()
	timeoutSeconds := int64(sshKeyScanTimeoutSeconds())

	for {
		sshKeyScanCommand := fmt.Sprintf("ssh-keyscan -p %d %s 2>/dev/null | head -n1 > \"%s\"", connInfo.Port, connInfo.Host, targetPath)
		if err := p.localRunCommand(o, sshKeyScanCommand); err != nil {
			return "", err
		}
		fi, err := os.Stat(targetPath)
		if err != nil {
			return "", err
		}
		if fi.Size() > 0 {
			break
		} else {
			o.Output("ssh-keyscan hasn't succeeded yet; retrying...")
			if time.Now().Unix()-startedAt > timeoutSeconds {
				return "", fmt.Errorf("ssh-keyscan %s:%d has not completed within the timeout of %d seconds", connInfo.Host, connInfo.Port, timeoutSeconds)
			}
		}
	}

	return targetPath, nil
}

func (p *provisioner) localWritePem(o terraform.UIOutput, connInfo *connectionInfo) (string, error) {
	if connInfo.PrivateKey != "" {
		file, err := ioutil.TempFile(os.TempDir(), "temporary-private-key.pem")
		defer file.Close()
		if err != nil {
			return "", err
		}

		o.Output(fmt.Sprintf("Writing temprary PEM to '%s'...", file.Name()))
		if err := ioutil.WriteFile(file.Name(), []byte(connInfo.PrivateKey), 0400); err != nil {
			return "", err
		}

		o.Output("Ansible inventory written.")
		return file.Name(), nil
	}
	return "", nil
}

func (p *provisioner) localGatherRunnables(o terraform.UIOutput, connInfo *connectionInfo) ([]runnablePlay, error) {

	response := make([]runnablePlay, 0)
	for _, playDef := range p.Plays {
		if playDef.Enabled == no {
			continue
		}
		if playDef.CallableType == ansibleCallablePlaybook {
			inventoryFile, err := p.localWriteInventory(o, connInfo, playDef.CallArgs, playDef.InventoryMeta)
			if err != nil {
				return response, err
			}
			response = append(response, runnablePlay{
				Play:                   playDef,
				VaultPasswordFile:      playDef.CallArgs.Shared.VaultPasswordFile,
				InventoryFile:          inventoryFile,
				InventoryFileTemporary: len(playDef.CallArgs.Shared.InventoryFile) == 0,
			})
		} else if playDef.CallableType == ansibleCallableModule {
			inventoryFile, err := p.localWriteInventory(o, connInfo, playDef.CallArgs, playDef.InventoryMeta)
			if err != nil {
				return response, err
			}
			response = append(response, runnablePlay{
				Play:                   playDef,
				VaultPasswordFile:      playDef.CallArgs.Shared.VaultPasswordFile,
				InventoryFile:          inventoryFile,
				InventoryFileTemporary: len(playDef.CallArgs.Shared.InventoryFile) == 0,
			})
		}
	}
	return response, nil

}

func (p *provisioner) localWriteInventory(o terraform.UIOutput, connInfo *connectionInfo, callArgs ansibleCallArgs, inventoryMeta ansibleInventoryMeta) (string, error) {
	if len(callArgs.Shared.InventoryFile) == 0 {
		if connInfo.Host == "" {
			return "", fmt.Errorf("Host could not be established from the connection info")
		}

		inplaceMeta := ansibleInventoryMeta{
			Hosts:  []string{connInfo.Host},
			Groups: inventoryMeta.Groups,
		}

		o.Output("Generating temporary ansible inventory...")
		t := template.Must(template.New("hosts").Parse(inventoryTemplateLocal))
		var buf bytes.Buffer
		err := t.Execute(&buf, inplaceMeta)
		if err != nil {
			return "", fmt.Errorf("Error executing 'hosts' template: %s", err)
		}

		file, err := ioutil.TempFile(os.TempDir(), "temporary-ansible-inventory")
		defer file.Close()
		if err != nil {
			return "", err
		}

		o.Output(fmt.Sprintf("Writing temporary ansible inventory to '%s'...", file.Name()))
		if err := ioutil.WriteFile(file.Name(), buf.Bytes(), 0644); err != nil {
			return "", err
		}

		o.Output("Ansible inventory written.")
		return file.Name(), nil

	}

	return callArgs.Shared.InventoryFile, nil
}

func (p *provisioner) localRunCommand(o terraform.UIOutput, command string) error {
	localExecProvisioner := localExec.Provisioner()

	instanceState := &terraform.InstanceState{
		ID:         command,
		Attributes: make(map[string]string),
		Ephemeral: terraform.EphemeralState{
			ConnInfo: make(map[string]string),
			Type:     "local-exec",
		},
		Meta: map[string]interface{}{
			"command": command,
		},
		Tainted: false,
	}

	config := &terraform.ResourceConfig{
		ComputedKeys: make([]string, 0),
		Raw: map[string]interface{}{
			"command": command,
		},
		Config: map[string]interface{}{
			"command": command,
		},
	}

	return localExecProvisioner.Apply(o, instanceState, config)
}
