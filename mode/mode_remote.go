package mode

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	linereader "github.com/mitchellh/go-linereader"
	"github.com/radekg/terraform-provisioner-ansible/types"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
	uuid "github.com/satori/go.uuid"
)

const installerProgramTemplate = `#!/usr/bin/env bash
if [ -z "$(which ansible-playbook)" ]; then
  
  # only check the cloud boot finished if the directory exists
  if [ -d /var/lib/cloud/instance ]; then
    until [[ -f /var/lib/cloud/instance/boot-finished ]]; do
      sleep 1
    done
  fi

  # install dependencies
  if [[ -f /etc/redhat-release ]]; then
    yum update -y \
    && yum groupinstall -y "Development Tools" \
    && yum install -y python-devel
  else
    apt-get update \
    && apt-get install -y build-essential python-dev
  fi

  # install pip, if necessary
  if [ -z "$(which pip)" ]; then
    curl https://bootstrap.pypa.io/get-pip.py | sudo python
  fi

  # install ansible
  pip install {{ .AnsibleVersion}}

else

  expected_version="{{ .AnsibleVersion}}"
  installed_version=$(ansible-playbook --version | head -n1 | awk '{print $2}')
  installed_version="ansible==$installed_version"
  if [[ "$expected_version" = *"=="* ]]; then
    if [ "$expected_version" != "$installed_version" ]; then
      pip install $expected_version
    fi
  fi
  
fi
`

// RemoteMode represents remote provisioner mode.
type RemoteMode struct {
	o              terraform.UIOutput
	comm           communicator.Communicator
	remoteSettings *types.RemoteSettings
}

type ansibleInstaller struct {
	AnsibleVersion string
}

const (
	bootstrapDirectory string = "/tmp/ansible-terraform-bootstrap"
)

// NewRemoteMode returns configured remote mode provisioner.
func NewRemoteMode(o terraform.UIOutput, s *terraform.InstanceState, remoteSettings *types.RemoteSettings) (*RemoteMode, error) {
	// Get a new communicator
	comm, err := communicator.New(s)
	if err != nil {
		return nil, err
	}
	return &RemoteMode{
		o:              o,
		comm:           comm,
		remoteSettings: remoteSettings,
	}, nil
}

// Run executes remote provisioning process.
func (v *RemoteMode) Run(plays []*types.Play) error {
	// Wait and retry until we establish the connection
	err := v.retryFunc(v.comm.Timeout(), func() error {
		return v.comm.Connect(v.o)
	})
	if err != nil {
		return err
	}
	defer v.comm.Disconnect()

	if !v.remoteSettings.SkipInstall() {
		if err := v.installAnsible(v.remoteSettings); err != nil {
			return err
		}
	}

	err = v.deployAnsibleData(plays)

	if err != nil {
		v.o.Output(fmt.Sprintf("%+v", err))
		return err
	}

	for _, play := range plays {
		command, err := play.ToCommand()
		if err != nil {
			return err
		}
		v.o.Output(fmt.Sprintf("running command: %s", command))
		if err := v.runCommandSudo(command); err != nil {
			return err
		}
	}

	if !v.remoteSettings.SkipCleanup() {
		v.cleanupAfterBootstrap()
	}

	return nil

}

// retryFunc is used to retry a function for a given duration
func (v *RemoteMode) retryFunc(timeout time.Duration, f func() error) error {
	finish := time.After(timeout)
	for {
		err := f()
		if err == nil {
			return nil
		}
		log.Printf("Retryable error: %v", err)

		select {
		case <-finish:
			return err
		case <-time.After(3 * time.Second):
		}
	}
}

func (v *RemoteMode) getMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (v *RemoteMode) deployAnsibleData(plays []*types.Play) error {

	for _, play := range plays {
		if !play.Enabled() {
			continue
		}

		switch entity := play.Entity().(type) {
		case types.Playbook:
			playbookPath, err := types.ResolvePath(entity.FilePath())
			if err != nil {
				return err
			}

			// playbook file is at the top level of the module
			// parse the playbook path's directory and upload the entire directory
			playbookDir := filepath.Dir(playbookPath)
			playbookDirHash := v.getMD5Hash(playbookDir)

			remotePlaybookDir := filepath.Join(bootstrapDirectory, playbookDirHash)
			remotePlaybookPath := filepath.Join(remotePlaybookDir, filepath.Base(playbookPath))

			if err := v.runCommandNoSudo(fmt.Sprintf("mkdir -p \"%s\"", bootstrapDirectory)); err != nil {
				return err
			}

			errCmdCheck := v.runCommandNoSudo(fmt.Sprintf("/bin/bash -c 'if [ -d \"%s\" ]; then exit 50; fi'", remotePlaybookDir))
			if err != nil {
				errCmdCheckDetail := strings.Split(fmt.Sprintf("%v", errCmdCheck), ": ")
				if errCmdCheckDetail[len(errCmdCheckDetail)-1] == "50" {
					v.o.Output(fmt.Sprintf("The playbook '%s' directory '%s' has been already uploaded.", entity.FilePath(), playbookDir))
				} else {
					return err
				}
			} else {
				v.o.Output(fmt.Sprintf("Uploading the parent directory '%s' of playbook '%s' to '%s'...", playbookDir, entity.FilePath(), remotePlaybookDir))
				// upload ansible source and playbook to the host
				if err := v.comm.UploadDir(remotePlaybookDir, playbookDir); err != nil {
					return err
				}
			}

			entity.SetRunnableFilePath(remotePlaybookPath)

			// always upload vault password file:
			uploadedVaultPasswordFilePath, err := v.uploadVaultPasswordFile(remotePlaybookDir, play)
			if err != nil {
				return err
			}
			play.SetRemoteVaultPasswordPath(uploadedVaultPasswordFilePath)

			// always create temp inventory:
			inventoryFile, err := v.writeInventory(remotePlaybookDir, play)
			if err != nil {
				return err
			}
			play.SetOverrideInventoryFile(inventoryFile)

		case types.Module:
			if err := v.runCommandNoSudo(fmt.Sprintf("mkdir -p \"%s\"", bootstrapDirectory)); err != nil {
				return err
			}

			// always upload vault password file:
			uploadedVaultPasswordFilePath, err := v.uploadVaultPasswordFile(bootstrapDirectory, play)
			if err != nil {
				return err
			}
			play.SetRemoteVaultPasswordPath(uploadedVaultPasswordFilePath)

			// always create temp inventory:
			inventoryFile, err := v.writeInventory(bootstrapDirectory, play)
			if err != nil {
				return err
			}
			play.SetOverrideInventoryFile(inventoryFile)

		}
	}

	return nil
}

func (v *RemoteMode) installAnsible(remoteSettings *types.RemoteSettings) error {

	installer := &ansibleInstaller{
		AnsibleVersion: "ansible",
	}
	if remoteSettings.InstallVersion() != "" {
		installer.AnsibleVersion = fmt.Sprintf("%s==%s", installer.AnsibleVersion, remoteSettings.InstallVersion())
	}

	v.o.Output(fmt.Sprintf("Installing '%s'...", installer.AnsibleVersion))

	t := template.Must(template.New("installer").Parse(installerProgramTemplate))
	var buf bytes.Buffer
	err := t.Execute(&buf, installer)
	if err != nil {
		return fmt.Errorf("Error executing 'installer' template: %s", err)
	}
	targetPath := "/tmp/ansible-install.sh"

	v.o.Output(fmt.Sprintf("Uploading ansible installer program to %s...", targetPath))
	if err := v.comm.UploadScript(targetPath, bytes.NewReader(buf.Bytes())); err != nil {
		return err
	}

	if err := v.runCommandSudo(fmt.Sprintf("/bin/bash -c '\"%s\" && rm \"%s\"'", targetPath, targetPath)); err != nil {
		return err
	}

	v.o.Output("Ansible installed.")
	return nil
}

func (v *RemoteMode) uploadVaultPasswordFile(destination string, play *types.Play) (string, error) {

	if play.VaultPasswordFile() == "" {
		return "", nil
	}

	source, err := types.ResolvePath(play.VaultPasswordFile())
	if err != nil {
		return "", err
	}

	u1 := uuid.Must(uuid.NewV4())
	targetPath := filepath.Join(destination, fmt.Sprintf(".vault-password-file-%s", u1))

	v.o.Output(fmt.Sprintf("Uploading ansible vault password file to '%s'...", targetPath))

	file, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if err := v.comm.Upload(targetPath, bufio.NewReader(file)); err != nil {
		return "", err
	}

	v.o.Output("Ansible vault password file uploaded.")

	return targetPath, nil
}

func (v *RemoteMode) writeInventory(destination string, play *types.Play) (string, error) {

	if play.InventoryFile() != "" {

		v.o.Output(fmt.Sprintf("Using provided inventory file '%s'...", play.InventoryFile()))
		source, err := types.ResolvePath(play.InventoryFile())
		if err != nil {
			return "", err
		}
		u1 := uuid.Must(uuid.NewV4())
		targetPath := filepath.Join(destination, fmt.Sprintf(".inventory-%s", u1))
		v.o.Output(fmt.Sprintf("Uploading provided inventory file '%s' to '%s'...", play.InventoryFile(), targetPath))

		file, err := os.Open(source)
		if err != nil {
			return "", err
		}
		defer file.Close()

		if err := v.comm.Upload(targetPath, bufio.NewReader(file)); err != nil {
			return "", err
		}

		v.o.Output("Ansible inventory uploaded.")

		return targetPath, nil

	}

	/*
		$$ TODO: resotre this without a template:
		o.Output("Generating temporary ansible inventory...")
		t := template.Must(template.New("hosts").Parse(inventoryTemplateRemote))
		var buf bytes.Buffer
		err := t.Execute(&buf, inventoryMeta)
		if err != nil {
			return "", fmt.Errorf("Error executing 'hosts' template: %s", err)
		}

		u1 := uuid.Must(uuid.NewV4())
		targetPath := filepath.Join(destination, fmt.Sprintf(".inventory-%s", u1))

		o.Output(fmt.Sprintf("Writing temporary ansible inventory to '%s'...", targetPath))
		if err := comm.Upload(targetPath, bytes.NewReader(buf.Bytes())); err != nil {
			return "", err
		}

		o.Output("Ansible inventory written.")
		return targetPath, nil
	*/
	return "", nil
}

func (v *RemoteMode) cleanupAfterBootstrap() {
	v.o.Output("Cleaning up after bootstrap...")
	v.runCommandNoSudo(fmt.Sprintf("rm -rf \"%s\"", bootstrapDirectory))
	v.o.Output("Cleanup complete.")
}

func (v *RemoteMode) runCommandSudo(command string) error {
	return v.runCommand(command, true)
}

func (v *RemoteMode) runCommandNoSudo(command string) error {
	return v.runCommand(command, false)
}

func (v *RemoteMode) runCommand(command string, shouldSudo bool) error {
	// Unless prevented, prefix the command with sudo
	if shouldSudo && v.remoteSettings.UseSudo() {
		command = fmt.Sprintf("sudo %s", command)
	}

	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	outDoneCh := make(chan struct{})
	errDoneCh := make(chan struct{})
	go v.copyOutput(outR, outDoneCh)
	go v.copyOutput(errR, errDoneCh)

	cmd := &remote.Cmd{
		Command: command,
		Stdout:  outW,
		Stderr:  errW,
	}

	err := v.comm.Start(cmd)
	if err != nil {
		return fmt.Errorf("Error executing command %q: %v", cmd.Command, err)
	}

	err = cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*remote.ExitError); ok {
			err = fmt.Errorf(
				"Command '%q' exited with non-zero exit status: %d, reason %+v", cmd.Command, exitErr.ExitStatus, exitErr.Err)
		} else {
			err = fmt.Errorf(
				"Command '%q' failed, reason: %+v", cmd.Command, err)
		}
	}

	// Wait for output to clean up
	outW.Close()
	errW.Close()
	<-outDoneCh
	<-errDoneCh

	return err
}

func (v *RemoteMode) copyOutput(r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)
	lr := linereader.New(r)
	for line := range lr.Ch {
		v.o.Output(line)
	}
}