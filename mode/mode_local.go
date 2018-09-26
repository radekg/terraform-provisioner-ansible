package mode

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/radekg/terraform-provisioner-ansible/types"

	localExec "github.com/hashicorp/terraform/builtin/provisioners/local-exec"
	"github.com/hashicorp/terraform/terraform"
	uuid "github.com/satori/go.uuid"
)

type LocalMode struct {
	o        terraform.UIOutput
	connInfo *connectionInfo
}

func NewLocalMode(o terraform.UIOutput, s *terraform.InstanceState) (*LocalMode, error) {

	connType := s.Ephemeral.ConnInfo["type"]
	switch connType {
	case "ssh", "": // The default connection type is ssh, so if connType is empty use ssh
	default:
		return nil, fmt.Errorf("Currently, only SSH connection is supported")
	}

	connInfo, err := parseConnectionInfo(s)
	if err != nil {
		return nil, err
	}
	if connInfo.User == "" || connInfo.Host == "" {
		return nil, fmt.Errorf("Local mode requires a connection with username and host")
	}
	if connInfo.PrivateKey == "" {
		o.Output(fmt.Sprintf("no private key for %s@%s found, assuming ssh agent...", connInfo.User, connInfo.Host))
	}

	return &LocalMode{
		o:        o,
		connInfo: connInfo,
	}, nil
}

func (v *LocalMode) Run(plays []*types.Play, ansibleSSHSettings *types.AnsibleSSHSettings) error {

	pemFile := ""
	if v.connInfo.PrivateKey != "" {
		pemFile, err := v.writePem()
		if err != nil {
			return err
		}
		defer os.Remove(pemFile)
	}

	knownHostsFile, err := v.ensureKnownHosts(ansibleSSHSettings)
	if err != nil {
		return err
	}
	defer os.Remove(knownHostsFile)

	bastionHost := ""
	bastionUsername := v.connInfo.User
	bastionPemFile := pemFile
	bastionPort := v.connInfo.Port

	if v.connInfo.BastionHost != "" {
		bastionHost = v.connInfo.BastionHost
		if v.connInfo.BastionUser != "" {
			bastionUsername = v.connInfo.BastionUser
		}
		if v.connInfo.BastionPrivateKey != "" {
			bastionPemFile = v.connInfo.BastionPrivateKey
		}
		if v.connInfo.BastionPort > 0 {
			bastionPort = v.connInfo.BastionPort
		}
	}

	for _, play := range plays {

		if !play.Enabled() {
			continue
		}

		inventoryFile, err := v.writeInventory(play)

		if err != nil {
			v.o.Output(fmt.Sprintf("%+v", err))
			return err
		}

		if inventoryFile != play.InventoryFile() {
			play.SetOverrideInventoryFile(inventoryFile)
			defer os.Remove(play.InventoryFile())
		}

		command, err := play.ToLocalCommand(types.LocalModeAnsibleArgs{
			Username:        v.connInfo.User,
			Port:            v.connInfo.Port,
			PemFile:         pemFile,
			KnownHostsFile:  knownHostsFile,
			BastionHost:     bastionHost,
			BastionPemFile:  bastionPemFile,
			BastionPort:     bastionPort,
			BastionUsername: bastionUsername,
		}, ansibleSSHSettings)

		if err != nil {
			return err
		}

		if v.connInfo.BastionHost != "" {
			v.o.Output(fmt.Sprintf("executing ssh-keyscan on bastion: %s@%s", bastionUsername, fmt.Sprintf("%s:%d", bastionHost, bastionPort)))
			bastionSSHKeyScan := NewBastionKeyScan(
				bastionHost,
				bastionPort,
				bastionUsername,
				bastionPemFile)
			if err := bastionSSHKeyScan.Scan(v.o, v.connInfo.Host, v.connInfo.Port, ansibleSSHSettings); err != nil {
				return err
			}
		}

		v.o.Output(fmt.Sprintf("running local command: %s", command))

		if err := v.runCommand(command); err != nil {
			return err
		}

	}

	return nil
}

func (v *LocalMode) ensureKnownHosts(ansibleSSHSettings *types.AnsibleSSHSettings) (string, error) {

	if v.connInfo.Host == "" {
		return "", fmt.Errorf("Host could not be established from the connection info")
	}
	u1 := uuid.Must(uuid.NewV4())
	targetPath := filepath.Join(os.TempDir(), u1.String())

	startedAt := time.Now().Unix()
	timeoutSeconds := int64(ansibleSSHSettings.SSHKeyscanSeconds())

	for {
		sshKeyScanCommand := fmt.Sprintf("ssh-keyscan -p %d %s 2>/dev/null | head -n1 > \"%s\"", v.connInfo.Port, v.connInfo.Host, targetPath)
		if err := v.runCommand(sshKeyScanCommand); err != nil {
			return "", err
		}
		fi, err := os.Stat(targetPath)
		if err != nil {
			return "", err
		}
		if fi.Size() > 0 {
			break
		} else {
			v.o.Output("ssh-keyscan hasn't succeeded yet; retrying...")
			if time.Now().Unix()-startedAt > timeoutSeconds {
				return "", fmt.Errorf("ssh-keyscan %s:%d has not completed within the timeout of %d seconds", v.connInfo.Host, v.connInfo.Port, timeoutSeconds)
			}
		}
	}

	return targetPath, nil
}

func (v *LocalMode) writePem() (string, error) {
	if v.connInfo.PrivateKey != "" {
		file, err := ioutil.TempFile(os.TempDir(), "temporary-private-key.pem")
		defer file.Close()
		if err != nil {
			return "", err
		}

		v.o.Output(fmt.Sprintf("Writing temprary PEM to '%s'...", file.Name()))
		if err := ioutil.WriteFile(file.Name(), []byte(v.connInfo.PrivateKey), 0400); err != nil {
			return "", err
		}

		v.o.Output("Ansible inventory written.")
		return file.Name(), nil
	}
	return "", nil
}

func (v *LocalMode) writeInventory(play *types.Play) (string, error) {
	if play.InventoryFile() == "" {
		if v.connInfo.Host == "" {
			return "", fmt.Errorf("Host could not be established from the connection info")
		}

		/*
			$$ TODO: restore this functionality but without a template:
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
		*/
	}

	return play.InventoryFile(), nil
}

func (v *LocalMode) runCommand(command string) error {
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

	return localExecProvisioner.Apply(v.o, instanceState, config)
}
