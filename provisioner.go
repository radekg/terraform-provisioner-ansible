package main

import (
  "bufio"
  "bytes"
  "context"
  "crypto/md5"
  "errors"
  "encoding/hex"
  "fmt"
  "io"
  "io/ioutil"
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
  localExec "github.com/hashicorp/terraform/builtin/provisioners/local-exec"

  "github.com/mitchellh/go-homedir"
  "github.com/mitchellh/go-linereader"

  "github.com/satori/go.uuid"
)

const (
  bootstrapDirectory string = "/tmp/ansible-terraform-bootstrap"
  // shared:
  defaultBecome string = ""
  defaultBecome_Set string = "no"
  defaultBecomeMethod string = ""
  defaultBecomeMethod_Set string = "sudo"
  defaultBecomeUser string = ""
  defaultBecomeUser_Set string = "root"
  defaultForks int = 5
  defaultInventoryFile string = ""
  defaultLimit string = ""
  defaultVaultPasswordFile string = ""
  defaultVerbose string = ""
  // playbook only:
  defaultForceHandlers string = ""
  defaultStartAtTask string = ""
  // module only:
  defaultBackground int = 0
  defaultHostPattern string = "all"
  defaultOneLine string = ""
  defaultPoll int = 15
)

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
  InventoryFile     string
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

// -- provisioner:

type provisioner struct {
  Plays          []play
  InventoryMeta  ansibleInventoryMeta
  Shared         ansibleCallArgsShared
  useSudo        bool
  skipInstall    bool
  skipCleanup    bool
  installVersion string
  local          bool
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
              Default:  defaultBackground,
            },
            "host_pattern": &schema.Schema{
              Type:     schema.TypeString,
              Optional: true,
              Default:  defaultHostPattern,
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
              Default:  defaultStartAtTask,
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
              Default:  defaultForks,
            },
            "inventory_file": &schema.Schema{
              Type:     schema.TypeString,
              Optional: true,
              Default:  defaultInventoryFile,
            },
            "limit": &schema.Schema{
              Type:     schema.TypeString,
              Optional: true,
              Default:  defaultLimit,
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
        Default:  defaultForks,
      },
      "inventory_file": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default:  defaultInventoryFile,
      },
      "limit": &schema.Schema{
        Type:     schema.TypeString,
        Optional: true,
        Default:  defaultLimit,
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
      "local": &schema.Schema{
        Type:     schema.TypeBool,
        Optional: true,
        Default:  false,
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
  
  if !p.local {

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

  }

  if !p.local {
    if runnables, err := p.remote_deployAnsibleData(o, comm); err != nil {
      o.Output(fmt.Sprintf("%+v", err))
      return err
    } else {
      
      for _, runnable := range runnables {
        command, err := runnable.ToCommand()
        if err != nil {
          return err
        }
        o.Output(fmt.Sprintf("running command: %s", command))
        if err := p.remote_runCommandSudo(o, comm, command); err != nil {
          return err
        }
      }

    }
  } else {

    connType := s.Ephemeral.ConnInfo["type"]
    switch connType {
    case "ssh", "": // The default connection type is ssh, so if connType is empty use ssh
    default:
      return errors.New("Currently, only SSH connection is supported.")
    }

    connInfo, err := parseConnectionInfo(s)
    if err != nil {
      return err
    }

    if connInfo.PrivateKey == "" || connInfo.User == "" || connInfo.Host == "" {
      return errors.New("Local mode requires a connection with a pem file, username and host.")
    }

    if runnables, err := p.local_gatherRunnables(o, connInfo); err != nil {
      o.Output(fmt.Sprintf("%+v", err))
      return err
    } else {

      pemFile, err := p.local_writePem(o, connInfo)
      if err != nil {
        return err
      }
      defer os.Remove(pemFile)

      knownHostsFile, err := p.local_ensureKnownHosts(o, connInfo)
      if err != nil {
        return err
      }
      defer os.Remove(knownHostsFile)

      for _, runnable := range runnables {

        if runnable.InventoryFileTemporary {
          defer os.Remove(runnable.InventoryFile)
        }

        command, err := runnable.ToLocalCommand(o, runnablePlayLocalAnsibleArgs{
          Username: connInfo.User,
          PemFile: pemFile,
          KnownHostsFile: knownHostsFile,
        })
        if err != nil {
          return err
        }

        o.Output(fmt.Sprintf("running local command: %s", command))
        if err := p.local_runCommand(o, command); err != nil {
          return err
        }
      }

    }
  }

  if !p.local && !p.skipCleanup {
    p.remote_cleanupAfterBootstrap(o, comm)
  }

  return nil

}

func validateFn(c *terraform.ResourceConfig) (ws []string, es []error) {
  
  becomeMethod, ok := c.Get("become_method")
  if ok {
    if !becomeMethods[becomeMethod.(string)] {
      es = append(es, errors.New(becomeMethod.(string)+" is not a valid become_method"))
    }
  }

  fields := []string{"verbose", "force_handlers", "one_line", "become"}
  for _, field := range fields {
    v, ok := c.Get(field)
    if ok && v.(string) != "" {
      if !yesNoStates[v.(string)] {
        es = append(es, errors.New(v.(string)+" is not a valid " + field))
      }
    }
  }

  for _, ftt := range []string{"inventory_file", "vault_password_file"} {
    value, ok := c.Get(ftt)
    if ok && len(value.(string)) > 0 {
      if _, err := resolvePath(value.(string)); err != nil {
        es = append(es, errors.New("file " + value.(string) + " does not exist"))
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
          es = append(es, errors.New(becomeMethodPlay.(string)+" is not a valid become_method"))
        }
      }

      for _, fieldPlay := range fields {
        v, ok := p[fieldPlay]
        if ok && v.(string) != "" {
          if !yesNoStates[v.(string)] {
            es = append(es, errors.New(v.(string)+" is not a valid " + fieldPlay))
          }
        }
      }

      for _, ftt := range []string{"inventory_file", "playbook", "vault_password_file"} {
        value, ok := p[ftt]
        if ok && len(value.(string)) > 0 {
          if _, err := resolvePath(value.(string)); err != nil {
            es = append(es, errors.New("file " + value.(string) + " does not exist"))
          }
        }
      }

    }
  } else {
    ws = append(ws, "nothing to play")
  }

  local, ok := c.Get("local")
  if ok {
    if local.(bool) {
      for _, cf := range []string{"use_sudo", "install_version", "skip_cleanup", "skip_install"} {
        if _, ok := c.Get(cf); ok {
          es = append(es, errors.New(cf + " must not be used when local = true"))
        }
      }
    }
  }

  return ws, es
}

func (p *provisioner) remote_deployAnsibleData(o terraform.UIOutput, comm communicator.Communicator) ([]runnablePlay, error) {
  
  response := make([]runnablePlay, 0)
  
  for _, playDef := range p.Plays {
    if playDef.CallableType == AnsibleCallable_Playbook {

      playbookPath, err := resolvePath(playDef.Callable)
      if err != nil {
        return response, err
      }

      // playbook file is at the top level of the module
      // parse the playbook path's directory and upload the entire directory
      playbookDir := filepath.Dir(playbookPath)
      playbookDirHash := getMD5Hash(playbookDir)

      remotePlaybookDir := filepath.Join(bootstrapDirectory, playbookDirHash)
      remotePlaybookPath := filepath.Join(remotePlaybookDir, filepath.Base(playbookPath))

      if err := p.remote_runCommandNoSudo(o, comm, fmt.Sprintf("mkdir -p %s", bootstrapDirectory)); err != nil {
        return response, err
      }

      errCmdCheck := p.remote_runCommandNoSudo(o, comm, fmt.Sprintf("/bin/bash -c 'if [ -d \"%s\" ]; then exit 50; fi'", remotePlaybookDir))
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
      inventoryFile, err := p.remote_writeInventory(o, comm, remotePlaybookDir, playDef.CallArgs, playDef.InventoryMeta)
      if err != nil {
        return response, err
      }

      response = append(response, runnablePlay{
        Play:                   playDef,
        VaultPasswordFile:      uploadedVaultPasswordFilePath,
        InventoryFile:          inventoryFile,
        InventoryFileTemporary: len(playDef.CallArgs.Shared.InventoryFile) == 0,
      })

    } else if playDef.CallableType == AnsibleCallable_Module {

      if err := p.remote_runCommandNoSudo(o, comm, fmt.Sprintf("mkdir -p %s", bootstrapDirectory)); err != nil {
        return response, err
      }

      // always upload vault password file:
      uploadedVaultPasswordFilePath, err := p.remote_uploadVaultPasswordFile(o, comm, bootstrapDirectory, playDef.CallArgs.Shared)
      if err != nil {
        return response, err
      }

      // always create temp inventory:
      inventoryFile, err := p.remote_writeInventory(o, comm, bootstrapDirectory, playDef.CallArgs, playDef.InventoryMeta)
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

  if err := p.remote_runCommandSudo(o, comm, fmt.Sprintf("/bin/bash -c '%s && rm %s'", targetPath, targetPath)); err != nil {
    return err
  }

  o.Output("Ansible installed.")
  return nil
}

func (p *provisioner) remote_uploadVaultPasswordFile(o terraform.UIOutput, comm communicator.Communicator, destination string, callArgs ansibleCallArgsShared) (string, error) {

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

func (p *provisioner) remote_writeInventory(o terraform.UIOutput, comm communicator.Communicator, destination string, callArgs ansibleCallArgs, inventoryMeta ansibleInventoryMeta) (string, error) {
  
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

  } else {

    o.Output("Generating temporary ansible inventory...")
    t := template.Must(template.New("hosts").Parse(inventoryTemplate_Remote))
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
}

func (p *provisioner) remote_cleanupAfterBootstrap(o terraform.UIOutput, comm communicator.Communicator) {
  o.Output("Cleaning up after bootstrap...")
  p.remote_runCommandNoSudo(o, comm, fmt.Sprintf("rm -r %s", bootstrapDirectory))
  o.Output("Cleanup complete.")
}

func (p *provisioner) remote_runCommandSudo(o terraform.UIOutput, comm communicator.Communicator, command string) error {
  return p.remote_runCommand(o, comm, command, true)
}

func (p *provisioner) remote_runCommandNoSudo(o terraform.UIOutput, comm communicator.Communicator, command string) error {
  return p.remote_runCommand(o, comm, command, false)
}

// runCommand is used to run already prepared commands
func (p *provisioner) remote_runCommand(o terraform.UIOutput, comm communicator.Communicator, command string, shouldSudo bool) error {
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

// -- LOCAL:

func (p *provisioner) local_ensureKnownHosts(o terraform.UIOutput, connInfo *connectionInfo) (string, error) {

  if connInfo.Host == "" {
    return "", errors.New("Host could not be established from the connection info.")
  }
  u1 := uuid.Must(uuid.NewV4())
  targetPath := filepath.Join(os.TempDir(), u1.String())

  for {
    sshKeyScanCommand := fmt.Sprintf("ssh-keyscan %s 2>/dev/null | head -n1 > %s", connInfo.Host, targetPath)
    if err := p.local_runCommand(o, sshKeyScanCommand); err != nil {
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
    }
  }

  return targetPath, nil
}

func (p *provisioner) local_writePem(o terraform.UIOutput, connInfo *connectionInfo) (string, error) {
  if connInfo.PrivateKey != "" {
    file, err := ioutil.TempFile(os.TempDir(), "temporary-private-key.pem")
    defer file.Close()
    if err != nil {
      return "", err
    }

    o.Output(fmt.Sprintf("Writing temprary PEM to '%s'...", file.Name()))
    if err := ioutil.WriteFile(file.Name(), []byte(connInfo.PrivateKey), 0400); err != nil {
      return "", err
    } else {
      o.Output("Ansible inventory written.")
      return file.Name(), nil
    }
  }
  return "", nil
}

func (p *provisioner) local_gatherRunnables(o terraform.UIOutput, connInfo *connectionInfo) ([]runnablePlay, error) {

  response := make([]runnablePlay, 0)
  for _, playDef := range p.Plays {
    if playDef.CallableType == AnsibleCallable_Playbook {
      inventoryFile, err := p.local_writeInventory(o, connInfo, playDef.CallArgs, playDef.InventoryMeta)
      if err != nil {
        return response, err
      }
      response = append(response, runnablePlay{
        Play: playDef,
        VaultPasswordFile:      playDef.CallArgs.Shared.VaultPasswordFile,
        InventoryFile:          inventoryFile,
        InventoryFileTemporary: len(playDef.CallArgs.Shared.InventoryFile) == 0,
      })
    } else if playDef.CallableType == AnsibleCallable_Module {
      inventoryFile, err := p.local_writeInventory(o, connInfo, playDef.CallArgs, playDef.InventoryMeta)
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

func (p *provisioner) local_writeInventory(o terraform.UIOutput, connInfo *connectionInfo, callArgs ansibleCallArgs, inventoryMeta ansibleInventoryMeta) (string, error) {
  if len(callArgs.Shared.InventoryFile) == 0 {
    if connInfo.Host == "" {
      return "", errors.New("Host could not be established from the connection info.")
    }

    inplaceMeta := ansibleInventoryMeta{
      Hosts: []string{ connInfo.Host },
      Groups: inventoryMeta.Groups,
    }

    o.Output("Generating temporary ansible inventory...")
    t := template.Must(template.New("hosts").Parse(inventoryTemplate_Local))
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
    } else {
      o.Output("Ansible inventory written.")
      return file.Name(), nil
    }
  } else {
    return callArgs.Shared.InventoryFile, nil
  }
}

func (p *provisioner) local_runCommand(o terraform.UIOutput, command string) error {
  localExecProvisioner := localExec.Provisioner()

  instanceState := &terraform.InstanceState{
    ID: command,
    Attributes: make(map[string]string),
    Ephemeral: terraform.EphemeralState{
      ConnInfo: make(map[string]string),
      Type: "local-exec",
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

// -- UTILITY:

func (p *provisioner) copyOutput(o terraform.UIOutput, r io.Reader, doneCh chan<- struct{}) {
  defer close(doneCh)
  lr := linereader.New(r)
  for line := range lr.Ch {
    o.Output(line)
  }
}

func resolvePath(path string) (string, error) {
  expandedPath, _ := homedir.Expand(path)
  if _, err := os.Stat(expandedPath); err == nil {
    return expandedPath, nil
  }
  return "", fmt.Errorf("Ansible module not found at path: [%s]", path)
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
    local:          d.Get("local").(bool),
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
      InventoryFile:     d.Get("inventory_file").(string),
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
          InventoryFile:     withStringFallback(playData["inventory_file"].(string), defaultInventoryFile, fallbackArgs.InventoryFile),
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