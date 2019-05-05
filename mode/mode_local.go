package mode

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/radekg/terraform-provisioner-ansible/types"
	uuid "github.com/satori/go.uuid"

	localExec "github.com/hashicorp/terraform/builtin/provisioners/local-exec"
	"github.com/hashicorp/terraform/terraform"
)

// LocalMode represents local provisioner mode.
type LocalMode struct {
	o        terraform.UIOutput
	connInfo *connectionInfo
}

type inventoryTemplateLocalDataHost struct {
	Alias       string
	AnsibleHost string
}

type inventoryTemplateLocalData struct {
	Hosts  []inventoryTemplateLocalDataHost
	Groups []string
}

const inventoryTemplateLocal = `{{$top := . -}}
{{range .Hosts -}}
{{.Alias -}}
{{if ne .AnsibleHost "" -}}
{{" "}}ansible_host={{.AnsibleHost -}}
{{end -}}
{{printf "\n" -}}
{{end}}

{{range .Groups -}}
[{{.}}]
{{range $top.Hosts -}}
{{.Alias -}}
{{if ne .AnsibleHost "" -}}
{{" "}}ansible_host={{.AnsibleHost -}}
{{end -}}
{{printf "\n" -}}
{{end}}

{{end}}`

// NewLocalMode returns configured local mode provisioner.
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

	// Checks on connInfo unnecessary
	// connInfo.User defaulted to "root" by Terraform
	// connInfo.Host always populated when running under compute resource. 

	return &LocalMode{
		o:        o,
		connInfo: connInfo,
	}, nil
}

func (v *LocalMode) ComputeResource() bool {
	if v.connInfo.Host != "" {
		return true
	} else {
		return false
	}
}


// Run executes local provisioning process.
func (v *LocalMode) Run(plays []*types.Play, ansibleSSHSettings *types.AnsibleSSHSettings) error {

	// Validate config for null_resource
	compute_resource := v.ComputeResource()
	if !compute_resource  {
		for _, play := range plays {
			if len(play.Hosts()) == 0 && play.InventoryFile() == "" {
				return fmt.Errorf("Hosts or Inventory file must be specified on each plays attribute when using null_resource")
			}
		}	
		// Force StrictHostKeyChecking=no for null_resource
		ansibleSSHSettings.SetOverrideStrictHostKeyChecking()					
	}		

	bastionPemFile := ""
	if v.connInfo.BastionPrivateKey != "" {
		var err error
		bastionPemFile, err = v.writePem(v.connInfo.BastionPrivateKey)
		if err != nil {
			return err
		}
		defer os.Remove(bastionPemFile)
	}

	targetPemFile := ""
	if v.connInfo.PrivateKey != "" {
		var err error
		targetPemFile, err = v.writePem(v.connInfo.PrivateKey)
		if err != nil {
			return err
		}

		defer os.Remove(targetPemFile)
	}

	bastion := newBastionHostFromConnectionInfo(v.connInfo)
	target := newTargetHostFromConnectionInfo(v.connInfo)

	knownHostsTarget := make([]string, 0)
	knownHostsBastion := make([]string, 0)

	if bastion.inUse() {
		// wait for bastion:
		sshClient, err := bastion.connect()
		if err != nil {
			return err
		}
		defer sshClient.Close()
		if !ansibleSSHSettings.InsecureNoStrictHostKeyChecking() {
			if ansibleSSHSettings.UserKnownHostsFile() == "" {
				if target.hostKey() == "" {
					v.o.Output(fmt.Sprintf("Host key not given, executing ssh-keyscan on bastion: %s@%s:%d",
						bastion.user(),
						bastion.host(),
						bastion.port()))
					targetKnownHosts, err := newBastionKeyScan(v.o,
						sshClient,
						target.host(),
						target.port(),
						ansibleSSHSettings.SSHKeyscanSeconds()).scan()
					if err != nil {
						return err
					}
					// ssh-keyscan gave us full lines with hosts, like this:
					// <ip> ecdsa-sha2-nistp256 AAAA...
					// <ip> ssh-rsa AAAAB...
					// <ip> ssh-ed25519 AAAAC...
					knownHostsTarget = append(knownHostsTarget, targetKnownHosts)
				} else {
					knownHostsTarget = append(knownHostsTarget, fmt.Sprintf("%s %s", target.host(), target.hostKey()))
				}
			} else {
				v.o.Output(fmt.Sprintf("bastion %s@%s:%d will use '%s' as a user known hosts file",
					bastion.user(),
					bastion.host(),
					bastion.port(),
					ansibleSSHSettings.UserKnownHostsFile()))
			}

		} else {
			v.o.Output(fmt.Sprintf("target host StrictHostKeyChecking=no, not verifying host keys on bastion: %s@%s:%d",
				bastion.user(),
				bastion.host(),
				bastion.port()))
		}
		knownHostsBastion = append(knownHostsBastion, fmt.Sprintf("%s %s", bastion.host(), bastion.hostKey()))
	} else {
		if !ansibleSSHSettings.InsecureNoStrictHostKeyChecking() {
			v.o.Output(fmt.Sprintf("InsecureNoStrictHostKeyChecking false" ))
			if compute_resource {
				if ansibleSSHSettings.UserKnownHostsFile() == "" {
					if target.hostKey() == "" {
						v.o.Output(fmt.Sprintf("host key for '%s' not passed", target.host()))
						// fetchHostKey will issue an ssh Dial and update the hostKey() value
						// as with bastionKeyScan, we might ask for the host key while the instance
						// is not ready to respond to SSH, we need to retry for a number of times
						timeoutMs := ansibleSSHSettings.SSHKeyscanSeconds() * 1000
						timeSpentMs := 0
						intervalMs := 5000


						for {
							if err := target.fetchHostKey(); err != nil {
								v.o.Output(fmt.Sprintf("host key for '%s' not received yet; retrying...", target.host()))
								time.Sleep(time.Duration(intervalMs) * time.Millisecond)
								timeSpentMs = timeSpentMs + intervalMs
								if timeSpentMs > timeoutMs {
									v.o.Output(fmt.Sprintf("host key for '%s' not received within %d seconds",
										target.host(),
										ansibleSSHSettings.SSHKeyscanSeconds()))
									return err
								}
							} else {
								break
							}
						}
						if target.hostKey() == "" {
							return fmt.Errorf("expected to receive the host key for '%s', but no host key arrived", target.host())
						}
					}
					knownHostsTarget = append(knownHostsTarget, fmt.Sprintf("%s %s", target.host(), target.hostKey()))
				} else {
					v.o.Output(fmt.Sprintf("using '%s' as a known hosts file", ansibleSSHSettings.UserKnownHostsFile()))
				}
			} else {
				v.o.Output("null_resource, not verifying host keys")
				// StrictHostKeyChecking=no set during play execution
			}
		} else {
			v.o.Output("StrictHostKeyChecking=no specified or set for null_resource, not verifying host keys")
		}
	}

	knownHostsFileBastion, err := v.writeKnownHosts(knownHostsBastion)
	if err != nil {
		return err
	}
	defer os.Remove(knownHostsFileBastion)

	knownHostsFileTarget, err := v.writeKnownHosts(knownHostsTarget)
	if err != nil {
		return err
	}
	defer os.Remove(knownHostsFileTarget)

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

		// we can't pass bastion instance into this function
		// we would end up with a circular import
		command, err := play.ToLocalCommand(types.LocalModeAnsibleArgs{
			Username:              v.connInfo.User,
			Port:                  v.connInfo.Port,
			PemFile:               targetPemFile,
			KnownHostsFile:        knownHostsFileTarget,
			BastionKnownHostsFile: knownHostsFileBastion,
			BastionHost:           bastion.host(),
			BastionPemFile:        bastionPemFile,
			BastionPort:           bastion.port(),
			BastionUsername:       bastion.user(),
		}, ansibleSSHSettings)

		if err != nil {
			return err
		}

		v.o.Output(fmt.Sprintf("running local command: %s", command))

		if err := v.runCommand(command); err != nil {
			return err
		}

	}

	return nil
}

func (v *LocalMode) writeKnownHosts(knownHosts []string) (string, error) {
	trimmedKnownHosts := make([]string, 0)
	for _, entry := range knownHosts {
		trimmedKnownHosts = append(trimmedKnownHosts, strings.TrimSpace(entry))
	}
	knownHostsFileContents := strings.Join(trimmedKnownHosts, "\n")
	file, err := ioutil.TempFile(os.TempDir(), uuid.Must(uuid.NewV4()).String())
	defer file.Close()
	if err != nil {
		return "", err
	}
	v.o.Output(fmt.Sprintf("Write known hosts %s\n", knownHostsFileContents) )
	if err := ioutil.WriteFile(file.Name(), []byte(fmt.Sprintf("%s\n", knownHostsFileContents)), 0644); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func (v *LocalMode) writePem(pk string) (string, error) {
	if v.connInfo.PrivateKey != "" {
		file, err := ioutil.TempFile(os.TempDir(), uuid.Must(uuid.NewV4()).String())
		defer file.Close()
		if err != nil {
			return "", err
		}

		v.o.Output(fmt.Sprintf("Writing temprary PEM to '%s'...", file.Name()))
		if err := ioutil.WriteFile(file.Name(), []byte(pk), 0400); err != nil {
			return "", err
		}

		v.o.Output("Ansible inventory written.")
		return file.Name(), nil
	}
	return "", nil
}

func (v *LocalMode) writeInventory(play *types.Play) (string, error) {
	if play.InventoryFile() == "" {


		playHosts := play.Hosts()

		templateData := inventoryTemplateLocalData{
			Hosts:  make([]inventoryTemplateLocalDataHost, 0),
			Groups: play.Groups(),
		}

        // Compute resource path
		if v.connInfo.Host != "" {
			if len(playHosts) > 0 {
				if playHosts[0] != "" {
					templateData.Hosts = append(templateData.Hosts, inventoryTemplateLocalDataHost{
						Alias:       playHosts[0],
						AnsibleHost: v.connInfo.Host,
					})
				} else {
					templateData.Hosts = append(templateData.Hosts, inventoryTemplateLocalDataHost{
						Alias: v.connInfo.Host,
					})
				}
			} else {
				templateData.Hosts = append(templateData.Hosts, inventoryTemplateLocalDataHost{
					Alias: v.connInfo.Host,
				})
			}	
		} else {
			// Path for null resource, which does not use v.connInfo.Host
			for _, host := range playHosts {
				if host != "" {
					templateData.Hosts = append(templateData.Hosts, inventoryTemplateLocalDataHost{
							Alias: host,
						})
				}
			}

		}

		v.o.Output("Generating temporary ansible inventory...")
		t := template.Must(template.New("hosts").Parse(inventoryTemplateLocal))
		var buf bytes.Buffer
		err := t.Execute(&buf, templateData)
		if err != nil {
			return "", fmt.Errorf("Error executing 'hosts' template: %s", err)
		}

		file, err := ioutil.TempFile(os.TempDir(), "temporary-ansible-inventory")
		defer file.Close()
		if err != nil {
			return "", err
		}

		v.o.Output(fmt.Sprintf("Writing temporary ansible inventory to '%s'...", file.Name()))
		if err := ioutil.WriteFile(file.Name(), buf.Bytes(), 0644); err != nil {
			return "", err
		}

		v.o.Output("Ansible inventory written.")

		return file.Name(), nil
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
