package main

import (
  "bufio"
  "bytes"
  "context"
  "crypto/md5"
  "errors"
  "encoding/hex"
  "encoding/json"
  "fmt"
  "io"
  "log"
  "os"
  "path/filepath"
  "strings"
  "text/template"
  "time"

  "github.com/hashicorp/terraform/communicator"
  "github.com/hashicorp/terraform/communicator/remote"
  "github.com/hashicorp/terraform/helper/schema"
  "github.com/hashicorp/terraform/terraform"

  "github.com/mitchellh/go-homedir"
  "github.com/mitchellh/go-linereader"

  "github.com/satori/go.uuid"
)

const (
  bootstrapDirectory string = "/tmp/ansible-terraform-bootstrap"
  defaultStartAtTask string = ""
  defaultLimit string = ""
  defaultForks int = 5
  defaultVerbose string = ""
  defaultVerbose_Set string = "no"
  defaultForceHandlers string = ""
  defaultForceHandlers_Set string = "no"
  defaultOneLine string = ""
  defaultOneLine_Set string = "no"
  defaultBecome string = ""
  defaultBecome_Set string = "no"
  defaultBecomeMethod string = ""
  defaultBecomeMethod_Set string = "sudo"
  defaultBecomeUser string = ""
  defaultBecomeUser_Set string = "root"
  defaultVaultPasswordFile string = ""

  defaultBackground int = 0
  defaultHostPattern string = "all"
  defaultPoll int = 15
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

const inventoryTemplate = `{{$top := . -}}
{{range .Hosts -}}
{{.}} ansible_connection=local
{{end}}

{{range .Groups -}}
[{{.}}]
{{range $top.Hosts -}}
{{.}} ansible_connection=local
{{end}}

{{end}}`

var yesNoStates = map[string]bool{"yes": true, "no": true}
var becomeMethods = map[string]bool{"sudo": true, "su": true, "pbrun": true, "pfexec": true, "doas": true, "dzdo": true, "ksu": true, "runas": true}

type ansibleInstaller struct {
  AnsibleVersion string
}

type AnsibleCallbaleType int
const (
  AnsibleCallable_Undefined AnsibleCallbaleType = iota
  AnsibleCallable_Conflicting
  AnsibleCallable_Playbook
  AnsibleCallable_Module
)

type ansibleInventoryMeta struct {
  Hosts  []string
  Groups []string
}

type ansibleCallArgsShared struct {
  Become            string
  BecomeMethod      string
  BecomeUser        string
  ExtraVars         map[string]interface{}
  Forks             int
  Limit             string
  VaultPasswordFile string
  Verbose           string
}

type ansibleCallArgs struct {
  // module only:
  Args          map[string]interface{}
  Background    int
  HostPattern   string
  OneLine       string
  Poll          int
  // Playbook only:
  ForceHandlers string
  SkipTags      []string
  StartAtTask   string
  Tags          []string
  // shared:
  Shared        ansibleCallArgsShared
}

// -- play:

type play struct {
  InventoryMeta ansibleInventoryMeta
  Callable      string
  CallableType  AnsibleCallbaleType
  CallArgs      ansibleCallArgs
}

func (p *play) ToCommand(inventoryFile string, vaultPasswordFile string) (string, error) {

  command := ""
  // entity to call:
  if p.CallableType == AnsibleCallable_Playbook {

    command = fmt.Sprintf("ansible-playbook %s", p.Callable)

    // force handlers:
    if p.CallArgs.ForceHandlers == "yes" {
      command = fmt.Sprintf("%s --force-handlers", command)
    }
    // skip tags:
    if len(p.CallArgs.SkipTags) > 0 {
      command = fmt.Sprintf("%s --skip-tags='%s'", command, strings.Join(p.CallArgs.SkipTags, ","))
    }
    // start at task:
    if len(p.CallArgs.StartAtTask) > 0 {
      command = fmt.Sprintf("%s --start-at-task='%s'", command, p.CallArgs.StartAtTask)
    }
    // tags:
    if len(p.CallArgs.Tags) > 0 {
      command = fmt.Sprintf("%s --tags='%s'", command, strings.Join(p.CallArgs.Tags, ","))
    }

  } else if p.CallableType == AnsibleCallable_Module {

    hostPattern := p.CallArgs.HostPattern
    if hostPattern == "" {
      hostPattern = defaultHostPattern
    }
    command = fmt.Sprintf("ansible %s --module-name='%s'", hostPattern, p.Callable)
    
    if p.CallArgs.Background > 0 {
      command = fmt.Sprintf("%s --background=%s", command, p.CallArgs.Background)
      if p.CallArgs.Poll > 0 {
        command = fmt.Sprintf("%s --poll=%s", command, p.CallArgs.Poll)
      }
    }
    // module args:
    if len(p.CallArgs.Args) > 0 {
      args := make([]string, 0)
      for mak, mav := range p.CallArgs.Args {
        args = append(args, fmt.Sprintf("%s=%+v", mak, mav))
      }
      command = fmt.Sprintf("%s --args=\"%s\"", command, strings.Join(args, " "))
    }
    // one line:
    if p.CallArgs.OneLine == "yes" {
      command = fmt.Sprintf("%s --one-line", command)
    }

  }
  // inventory file:
  command = fmt.Sprintf("%s --inventory-file='%s'", command, inventoryFile)

  // shared arguments:

  // become:
  if p.CallArgs.Shared.Become == "yes" {
    command = fmt.Sprintf("%s --become", command)
    if p.CallArgs.Shared.BecomeMethod != "" {
      command = fmt.Sprintf("%s --become-method='%s'", command, p.CallArgs.Shared.BecomeMethod)
    } else {
      command = fmt.Sprintf("%s --become-method='%s'", command, defaultBecomeMethod_Set)
    }
    if p.CallArgs.Shared.BecomeUser != "" {
      command = fmt.Sprintf("%s --become-user='%s'", command, p.CallArgs.Shared.BecomeUser)
    } else {
      command = fmt.Sprintf("%s --become-user='%s'", command, defaultBecomeUser_Set)
    }
  }
  // extra vars:
  if len(p.CallArgs.Shared.ExtraVars) > 0 {
    extraVars, err := json.Marshal(p.CallArgs.Shared.ExtraVars)
    if err != nil {
      return "", err
    }
    command = fmt.Sprintf("%s --extra-vars='%s'", command, string(extraVars))
  }
  // forks:
  if p.CallArgs.Shared.Forks > 0 {
    command = fmt.Sprintf("%s --forks=%d", command, p.CallArgs.Shared.Forks)
  }
  // limit
  if len(p.CallArgs.Shared.Limit) > 0 {
    command = fmt.Sprintf("%s --limit='%s'", command, p.CallArgs.Shared.Limit)
  }
  // vault password file:
  if len(vaultPasswordFile) > 0 {
    command = fmt.Sprintf("%s --vault-password-file='%s'", command, vaultPasswordFile)
  }
  // verbose:
  if p.CallArgs.Shared.Verbose == "yes" {
    command = fmt.Sprintf("%s --verbose", command)
  }

  return command, nil
}

// -- runnable play:

type runnablePlay struct {
  Play              play
  VaultPasswordFile string
  InventoryFile     string
}

func (r *runnablePlay) ToCommand() (string, error) {
  return r.Play.ToCommand(r.InventoryFile, r.VaultPasswordFile)
}

// -- provisioner:

type provisioner struct {
  Plays          []play
  InventoryMeta  ansibleInventoryMeta
  Shared         ansibleCallArgsShared
  useSudo        bool
  skipInstall    bool
  skipCleanup    bool
  installVersion string
}

func Provisioner() terraform.ResourceProvisioner {
  return &schema.Provisioner{
    Schema: map[string]*schema.Schema{

      "plays": &schema.Schema{
        Type:     schema.TypeList,
        Optional: true,
        Computed: true,
        Elem: &schema.Resource{
          Schema: map[string]*schema.Schema{
            // entity to run:
            "playbook": &schema.Schema{
              ConflictsWith: []string{"plays.module"},
              Type:     schema.TypeString,
              Optional: true,
            },
            "module": &schema.Schema{
              ConflictsWith: []string{"plays.playbook"},
              Type:     schema.TypeString,
              Optional: true,
            },
            // meta for temporary inventory:
            "hosts": &schema.Schema{
              Type:     schema.TypeList,
              Elem:     &schema.Schema{ Type: schema.TypeString },
              Optional: true,
            },
            "groups": &schema.Schema{
              Type:     schema.TypeList,
              Elem:     &schema.Schema{ Type: schema.TypeString },
              Optional: true,
            },
            // module only:
            "args": &schema.Schema{
              Type:     schema.TypeMap,
              Optional: true,
              Computed: true,
            },
            "background": &schema.Schema{
              Type:     schema.TypeInt,
              Optional: true,
              Default: defaultBackground,
            },
            "host_pattern": &schema.Schema{
              Type:     schema.TypeString,
              Optional: true,
              Default: defaultHostPattern,
            },
            "one_line": &schema.Schema{
              Type:     schema.TypeString,
              Optional: true,
              Default:  defaultOneLine,
            },
            "poll": &schema.Schema{
              Type:     schema.TypeInt,
              Optional: true,
              Default:  defaultPoll,
            },
            // playbook only:
            "force_handlers": &schema.Schema{
              Type:     schema.TypeString,
              Optional: true,
              Default:  defaultForceHandlers,
            },
            "skip_tags": &schema.Schema{
              Type:     schema.TypeList,
              Elem:     &schema.Schema{ Type: schema.TypeString },
              Optional: true,
            },
            "start_at_task": &schema.Schema{
              Type:     schema.TypeString,
              Optional: true,
              Default: defaultStartAtTask,
            },
            "tags": &schema.Schema{
              Type:     schema.TypeList,
              Elem:     &schema.Schema{ Type: schema.TypeString },
              Optional: true,
            },
            // shared:
            "become": &schema.Schema{
              Type:     schema.TypeString,
              Optional: true,
              Default:  defaultBecome,
            },
            "become_method": &schema.Schema{
              Type:     schema.TypeString,
              Optional: true,
              Default:  defaultBecomeMethod,
            },
            "become_user": &schema.Schema{
              Type:     schema.TypeString,
              Optional: true,
              Default:  defaultBecomeUser,
            },
            "extra_vars": &schema.Schema{
              Type:     schema.TypeMap,
              Optional: true,
              Computed: true,
            },
            "forks": &schema.Schema{
              Type:     schema.TypeInt,
              Optional: true,
              Default: defaultForks,
            },
            "limit": &schema.Schema{
              Type:     schema.TypeString,
              Optional: true,
              Default: defaultLimit,
            },
            "vault_password_file": &schema.Schema{
              Type:     schema.TypeString,
              Optional: true,
              Default:  defaultVaultPasswordFile,
            },
            "verbose": &schema.Schema{
              Type:     schema.TypeString,
              Optional: true,
              Default:  defaultVerbose,
            },

          },
        },
      },

      // inventory meta:
      "hosts": &schema.Schema{
        Type:     schema.TypeList,
        Elem:     &schema.Schema{ Type: schema.TypeString },
        Optional: true,
      },
      "groups": &schema.Schema{
        Type:     schema.TypeList,
        Elem:     &schema.Schema{ Type: schema.TypeString },
        Optional: true,
      },

      // shared:
      "become": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default:  defaultBecome,
      },
      "become_method": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default:  defaultBecomeMethod,
      },
      "become_user": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default:  defaultBecomeUser,
      },
      "extra_vars": &schema.Schema{
        Type:     schema.TypeMap,
        Optional: true,
        Computed: true,
      },
      "forks": &schema.Schema{
        Type:     schema.TypeInt,
        Optional: true,
        Default: defaultForks,
      },
      "limit": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default: defaultLimit,
      },
      "vault_password_file": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default:  defaultVaultPasswordFile,
      },
      "verbose": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default:  defaultVerbose,
      },

      "use_sudo": &schema.Schema{
        Type:     schema.TypeBool,
        Optional: true,
        Default:  true,
      },
      "skip_install": &schema.Schema{
        Type:     schema.TypeBool,
        Optional: true,
        Default:  false,
      },
      "skip_cleanup": &schema.Schema{
        Type:     schema.TypeBool,
        Optional: true,
        Default:  false,
      },
      "install_version": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default:  "", // latest
      },
    },
    ApplyFunc:    applyFn,
    ValidateFunc: validateFn,
  }
}

func applyFn(ctx context.Context) error {
  
  o := ctx.Value(schema.ProvOutputKey).(terraform.UIOutput)
  s := ctx.Value(schema.ProvRawStateKey).(*terraform.InstanceState)
  d := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)

    // Decode the provisioner config
  p, err := decodeConfig(d)
  if err != nil {
    return err
  }

  // Get a new communicator
  comm, err := communicator.New(s)
  if err != nil {
    return err
  }
  // Wait and retry until we establish the connection
  err = retryFunc(comm.Timeout(), func() error {
    return comm.Connect(o)
  })
  if err != nil {
    return err
  }
  defer comm.Disconnect()

  if !p.skipInstall {
    if err := p.remote_installAnsible(o, comm); err != nil {
      return err
    }
  }

  runnablePlays := make([]runnablePlay, 0)

  if runnables, err := p.remote_deployAnsibleData(o, comm); err != nil {
    o.Output(fmt.Sprintf("%+v", err))
    return err
  } else {
    runnablePlays = runnables
  }

  for _, runnable := range runnablePlays {
    command, err := runnable.ToCommand()
    if err != nil {
      return err
    }
    o.Output(fmt.Sprintf("running command: %s", command))
    if err := p.runCommandSudo(o, comm, command); err != nil {
      return err
    }
  }

  if !p.skipCleanup {
    p.remote_cleanupAfterBootstrap(o, comm)
  }

  return nil

}

func validateFn(c *terraform.ResourceConfig) (ws []string, es []error) {
  becomeMethod, ok := c.Get("become_method")
  if ok {
    if !becomeMethods[becomeMethod.(string)] {
      es = append(es, errors.New(becomeMethod.(string)+" is not a valid become_method."))
    }
  }

  fields := []string{"verbose", "force_handlers", "one_line", "become"}
  for _, field := range fields {
    v, ok := c.Get(field)
    if ok && v.(string) != "" {
      if !yesNoStates[v.(string)] {
        es = append(es, errors.New(v.(string)+" is not a valid " + field + "."))
      }
    }
  }

  // Validate plays configs
  plays, ok := c.Get("plays")
  if ok {
    for _, p := range plays.([]map[string]interface{}) {

      playbook, okp := p["playbook"]
      module, okm := p["module"]

      if okp && okm && len(playbook.(string)) > 0 && len(module.(string)) > 0 {
        es = append(es, errors.New("playbook and module can't be used together"))
      } else {

        isPlaybook := okp && len(playbook.(string)) > 0
        isModule := okm && len(module.(string)) > 0

        if !okp && !okm {
          es = append(es, errors.New("playbook or module must be set"))
        }

        if isPlaybook {
          disallowedFields := []string{"args", "background", "host_pattern", "one_line", "poll"}
          for _, df := range disallowedFields {
            if _, ok = p[df]; ok {
              es = append(es, errors.New(fmt.Sprintf("%s must not be used with playbook", df)))
            }
          }
        }

        if isModule {
          disallowedFields := []string{"force_handlers", "skip_tags", "start_at_task", "tags"}
          for _, df := range disallowedFields {
            if _, ok = p[df]; ok {
              es = append(es, errors.New(fmt.Sprintf("%s must not be used with module", df)))
            }
          }
        }

      }

      becomeMethodPlay, ok := p["become_method"]
      if ok {
        if !becomeMethods[becomeMethodPlay.(string)] {
          es = append(es, errors.New(becomeMethodPlay.(string)+" is not a valid become_method."))
        }
      }

      for _, fieldPlay := range fields {
        v, ok := p[fieldPlay]
        if ok && v.(string) != "" {
          if !yesNoStates[v.(string)] {
            es = append(es, errors.New(v.(string)+" is not a valid " + fieldPlay + "."))
          }
        }
      }
    }
  } else {
    ws = append(ws, "Nothing to play.")
  }

  return ws, es
}

func (p *provisioner) remote_deployAnsibleData(o terraform.UIOutput, comm communicator.Communicator) ([]runnablePlay, error) {
  
  response := make([]runnablePlay, 0)
  
  for _, playDef := range p.Plays {
    if playDef.CallableType == AnsibleCallable_Playbook {

      playbookPath, err := p.resolvePath(playDef.Callable, o)
      if err != nil {
        return response, err
      }

      // playbook file is at the top level of the module
      // parse the playbook path's directory and upload the entire directory
      playbookDir := filepath.Dir(playbookPath)
      playbookDirHash := getMD5Hash(playbookDir)

      remotePlaybookDir := filepath.Join(bootstrapDirectory, playbookDirHash)
      remotePlaybookPath := filepath.Join(remotePlaybookDir, filepath.Base(playbookPath))

      if err := p.runCommandNoSudo(o, comm, fmt.Sprintf("mkdir -p %s", bootstrapDirectory)); err != nil {
        return response, err
      }

      errCmdCheck := p.runCommandNoSudo(o, comm, fmt.Sprintf("/bin/bash -c 'if [ -d \"%s\" ]; then exit 50; fi'", remotePlaybookDir))
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
      uploadedVaultPasswordFilePath, err := p.remote_uploadVaultPasswordFile(o, comm, remotePlaybookDir, playDef.CallArgs.Shared)
      if err != nil {
        return response, err
      }

      // always create temp inventory:
      inventoryFile, err := p.remote_writeTemporaryInventory(o, comm, remotePlaybookDir, playDef.CallArgs)
      if err != nil {
        return response, err
      }

      response = append(response, runnablePlay{
        Play: playDef,
        VaultPasswordFile: uploadedVaultPasswordFilePath,
        InventoryFile: inventoryFile,
      })

    } else if playDef.CallableType == AnsibleCallable_Module {

      if err := p.runCommandNoSudo(o, comm, fmt.Sprintf("mkdir -p %s", bootstrapDirectory)); err != nil {
        return response, err
      }

      // always upload vault password file:
      uploadedVaultPasswordFilePath, err := p.remote_uploadVaultPasswordFile(o, comm, bootstrapDirectory, playDef.CallArgs.Shared)
      if err != nil {
        return response, err
      }

      // always create temp inventory:
      inventoryFile, err := p.remote_writeTemporaryInventory(o, comm, bootstrapDirectory, playDef.CallArgs)
      if err != nil {
        return response, err
      }

      response = append(response, runnablePlay{
        Play: playDef,
        VaultPasswordFile: uploadedVaultPasswordFilePath,
        InventoryFile: inventoryFile,
      })
    }
  }

  return response, nil
}

func (p *provisioner) remote_installAnsible(o terraform.UIOutput, comm communicator.Communicator) error {

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

  if err := p.runCommandSudo(o, comm, fmt.Sprintf("/bin/bash -c '%s && rm %s'", targetPath, targetPath)); err != nil {
    return err
  }

  o.Output("Ansible installed.")
  return nil
}

func (p *provisioner) remote_uploadVaultPasswordFile(o terraform.UIOutput, comm communicator.Communicator, destination string, callArgs ansibleCallArgsShared) (string, error) {

  if callArgs.VaultPasswordFile == "" {
    return "", nil
  }

  source, err := p.resolvePath(callArgs.VaultPasswordFile, o)
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

func (p *provisioner) remote_writeTemporaryInventory(o terraform.UIOutput, comm communicator.Communicator, destination string, callArgs ansibleCallArgs) (string, error) {
  o.Output("Generating temporary ansible inventory...")
  t := template.Must(template.New("hosts").Parse(inventoryTemplate))
  var buf bytes.Buffer
  err := t.Execute(&buf, callArgs)
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

func (p *provisioner) remote_cleanupAfterBootstrap(o terraform.UIOutput, comm communicator.Communicator) {
  o.Output("Cleaning up after bootstrap...")
  p.runCommandNoSudo(o, comm, fmt.Sprintf("rm -r %s", bootstrapDirectory))
  o.Output("Cleanup complete.")
}

func (p *provisioner) resolvePath(path string, o terraform.UIOutput) (string, error) {
  expandedPath, _ := homedir.Expand(path)
  if _, err := os.Stat(expandedPath); err == nil {
    return expandedPath, nil
  }
  return "", fmt.Errorf("Ansible module not found at path: [%s]", path)
}

func (p *provisioner) runCommandSudo(o terraform.UIOutput, comm communicator.Communicator, command string) error {
  return p.runCommand(o, comm, command, true)
}

func (p *provisioner) runCommandNoSudo(o terraform.UIOutput, comm communicator.Communicator, command string) error {
  return p.runCommand(o, comm, command, false)
}

// runCommand is used to run already prepared commands
func (p *provisioner) runCommand(o terraform.UIOutput, comm communicator.Communicator, command string, shouldSudo bool) error {
  // Unless prevented, prefix the command with sudo
  if shouldSudo && p.useSudo {
    command = "sudo " + command
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

  cmd.Wait()
  if cmd.ExitStatus != 0 {
    err = fmt.Errorf(
      "Command %q exited with non-zero exit status: %d", cmd.Command, cmd.ExitStatus)
  }

  // Wait for output to clean up
  outW.Close()
  errW.Close()
  <-outDoneCh
  <-errDoneCh

  return err
}

func (p *provisioner) copyOutput(o terraform.UIOutput, r io.Reader, doneCh chan<- struct{}) {
  defer close(doneCh)
  lr := linereader.New(r)
  for line := range lr.Ch {
    o.Output(line)
  }
}

// retryFunc is used to retry a function for a given duration
func retryFunc(timeout time.Duration, f func() error) error {
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

func decodeConfig(d *schema.ResourceData) (*provisioner, error) {
  p := &provisioner{
    useSudo:        d.Get("use_sudo").(bool),
    skipInstall:    d.Get("skip_install").(bool),
    skipCleanup:    d.Get("skip_cleanup").(bool),
    installVersion: d.Get("install_version").(string),
    Plays:          make([]play, 0),
    InventoryMeta:  ansibleInventoryMeta{
      Hosts:             getStringList(d.Get("hosts")),
      Groups:            getStringList(d.Get("groups")),
    },
    Shared:         ansibleCallArgsShared{
      Become:            d.Get("become").(string),
      BecomeMethod:      d.Get("become_method").(string),
      BecomeUser:        d.Get("become_user").(string),
      ExtraVars:         getStringMap(d.Get("extra_vars")),
      Forks:             d.Get("forks").(int),
      Limit:             d.Get("limit").(string),
      VaultPasswordFile: d.Get("vault_password_file").(string),
      Verbose:           d.Get("verbose").(string),
    },
  }
  p.InventoryMeta = ensureLocalhostInCallArgsHosts(p.InventoryMeta)
  p.Plays = decodePlays(d.Get("plays").([]interface{}), p.InventoryMeta, p.Shared)
  return p, nil
}

func decodePlays(v []interface{}, fallbackInventoryMeta ansibleInventoryMeta, fallbackArgs ansibleCallArgsShared) []play {
  plays := make([]play, 0, len(v))
  for _, rawPlayData := range v {

    callable := ""
    callableType := AnsibleCallable_Undefined
    playData := rawPlayData.(map[string]interface{})
    playbook := (playData["playbook"].(string))
    module   := (playData["module"].(string))

    if len(playbook) > 0 && len(module) > 0 {
      callableType = AnsibleCallable_Conflicting
    } else {
      if len(playbook) > 0 {
        callable = playbook
        callableType = AnsibleCallable_Playbook
      } else if len(module) > 0 {
        callable = module
        callableType = AnsibleCallable_Module
      } else {
        callableType = AnsibleCallable_Undefined
      }
    }

    playToAppend := play{
      Callable:      callable,
      CallableType:  callableType,
      InventoryMeta: ansibleInventoryMeta{
        Hosts:  withStringListFallback(getStringList(playData["hosts"]), fallbackInventoryMeta.Hosts),
        Groups: withStringListFallback(getStringList(playData["groups"]), fallbackInventoryMeta.Groups),
      },
      CallArgs:      ansibleCallArgs{
        Shared: ansibleCallArgsShared{
          Become:            withStringFallback(playData["become"].(string), defaultBecome, fallbackArgs.Become),
          BecomeMethod:      withStringFallback(playData["become_method"].(string), defaultBecomeMethod, fallbackArgs.BecomeMethod),
          BecomeUser:        withStringFallback(playData["become_user"].(string), defaultBecomeUser, fallbackArgs.BecomeUser),
          ExtraVars:         withStringInterfaceMapFallback(getStringMap(playData["extra_vars"]), fallbackArgs.ExtraVars),
          Forks:             withIntFallback(playData["forks"].(int), defaultForks, fallbackArgs.Forks),
          Limit:             withStringFallback(playData["limit"].(string), defaultLimit, fallbackArgs.Limit),
          VaultPasswordFile: withStringFallback(playData["vault_password_file"].(string), defaultVaultPasswordFile, fallbackArgs.VaultPasswordFile),
          Verbose:           withStringFallback(playData["verbose"].(string), defaultVerbose, fallbackArgs.Verbose),
        },
        // module only:
        Args:        getStringMap(playData["args"]),
        Background:  playData["background"].(int),
        HostPattern: playData["host_pattern"].(string),
        OneLine:     playData["one_line"].(string),
        Poll:        playData["poll"].(int),
        // playbook only:
        ForceHandlers: playData["force_handlers"].(string),
        SkipTags:      getStringList(playData["skip_tags"]),
        StartAtTask:   playData["start_at_task"].(string),
        Tags:          getStringList(playData["tags"]),
      },
    }
    playToAppend.InventoryMeta = ensureLocalhostInCallArgsHosts(playToAppend.InventoryMeta)

    plays = append(plays, playToAppend)
  }
  return plays
}

func ensureLocalhostInCallArgsHosts(inventoryMeta ansibleInventoryMeta) ansibleInventoryMeta {
  lc := "localhost"
  found := false
  for _, v := range inventoryMeta.Hosts {
    if v == lc {
      found = true
      break
    }
  }
  if !found {
    inventoryMeta.Hosts = append(inventoryMeta.Hosts, lc)
  }
  return inventoryMeta
}

func getStringList(v interface{}) []string {
  var result []string
  switch v := v.(type) {
  case nil:
    return result
  case []interface{}:
    for _, vv := range v {
      if vv, ok := vv.(string); ok {
        result = append(result, vv)
      }
    }
    return result
  default:
    panic(fmt.Sprintf("Unsupported type: %T", v))
  }
}

func getStringMap(v interface{}) map[string]interface{} {
  switch v := v.(type) {
  case nil:
    return make(map[string]interface{})
  case map[string]interface{}:
    return v
  default:
    panic(fmt.Sprintf("Unsupported type: %T", v))
  }
}

func withStringFallback(intended string, defaultValue string, fallback string) string {
  if intended == defaultValue {
    return fallback
  }
  return intended
}

func withIntFallback(intended int, defaultValue int, fallback int) int {
  if intended == defaultValue {
    return fallback
  }
  return intended
}

func withStringListFallback(intended []string, fallback []string) []string {
  if len(intended) == 0 {
    return fallback
  }
  return intended
}
func withStringInterfaceMapFallback(intended map[string]interface{}, fallback map[string]interface{}) map[string]interface{} {
  if len(intended) == 0 {
    return fallback
  }
  return intended
}


func getMD5Hash(text string) string {
    hasher := md5.New()
    hasher.Write([]byte(text))
    return hex.EncodeToString(hasher.Sum(nil))
}