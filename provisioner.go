package main

import (
  "context"
  "encoding/json"
  "fmt"
  "io"
  "log"
  "os"
  "path/filepath"
  "strings"
  "time"

  "github.com/hashicorp/terraform/communicator"
  "github.com/hashicorp/terraform/communicator/remote"
  "github.com/hashicorp/terraform/helper/schema"
  "github.com/hashicorp/terraform/terraform"

  "github.com/mitchellh/go-homedir"
  "github.com/mitchellh/go-linereader"
)

type provisioner struct {
  ModulePath  string
  Playbook    string
  Plays       []string
  Hosts       []string
  Groups      []string
  UseSudo     bool
  ExtraVars   map[string]string
  SkipInstall bool
}

func Provisioner() terraform.ResourceProvisioner {
  return &schema.Provisioner{
    Schema: map[string]*schema.Schema{
      "module_path": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default: "ansible",
      },
      "use_sudo": &schema.Schema{
        Type:     schema.TypeBool,
        Optional: true,
        Default:  true,
      },
      "playbook": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default: "playbook.yaml",
      },
      "plays": &schema.Schema{
        Type:     schema.TypeList,
        Elem:     &schema.Schema{ Type: schema.TypeString },
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
      "extra_vars": &schema.Schema{
        Type:     schema.TypeMap,
        Optional: true,
        Computed: true,
      },
      "skip_install": &schema.Schema{
        Type:     schema.TypeBool,
        Optional: true,
        Default:  false,
      },
    },
    ApplyFunc:    applyFn,
    //ValidateFunc: validateFn,
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

  if !p.SkipInstall {
    if err := p.installAnsible(o, comm); err != nil {
      return err
    }
  }

  if err := p.deployAnsibleModule(o, comm); err != nil {
    o.Output(fmt.Sprintf("%+v", err))
    return err
  }

  return nil

}

func (p *provisioner) installAnsible(o terraform.UIOutput, comm communicator.Communicator) error {
  o.Output("Installing ansible...")
  provisionAnsibleCommands := []string{
    // https://github.com/hashicorp/terraform/issues/1025
    // cloud-init runs on fresh sources and can interfere with apt-get update commands causing intermittent failures
    "/bin/bash -c 'until [[ -f /var/lib/cloud/instance/boot-finished ]]; do sleep 1; done'",
    "/bin/bash -c ' if [[ -f /etc/redhat-release ]];then yum update -y && yum groupinstall -y \"Development Tools\" &&  yum install -y python-devel; else apt-get update && apt-get install -y build-essential python-dev; fi'",
    "curl https://bootstrap.pypa.io/get-pip.py | sudo python",
    "pip install ansible",
  }
  for _, command := range provisionAnsibleCommands {
    o.Output(fmt.Sprintf("running command: %s", command))
    err := p.runCommand(o, comm, command)
    if err != nil {
      return err
    }
  }
  o.Output("Ansible installed.")
  return nil
}

func (p *provisioner) deployAnsibleModule(o terraform.UIOutput, comm communicator.Communicator) error {
  // parse the playbook path and ensure that it is valid
  pathToResolve := p.Playbook
  if p.ModulePath != "" {
    pathToResolve = filepath.Join(p.ModulePath, p.Playbook)
  }

  playbookPath, err := p.resolvePath(pathToResolve)
  if err != nil {
    return err
  }

  // playbook file is at the top level of the module
  // parse the playbook path's directory and upload the entire directory
  playbookDir := filepath.Dir(playbookPath)

  remotePlaybookPath := filepath.Join("/tmp/ansible-terraform-bootstrap", filepath.Base(playbookPath))

  // upload ansible source and playbook to the host
  if err := comm.UploadDir("/tmp/ansible-terraform-bootstrap", playbookDir); err != nil {
    return err
  }

  extraVars, err := json.Marshal(p.ExtraVars)
  if err != nil {
    return err
  }

  // build a command to run ansible on the host machine
  command := fmt.Sprintf("ansible - --playbook=%s --hosts=%s --plays=%s --groups=%s --extra-vars=%s",
    remotePlaybookPath,
    strings.Join(p.Hosts, ","),
    strings.Join(p.Plays, ","),
    strings.Join(p.Groups, ","),
    string(extraVars))

  //o.Output(fmt.Sprintf("running command: %s", command))
  //if err := p.runCommand(o, comm, command); err != nil {
  //  return err
  //}

  o.Output(command)

  return nil
}

func (p *provisioner) resolvePath(path string) (string, error) {
  expandedPath, _ := homedir.Expand(path)
  if _, err := os.Stat(expandedPath); err == nil {
    return expandedPath, nil
  }

  cwd, err := os.Getwd()
  if err != nil {
    return "", fmt.Errorf("Unable to get current working directory to resolve path as a relative path")
  }

  relativePath := filepath.Join(cwd, path)
  if _, err := os.Stat(relativePath); err == nil {
    return relativePath, nil
  }

  modulePath := filepath.Join(p.ModulePath, path)
  if _, err := os.Stat(modulePath); err == nil {
    return modulePath, nil
  }

  return "", fmt.Errorf("Ansible module not found at path: [%s]", path)
}

// runCommand is used to run already prepared commands
func (p *provisioner) runCommand(o terraform.UIOutput, comm communicator.Communicator, command string) error {
  // Unless prevented, prefix the command with sudo
  if p.UseSudo {
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
/*
func validateFn(c *terraform.ResourceConfig) (ws []string, es []error) {

  if modulePath := c.Get("module")
  playbook := c.Get("playbook")

  if fmt.Sprintf("%+v::%+v", modulePath, playbook) == "::" {
    es = append(es, errors.New("\"module\" and \"playbook\" are empty: set at least one"))
  }

  return ws, es
}
*/
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
    ModulePath:  d.Get("module_path").(string),
    Playbook:    d.Get("playbook").(string),
    Plays:       getStringList(d.Get("plays")),
    Hosts:       getStringList(d.Get("hosts")),
    Groups:      getStringList(d.Get("groups")),
    UseSudo:     d.Get("use_sudo").(bool),
    ExtraVars:   getStringMap(d.Get("extra_vars")),
    SkipInstall: d.Get("skip_install").(bool),
  }
  return p, nil
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

func getStringMap(v interface{}) map[string]string {
  result := make(map[string]string)
  switch v := v.(type) {
  case nil:
    return result
  case map[string]interface{}:
    for key, val := range v {
      result[key] = fmt.Sprintf("%+v", val)
    }
    return result
  default:
    panic(fmt.Sprintf("Unsupported type: %T", v))
  }
}