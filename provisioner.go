package main

import (
  "bytes"
  "context"
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
)

const (
  bootstrapDirectory string = "/tmp/ansible-terraform-bootstrap"
  templateInventory string = "hosts"
)

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

type provisioner struct {
  Playbook       string
  Plays          []string
  Hosts          []string
  Groups         []string
  Tags           []string
  SkipTags       []string
  StartAtTask    string
  Limit          string
  Forks          int
  ExtraVars      map[string]string
  Verbose        bool
  ForceHandlers  bool

  Become         bool
  BecomeMethod   string
  BecomeUser     string

  useSudo        bool
  skipInstall    bool
  installVersion string
}

func Provisioner() terraform.ResourceProvisioner {
  return &schema.Provisioner{
    Schema: map[string]*schema.Schema{
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
        Default: "",
      },
      "limit": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default: "",
      },
      "extra_vars": &schema.Schema{
        Type:     schema.TypeMap,
        Optional: true,
        Computed: true,
      },
      "forks": &schema.Schema{
        Type:     schema.TypeInt,
        Optional: true,
        Default: 0, // only added to the command when greater than 0
      },
      "verbose": &schema.Schema{
        Type:     schema.TypeBool,
        Optional: true,
        Default:  false,
      },
      "force_handlers": &schema.Schema{
        Type:     schema.TypeBool,
        Optional: true,
        Default:  false,
      },

      "become": &schema.Schema{
        Type:     schema.TypeBool,
        Optional: true,
        Default:  false,
      },
      "become_method": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default:  "sudo",
      },
      "become_user": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default:  "user",
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
      "install_version": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default:  "", // latest
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

  if !p.skipInstall {
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
  }

  if len(p.installVersion) > 0 {
    provisionAnsibleCommands = append(provisionAnsibleCommands, fmt.Sprintf("pip install ansible==%s", p.installVersion))
  } else {
    provisionAnsibleCommands = append(provisionAnsibleCommands, "pip install ansible")
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

  playbookPath, err := p.resolvePath(pathToResolve, o)
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

  // build a command to run ansible on the host machine
  command, err := p.commandBuilder(remotePlaybookPath)
  if err != nil {
    return err
  }

  // create temp inventory:
  if err = p.uploadInventory(o, comm); err != nil {
    return err
  }

  o.Output(fmt.Sprintf("running command: %s", command))
  if err := p.runCommand(o, comm, command); err != nil {
    return err
  }

  return nil
}

func (p *provisioner) resolvePath(path string, o terraform.UIOutput) (string, error) {
  expandedPath, _ := homedir.Expand(path)
  if _, err := os.Stat(expandedPath); err == nil {
    return expandedPath, nil
  }
  return "", fmt.Errorf("Ansible module not found at path: [%s]", path)
}

// runCommand is used to run already prepared commands
func (p *provisioner) runCommand(o terraform.UIOutput, comm communicator.Communicator, command string) error {
  // Unless prevented, prefix the command with sudo
  if p.useSudo {
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

func (p *provisioner) uploadInventory(o terraform.UIOutput, comm communicator.Communicator) error {
  o.Output("Generating ansible inventory...")
  t := template.Must(template.New(templateInventory).Parse(inventoryTemplate))
  var buf bytes.Buffer
  err := t.Execute(&buf, p)
  if err != nil {
    return fmt.Errorf("Error executing '%s' template: %s", templateInventory, err)
  }
  targetPath := filepath.Join(bootstrapDirectory, ".inventory/hosts")

  commands := []string{
    fmt.Sprintf("mkdir -p %s", filepath.Dir(targetPath)),
    fmt.Sprintf("chmod 0777 %s", filepath.Dir(targetPath)),
  }

  for _, command := range commands {
    p.runCommand(o, comm, command)
  }

  o.Output(fmt.Sprintf("Uploading ansible inventory to %s...", targetPath))
  if err := comm.Upload(targetPath, bytes.NewReader(buf.Bytes())); err != nil {
    return err
  }
  o.Output("Ansible inventory uploaded.")
  return nil
}

func (p *provisioner) copyOutput(o terraform.UIOutput, r io.Reader, doneCh chan<- struct{}) {
  defer close(doneCh)
  lr := linereader.New(r)
  for line := range lr.Ch {
    o.Output(line)
  }
}

func (p *provisioner) commandBuilder(playbookFile string) (string, error) {
  command := fmt.Sprintf("ansible-playbook %s", playbookFile)
  command = fmt.Sprintf("%s --inventory-file=%s", command, filepath.Join(filepath.Dir(playbookFile), ".inventory/hosts"))
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
    Playbook:       d.Get("playbook").(string),
    Plays:          getStringList(d.Get("plays")),
    Hosts:          getStringList(d.Get("hosts")),
    Groups:         getStringList(d.Get("groups")),
    Tags:           getStringList(d.Get("tags")),
    SkipTags:       getStringList(d.Get("skip_tags")),
    StartAtTask:    d.Get("start_at_task").(string),
    Limit:          d.Get("limit").(string),
    Forks:          d.Get("forks").(int),
    ExtraVars:      getStringMap(d.Get("extra_vars")),
    Verbose:        d.Get("verbose").(bool),
    ForceHandlers:  d.Get("force_handlers").(bool),

    Become:         d.Get("become").(bool),
    BecomeMethod:   d.Get("become_method").(string),
    BecomeUser:     d.Get("become_user").(string),

    useSudo:        d.Get("use_sudo").(bool),
    skipInstall:    d.Get("skip_install").(bool),
    installVersion: d.Get("install_version").(string),
  }
  p.Hosts = append(p.Hosts, "localhost")
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