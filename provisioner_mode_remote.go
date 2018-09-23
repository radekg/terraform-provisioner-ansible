package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
	uuid "github.com/satori/go.uuid"
)

func (p *provisioner) remoteDeployAnsibleData(o terraform.UIOutput, comm communicator.Communicator) ([]runnablePlay, error) {

	response := make([]runnablePlay, 0)

	for _, playDef := range p.Plays {
		if !playDef.Enabled {
			continue
		}

		switch playCallable := playDef.Callable.(type) {
		case ansiblePlaybook:
			playbookPath, err := resolvePath(playCallable.FilePath)
			if err != nil {
				return response, err
			}

			// playbook file is at the top level of the module
			// parse the playbook path's directory and upload the entire directory
			playbookDir := filepath.Dir(playbookPath)
			playbookDirHash := getMD5Hash(playbookDir)

			remotePlaybookDir := filepath.Join(bootstrapDirectory, playbookDirHash)
			remotePlaybookPath := filepath.Join(remotePlaybookDir, filepath.Base(playbookPath))

			if err := p.remoteRunCommandNoSudo(o, comm, fmt.Sprintf("mkdir -p \"%s\"", bootstrapDirectory)); err != nil {
				return response, err
			}

			errCmdCheck := p.remoteRunCommandNoSudo(o, comm, fmt.Sprintf("/bin/bash -c 'if [ -d \"%s\" ]; then exit 50; fi'", remotePlaybookDir))
			if err != nil {
				errCmdCheckDetail := strings.Split(fmt.Sprintf("%v", errCmdCheck), ": ")
				if errCmdCheckDetail[len(errCmdCheckDetail)-1] == "50" {
					o.Output(fmt.Sprintf("The playbook '%s' directory '%s' has been already uploaded.", playDef.Callable, playbookDir))
				} else {
					return response, err
				}
			} else {
				o.Output(fmt.Sprintf("Uploading the parent directory '%s' of playbook '%s' to '%s'...", playbookDir, playDef.Callable, remotePlaybookDir))
				// upload ansible source and playbook to the host
				if err := comm.UploadDir(remotePlaybookDir, playbookDir); err != nil {
					return response, err
				}
			}

			playDef.Callable = remotePlaybookPath

			// always upload vault password file:
			uploadedVaultPasswordFilePath, err := p.remoteUploadVaultPasswordFile(o, comm, remotePlaybookDir, playDef.CallArgs.Shared)
			if err != nil {
				return response, err
			}

			// always create temp inventory:
			inventoryFile, err := p.remoteWriteInventory(o, comm, remotePlaybookDir, playDef.CallArgs, playDef.InventoryMeta)
			if err != nil {
				return response, err
			}

			response = append(response, runnablePlay{
				Play:                   playDef,
				VaultPasswordFile:      uploadedVaultPasswordFilePath,
				InventoryFile:          inventoryFile,
				InventoryFileTemporary: len(playDef.CallArgs.Shared.InventoryFile) == 0,
			})
		case ansibleModule:
			if err := p.remoteRunCommandNoSudo(o, comm, fmt.Sprintf("mkdir -p \"%s\"", bootstrapDirectory)); err != nil {
				return response, err
			}

			// always upload vault password file:
			uploadedVaultPasswordFilePath, err := p.remoteUploadVaultPasswordFile(o, comm, bootstrapDirectory, playDef.CallArgs.Shared)
			if err != nil {
				return response, err
			}

			// always create temp inventory:
			inventoryFile, err := p.remoteWriteInventory(o, comm, bootstrapDirectory, playDef.CallArgs, playDef.InventoryMeta)
			if err != nil {
				return response, err
			}

			response = append(response, runnablePlay{
				Play:                   playDef,
				VaultPasswordFile:      uploadedVaultPasswordFilePath,
				InventoryFile:          inventoryFile,
				InventoryFileTemporary: len(playDef.CallArgs.Shared.InventoryFile) == 0,
			})
		}
	}

	return response, nil
}

func (p *provisioner) remoteInstallAnsible(o terraform.UIOutput, comm communicator.Communicator) error {

	installer := &ansibleInstaller{
		AnsibleVersion: "ansible",
	}
	if len(p.installVersion) > 0 {
		installer.AnsibleVersion = fmt.Sprintf("%s==%s", installer.AnsibleVersion, p.installVersion)
	}

	o.Output(fmt.Sprintf("Installing '%s'...", installer.AnsibleVersion))

	t := template.Must(template.New("installer").Parse(installerProgramTemplate))
	var buf bytes.Buffer
	err := t.Execute(&buf, installer)
	if err != nil {
		return fmt.Errorf("Error executing 'installer' template: %s", err)
	}
	targetPath := "/tmp/ansible-install.sh"

	o.Output(fmt.Sprintf("Uploading ansible installer program to %s...", targetPath))
	if err := comm.UploadScript(targetPath, bytes.NewReader(buf.Bytes())); err != nil {
		return err
	}

	if err := p.remoteRunCommandSudo(o, comm, fmt.Sprintf("/bin/bash -c '\"%s\" && rm \"%s\"'", targetPath, targetPath)); err != nil {
		return err
	}

	o.Output("Ansible installed.")
	return nil
}

func (p *provisioner) remoteUploadVaultPasswordFile(o terraform.UIOutput, comm communicator.Communicator, destination string, callArgs ansibleCallArgsShared) (string, error) {

	if callArgs.VaultPasswordFile == "" {
		return "", nil
	}

	source, err := resolvePath(callArgs.VaultPasswordFile)
	if err != nil {
		return "", err
	}

	u1 := uuid.Must(uuid.NewV4())
	targetPath := filepath.Join(destination, fmt.Sprintf(".vault-password-file-%s", u1))

	o.Output(fmt.Sprintf("Uploading ansible vault password file to '%s'...", targetPath))

	file, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if err := comm.Upload(targetPath, bufio.NewReader(file)); err != nil {
		return "", err
	}

	o.Output("Ansible vault password file uploaded.")

	return targetPath, nil
}

func (p *provisioner) remoteWriteInventory(o terraform.UIOutput, comm communicator.Communicator, destination string, callArgs ansibleCallArgs, inventoryMeta ansibleInventoryMeta) (string, error) {

	if len(callArgs.Shared.InventoryFile) > 0 {

		o.Output(fmt.Sprintf("Using provided inventory file '%s'...", callArgs.Shared.InventoryFile))
		source, err := resolvePath(callArgs.Shared.InventoryFile)
		if err != nil {
			return "", err
		}
		u1 := uuid.Must(uuid.NewV4())
		targetPath := filepath.Join(destination, fmt.Sprintf(".inventory-%s", u1))
		o.Output(fmt.Sprintf("Uploading provided inventory file '%s' to '%s'...", callArgs.Shared.InventoryFile, targetPath))

		file, err := os.Open(source)
		if err != nil {
			return "", err
		}
		defer file.Close()

		if err := comm.Upload(targetPath, bufio.NewReader(file)); err != nil {
			return "", err
		}

		o.Output("Ansible inventory uploaded.")

		return targetPath, nil

	}

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
}

func (p *provisioner) remoteCleanupAfterBootstrap(o terraform.UIOutput, comm communicator.Communicator) {
	o.Output("Cleaning up after bootstrap...")
	p.remoteRunCommandNoSudo(o, comm, fmt.Sprintf("rm -rf \"%s\"", bootstrapDirectory))
	o.Output("Cleanup complete.")
}

func (p *provisioner) remoteRunCommandSudo(o terraform.UIOutput, comm communicator.Communicator, command string) error {
	return p.remoteRunCommand(o, comm, command, true)
}

func (p *provisioner) remoteRunCommandNoSudo(o terraform.UIOutput, comm communicator.Communicator, command string) error {
	return p.remoteRunCommand(o, comm, command, false)
}

func (p *provisioner) remoteRunCommand(o terraform.UIOutput, comm communicator.Communicator, command string, shouldSudo bool) error {
	// Unless prevented, prefix the command with sudo
	if shouldSudo && p.useSudo {
		command = fmt.Sprintf("sudo %s", command)
	}

	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	outDoneCh := make(chan struct{})
	errDoneCh := make(chan struct{})
	go p.copyOutput(o, outR, outDoneCh)
	go p.copyOutput(o, errR, errDoneCh)

	cmd := &remote.Cmd{
		Command: command,
		Stdout:  outW,
		Stderr:  errW,
	}

	err := comm.Start(cmd)
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
