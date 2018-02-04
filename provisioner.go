package main

import (
  "bufio"
  "bytes"
  "context"
  //"encoding/json"
  "fmt"
  "io"
  "log"
  "os"
  "path/filepath"
  //"strings"
  "text/template"
  "time"

  "github.com/hashicorp/terraform/communicator"
  "github.com/hashicorp/terraform/communicator/remote"
  "github.com/hashicorp/terraform/helper/schema"
  "github.com/hashicorp/terraform/terraform"

  "github.com/mitchellh/go-homedir"
  "github.com/mitchellh/go-linereader"
)

const (
  bootstrapDirectory string = "/tmp/ansible-terraform-bootstrap"
  defaultStartAtTask string = ""
  defaultLimit string = ""
  defaultForks int = 5
  defaultVerbose bool = false
  defaultForceHandlers bool = false
  defaultOneLine bool = false
  defaultBecome bool = false
  defaultBecomeMethod string = "sudo"
  defaultBecomeUser string = "root"
  defaultVaultPasswordFile string = ""
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

var inventoryFilePath string = filepath.Join(bootstrapDirectory, ".inventory-ansible-bootstrap/hosts")

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

type ansibleCallable struct {
  Callable string
  CallableType AnsibleCallbaleType
  CallArgs ansibleCallArgs
}

type ansibleCallArgs struct {
  Hosts             []string
  Groups            []string
  Tags              []string
  SkipTags          []string
  StartAtTask       string
  Limit             string
  Forks             int
  ModuleArgs        map[string]interface{}
  ExtraVars         map[string]interface{}
  Verbose           bool
  ForceHandlers     bool
  OneLine           bool

  Become            bool
  BecomeMethod      string
  BecomeUser        string

  VaultPasswordFile string
}

type play struct {
  Callable ansibleCallable
}

type provisioner struct {
  Plays             []play
  CallArgs          ansibleCallArgs
  useSudo           bool
  skipInstall       bool
  skipCleanup       bool
  installVersion    string
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
            "tags": &schema.Schema{
              Type:     schema.TypeList,
              Elem:     &schema.Schema{ Type: schema.TypeString },
              Optional: true,
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
            "limit": &schema.Schema{
              Type:     schema.TypeString,
              Optional: true,
              Default: defaultLimit,
            },
            "extra_vars": &schema.Schema{
              Type:     schema.TypeMap,
              Optional: true,
              Computed: true,
            },
            "module_args": &schema.Schema{
              Type:     schema.TypeMap,
              Optional: true,
              Computed: true,
            },
            "forks": &schema.Schema{
              Type:     schema.TypeInt,
              Optional: true,
              Default: defaultForks,
            },
            "verbose": &schema.Schema{
              Type:     schema.TypeBool,
              Optional: true,
              Default:  defaultVerbose,
            },
            "force_handlers": &schema.Schema{
              Type:     schema.TypeBool,
              Optional: true,
              Default:  defaultForceHandlers,
            },
            "one_line": &schema.Schema{
              Type:     schema.TypeBool,
              Optional: true,
              Default:  defaultOneLine,
            },

            "become": &schema.Schema{
              Type:     schema.TypeBool,
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
            "vault_password_file": &schema.Schema{
              Type:     schema.TypeString,
              Optional: true,
              Default:  defaultVaultPasswordFile,
            },

          },
        },
      },

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
      "tags": &schema.Schema{
        Type:     schema.TypeList,
        Elem:     &schema.Schema{ Type: schema.TypeString },
        Optional: true,
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
      "limit": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default: defaultLimit,
      },
      "extra_vars": &schema.Schema{
        Type:     schema.TypeMap,
        Optional: true,
        Computed: true,
      },
      "module_args": &schema.Schema{
        Type:     schema.TypeMap,
        Optional: true,
        Computed: true,
      },
      "forks": &schema.Schema{
        Type:     schema.TypeInt,
        Optional: true,
        Default: defaultForks,
      },
      "verbose": &schema.Schema{
        Type:     schema.TypeBool,
        Optional: true,
        Default:  defaultVerbose,
      },
      "force_handlers": &schema.Schema{
        Type:     schema.TypeBool,
        Optional: true,
        Default:  defaultForceHandlers,
      },
      "one_line": &schema.Schema{
        Type:     schema.TypeBool,
        Optional: true,
        Default:  defaultOneLine,
      },

      "become": &schema.Schema{
        Type:     schema.TypeBool,
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

      "vault_password_file": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default:  defaultVaultPasswordFile,
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
    if err := p.installAnsible(o, comm); err != nil {
      return err
    }
  }

  // TODO: restore
  //if err := p.deployAnsibleModule(o, comm); err != nil {
  //  o.Output(fmt.Sprintf("%+v", err))
  //  return err
  //}

  return nil

}

/*
func (p *provisioner) deployAnsibleModule(o terraform.UIOutput, comm communicator.Communicator) error {
  
  playbookPath, err := p.resolvePath("", o) // TODO: fixme
  //playbookPath, err := p.resolvePath(p.Playbook, o)
  if err != nil {
    return err
  }

  // playbook file is at the top level of the module
  // parse the playbook path's directory and upload the entire directory
  playbookDir := filepath.Dir(playbookPath)

  remotePlaybookPath := filepath.Join(bootstrapDirectory, filepath.Base(playbookPath))

  // upload ansible source and playbook to the host
  if err := comm.UploadDir(bootstrapDirectory, playbookDir); err != nil {
    return err
  }

  vaultPasswordFilePath := p.VaultPasswordFile
  uploadedVaultPasswordFilePath := ""
  if len(vaultPasswordFilePath) > 0 {
    vaultPasswordFilePath, err = p.resolvePath(vaultPasswordFilePath, o)
    if err != nil {
      return err
    }
    uploadedVaultPasswordFilePath, err = p.uploadVaultPasswordFile(o, comm, vaultPasswordFilePath)
    if err != nil {
      return err
    }
  }

  // build a command to run ansible on the host machine
  command, err := p.commandBuilder(remotePlaybookPath, uploadedVaultPasswordFilePath)
  if err != nil {
    return err
  }

  // create temp inventory:
  if err = p.uploadInventory(o, comm); err != nil {
    return err
  }

  o.Output(fmt.Sprintf("running command: %s", command))
  if err := p.runCommandSudo(o, comm, command); err != nil {
    return err
  }

  if !p.skipCleanup {
    p.cleanupAfterBootstrap(o, comm)
  }

  return nil
}
*/

func (p *provisioner) installAnsible(o terraform.UIOutput, comm communicator.Communicator) error {

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

func (p *provisioner) uploadVaultPasswordFile(o terraform.UIOutput, comm communicator.Communicator, passwordFilePath string) (string, error) {

  passwordFileName := filepath.Base(passwordFilePath)
  targetPath := filepath.Join(bootstrapDirectory, ".vault-ansible-bootstrap", passwordFileName)

  // create the directory for vault password file:
  p.runCommandNoSudo(o, comm, fmt.Sprintf("mkdir -p %s", filepath.Dir(targetPath)))

  o.Output(fmt.Sprintf("Uploading ansible vault password file to '%s'...", targetPath))

  file, err := os.Open(passwordFilePath)
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

func (p *provisioner) uploadInventory(o terraform.UIOutput, comm communicator.Communicator) error {
  o.Output("Generating ansible inventory...")
  t := template.Must(template.New("hosts").Parse(inventoryTemplate))
  var buf bytes.Buffer
  err := t.Execute(&buf, p)
  if err != nil {
    return fmt.Errorf("Error executing 'hosts' template: %s", err)
  }
  targetPath := inventoryFilePath

  // create directory for the inventory file:
  p.runCommandNoSudo(o, comm, fmt.Sprintf("mkdir -p %s", filepath.Dir(targetPath)))

  o.Output(fmt.Sprintf("Uploading ansible inventory to %s...", targetPath))
  if err := comm.Upload(targetPath, bytes.NewReader(buf.Bytes())); err != nil {
    return err
  }
  o.Output("Ansible inventory uploaded.")
  return nil
}

func (p *provisioner) cleanupAfterBootstrap(o terraform.UIOutput, comm communicator.Communicator) {
  o.Output("Cleaning up after bootstrap...")
  p.runCommandNoSudo(o, comm, fmt.Sprintf("rm -r %s", bootstrapDirectory))
  o.Output("Cleanup complete.")
}

/*
func (p *provisioner) commandBuilder(playbookFile string, uploadedVaultPasswordFilePath string) (string, error) {

  command := fmt.Sprintf("ansible-playbook %s", playbookFile)
  command = fmt.Sprintf("%s --inventory-file=%s", command, inventoryFilePath)
  if len(p.ExtraVars) > 0 {
    extraVars, err := json.Marshal(p.ExtraVars)
    if err != nil {
      return "", err
    }
    command = fmt.Sprintf("%s --extra-vars='%s'", command, string(extraVars))
  }
  if len(p.SkipTags) > 0 {
    command = fmt.Sprintf("%s --skip-tags=%s", command, strings.Join(p.SkipTags, ","))
  }
  if len(p.Tags) > 0 {
    command = fmt.Sprintf("%s --tags=%s", command, strings.Join(p.Tags, ","))
  }
  if len(uploadedVaultPasswordFilePath) > 0 {
    command = fmt.Sprintf("%s --vault-password-file=%s", command, uploadedVaultPasswordFilePath)
  }
  if len(p.StartAtTask) > 0 {
    command = fmt.Sprintf("%s --start-at-task=%s", command, p.StartAtTask)
  }
  if len(p.Limit) > 0 {
    command = fmt.Sprintf("%s --limit=%s", command, p.Limit)
  }
  if p.Forks > 0 {
    command = fmt.Sprintf("%s --forks=%d", command, p.Forks)
  }
  if p.Verbose {
    command = fmt.Sprintf("%s --verbose", command)
  }
  if p.ForceHandlers {
    command = fmt.Sprintf("%s --force-handlers", command)
  }
  if p.Become {
    command = fmt.Sprintf("%s --become --become-method='%s' --become-user='%s'", command, p.BecomeMethod, p.BecomeUser)
  }
  return command, nil
}
*/

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
    CallArgs:       ansibleCallArgs{
      Hosts:             getStringList(d.Get("hosts")),
      Groups:            getStringList(d.Get("groups")),
      Tags:              getStringList(d.Get("tags")),
      SkipTags:          getStringList(d.Get("skip_tags")),
      StartAtTask:       d.Get("start_at_task").(string),
      Limit:             d.Get("limit").(string),
      Forks:             d.Get("forks").(int),
      ExtraVars:         getStringMap(d.Get("extra_vars")),
      ModuleArgs:        getStringMap(d.Get("module_args")),
      Verbose:           d.Get("verbose").(bool),
      ForceHandlers:     d.Get("force_handlers").(bool),
      OneLine:           d.Get("one_line").(bool),

      Become:            d.Get("become").(bool),
      BecomeMethod:      d.Get("become_method").(string),
      BecomeUser:        d.Get("become_user").(string),

      VaultPasswordFile: d.Get("vault_password_file").(string),
    },
  }
  p.CallArgs = ensureLocalhostInCallArgsHosts(p.CallArgs)
  p.Plays = decodePlays(d.Get("plays").([]interface{}), p.CallArgs)
  return p, nil
}

func decodePlays(v []interface{}, fallback ansibleCallArgs) []play {
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
      Callable: ansibleCallable{
        Callable: callable,
        CallableType: callableType,
        CallArgs: ansibleCallArgs{
          Hosts:             withStringListFallback(getStringList(playData["hosts"]), fallback.Hosts),
          Groups:            withStringListFallback(getStringList(playData["groups"]), fallback.Groups),
          Tags:              withStringListFallback(getStringList(playData["tags"]), fallback.Tags),
          SkipTags:          withStringListFallback(getStringList(playData["skip_tags"]), fallback.SkipTags),
          StartAtTask:       withStringFallback((playData["start_at_task"].(string)), defaultStartAtTask, fallback.StartAtTask),
          Limit:             withStringFallback((playData["limit"].(string)), defaultLimit, fallback.Limit),
          Forks:             withIntFallback((playData["forks"].(int)), defaultForks, fallback.Forks),
          ModuleArgs:        withStringInterfaceMapFallback(getStringMap(playData["module_args"]), fallback.ModuleArgs),
          ExtraVars:         withStringInterfaceMapFallback(getStringMap(playData["extra_vars"]), fallback.ExtraVars),
          Verbose:           withBoolFallback((playData["verbose"].(bool)), defaultVerbose, fallback.Verbose),
          ForceHandlers:     withBoolFallback((playData["force_handlers"].(bool)), defaultForceHandlers, fallback.ForceHandlers),
          OneLine:           withBoolFallback((playData["one_line"].(bool)), defaultOneLine, fallback.OneLine),

          Become:            withBoolFallback((playData["become"].(bool)), defaultBecome, fallback.Become),
          BecomeMethod:      withStringFallback((playData["become_method"].(string)), defaultBecomeMethod, fallback.BecomeMethod),
          BecomeUser:        withStringFallback((playData["become_user"].(string)), defaultBecomeUser, fallback.BecomeUser),

          VaultPasswordFile: withStringFallback((playData["vault_password_file"].(string)), defaultVaultPasswordFile, fallback.VaultPasswordFile),
        },
      },
    }
    playToAppend.Callable.CallArgs = ensureLocalhostInCallArgsHosts(playToAppend.Callable.CallArgs)

    plays = append(plays, playToAppend)
  }
  return plays
}

func ensureLocalhostInCallArgsHosts(callArgs ansibleCallArgs) ansibleCallArgs {
  lc := "localhost"
  found := false
  for _, v := range callArgs.Hosts {
    if v == lc {
      found = true
      break
    }
  }
  if !found {
    callArgs.Hosts = append(callArgs.Hosts, lc)
  }
  return callArgs
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
func withBoolFallback(intended bool, defaultValue bool, fallback bool) bool {
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