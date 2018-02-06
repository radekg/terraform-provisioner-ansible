package main

import (
  "encoding/json"
  "fmt"
  "strings"

  "github.com/hashicorp/terraform/terraform"
)

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
    if p.CallArgs.ForceHandlers == yes {
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
    if p.CallArgs.OneLine == yes {
      command = fmt.Sprintf("%s --one-line", command)
    }

  }
  // inventory file:
  command = fmt.Sprintf("%s --inventory-file='%s'", command, inventoryFile)

  // shared arguments:

  // become:
  if p.CallArgs.Shared.Become == yes {
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
  if p.CallArgs.Shared.Verbose == yes {
    command = fmt.Sprintf("%s --verbose", command)
  }

  return command, nil
}

// -- runnable play:

type runnablePlay struct {
  Play                   play
  VaultPasswordFile      string
  InventoryFile          string
  InventoryFileTemporary bool
}

func (r *runnablePlay) ToCommand() (string, error) {
  return r.Play.ToCommand(r.InventoryFile, r.VaultPasswordFile)
}

func (r *runnablePlay) ToLocalCommand(o terraform.UIOutput, rpla runnablePlayLocalAnsibleArgs) (string, error) {
  baseCommand, err := r.ToCommand()
  if err != nil {
    return "", err
  }
  return fmt.Sprintf("%s %s", baseCommand, rpla.ToCommandArguments()), nil
}

type runnablePlayLocalAnsibleArgs struct {
  Username       string
  PemFile        string
  KnownHostsFile string
}

func (rpla *runnablePlayLocalAnsibleArgs) ToCommandArguments() string {
  args := fmt.Sprintf("--private-key='%s'", rpla.PemFile)
  args = fmt.Sprintf("%s --user='%s'", args, rpla.Username)
  args = fmt.Sprintf("%s --ssh-extra-args='-o UserKnownHostsFile=%s'", args, rpla.KnownHostsFile)
  return args
}
