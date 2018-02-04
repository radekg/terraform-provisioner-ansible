package main

import (
  "encoding/json"
  "strings"
  "testing"

  "github.com/hashicorp/terraform/config"
  "github.com/hashicorp/terraform/helper/schema"
  "github.com/hashicorp/terraform/terraform"
)

func TestResourceProvisioner_impl(t *testing.T) {
  var _ terraform.ResourceProvisioner = Provisioner()
}

func TestProvisioner(t *testing.T) {
  if err := Provisioner().(*schema.Provisioner).InternalValidate(); err != nil {
    t.Fatalf("error: %s", err)
  }
}

func TestResourceProvisioner_Validate_good_config(t *testing.T) {
  c := testConfig(t, map[string]interface{}{
    "plays": []map[string]interface{}{
      map[string]interface{}{
        "playbook": "ansible/test.yaml",
      },
      map[string]interface{}{
        "module": "some_module",
      },
    },
    "use_sudo":           false,
    "skip_install":       true,
    "skip_cleanup":       true,
    "install_version":    "2.3.0.0",
    "hosts":              []string{"localhost1", "localhost2"},
    "groups":             []string{"group1", "group2"},
    "tags":               []string{"tag1", "tag2"},
    "skip_tags":          []string{"tag2"},
    "start_at_task":      "test task",
    "limit":              "a=b",
    "forks":              10,
    "extra_vars":         map[string]interface{}{"VAR1": "value 1", "VAR2": "value 2"},
    "module_args":        map[string]interface{}{"ARG1": "value 1", "ARG2": "value 2"},
    "verbose":            "no",
    "force_handlers":     "no",
    "one_line":           "no",
    "become":             "no",
    "become_method":      "sudo",
    "become_user":        "test",
    "vault_password_file": "~/.vault_password_file",
  })

  warn, errs := Provisioner().Validate(c)
  if len(warn) > 0 {
    t.Fatalf("Warnings: %v", warn)
  }
  if len(errs) > 0 {
    t.Fatalf("Errors: %v", errs)
  }

}

func TestResourceProvisioner_Validate_config_without_plays(t *testing.T) {
  // no plays gives a warning:
  c := testConfig(t, map[string]interface{}{
    "use_sudo": false,
  })

  warn, errs := Provisioner().Validate(c)
  if len(warn) != 1 {
    t.Fatalf("Should have one warning.")
  }
  if len(errs) > 0 {
    t.Fatalf("Errors: %v", errs)
  }
}

func TestResourceProvisioner_Validate_bad_config(t *testing.T) {
  // Errors:
  // - plays contains both playbook and module
  // - become_method is not one of the supported methods
  // - one_line invalid value
  c := testConfig(t, map[string]interface{}{
    "plays": []map[string]interface{}{
      map[string]interface{}{
        "playbook": "ansible/test.yaml",
        "module": "some_module",
      },
    },
    "use_sudo":           false,
    "skip_install":       true,
    "skip_cleanup":       true,
    "install_version":    "2.3.0.0",
    "hosts":              []string{"localhost1", "localhost2"},
    "groups":             []string{"group1", "group2"},
    "tags":               []string{"tag1", "tag2"},
    "skip_tags":          []string{"tag2"},
    "start_at_task":      "test task",
    "limit":              "a=b",
    "forks":              10,
    "extra_vars":         map[string]interface{}{"VAR1": "value 1", "VAR2": "value 2"},
    "module_args":        map[string]interface{}{"ARG1": "value 1", "ARG2": "value 2"},
    "verbose":            "no",
    "force_handlers":     "no",
    "one_line":           "unknown",
    "become":             "yes",
    "become_method":      "test",
    "become_user":        "test",
    "vault_password_file": "~/.vault_password_file",
  })

  warn, errs := Provisioner().Validate(c)
  if len(warn) > 0 {
    t.Fatalf("Warnings: %v", warn)
  }
  if len(errs) != 3 {
    t.Fatalf("Should have three errors but have: %v", errs)
  }
}

func TestResourceProvisioner_Verify_fallbacks(t *testing.T) {

  expected_hosts :=             []string{"localhost1", "localhost2", "localhost"}
  expected_groups :=            []string{"group1", "group2"}
  expected_tags :=              []string{"tag1"}
  expected_skipTags :=           []string{"tag2"}
  expected_startAtTask :=       "some_test_task"
  expected_limit :=             "a=b"
  expected_forks :=             10
  expected_extraVars :=         map[string]interface{}{"VAR1": "value 1", "VAR2": "value 2"}
  expected_moduleArgs :=        map[string]interface{}{"ARG1": "value 1", "ARG2": "value 2"}
  expected_verbose :=           "yes"
  expected_forceHandlers :=     "yes"
  expected_oneLine :=           "yes"
  expected_become :=            "yes"
  expected_becomeMethod :=      "su"
  expected_becomeUser :=        "unit_test_user"
  expected_vaultPasswordFile := "~/test/.vault_password_file"

  c := map[string]interface{}{
    "plays": []map[string]interface{}{
      map[string]interface{}{
        "playbook": "ansible/test.yaml",
      },
      map[string]interface{}{
        "playbook": "ansible/test.yaml",
        "hosts":               []string{"localhost3", "localhost"},
        "groups":              []string{"group3"},
        "tags":                []string{"tag3"},
        "skip_tags":           []string{"tag4"},
        "start_at_task":       "another_task",
        "limit":               "b=c",
        "forks":               6,
        "extra_vars":          map[string]interface{}{"VAR3": "value 1", "VAR4": "value 2"},
        "module_args":         map[string]interface{}{"ARG3": "value 1", "ARG4": "value 2"},
        "verbose":             "no",
        "force_handlers":      "no",
        "one_line":            "no",
        "become":              "no",
        "become_method":       "sudo",
        "become_user":         "root",
        "vault_password_file": "~/another/.vault_password_file",
      },
    },
    "use_sudo":            false,
    "skip_install":        true,
    "skip_cleanup":        true,
    "install_version":     "2.3.0.0",
    "hosts":               expected_hosts,
    "groups":              expected_groups,
    "tags":                expected_tags,
    "skip_tags":           expected_skipTags,
    "start_at_task":       expected_startAtTask,
    "limit":               expected_limit,
    "forks":               expected_forks,
    "extra_vars":          expected_extraVars,
    "module_args":         expected_moduleArgs,
    "verbose":             expected_verbose,
    "force_handlers":      expected_forceHandlers,
    "one_line":            expected_oneLine,
    "become":              expected_become,
    "become_method":       expected_becomeMethod,
    "become_user":         expected_becomeUser,
    "vault_password_file": expected_vaultPasswordFile,
  }

  p, err := decodeConfig(
      schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, c),
  )
  if err != nil {
    t.Fatalf("Error: %v", err)
  }

  if len(p.Plays) != 2 {
    t.Fatalf("Must have 2 plays.")
  }

  firstPlayArgs := p.Plays[0].Callable.CallArgs

  if strings.Join(firstPlayArgs.Hosts,"") != strings.Join(expected_hosts,"") {
    t.Fatalf("First play: expected 'hosts' %v but received %v.", expected_hosts, firstPlayArgs.Hosts)
  }
  if strings.Join(firstPlayArgs.Groups,"") != strings.Join(expected_groups,"") {
    t.Fatalf("First play: expected 'groups' %v but received %v.", expected_groups, firstPlayArgs.Groups)
  }
  if strings.Join(firstPlayArgs.Tags,"") != strings.Join(expected_tags,"") {
    t.Fatalf("First play: expected 'tags' %v but received %v.", expected_tags, firstPlayArgs.Tags)
  }
  if strings.Join(firstPlayArgs.SkipTags,"") != strings.Join(expected_skipTags,"") {
    t.Fatalf("First play: expected 'skip_tags' %v but received %v.", expected_skipTags, firstPlayArgs.SkipTags)
  }
  if mapToJson(firstPlayArgs.ModuleArgs) != mapToJson(expected_moduleArgs) {
    t.Fatalf("First play: expected 'module_args' %v but received %v.", expected_moduleArgs, firstPlayArgs.ModuleArgs)
  }
  if mapToJson(firstPlayArgs.ExtraVars) != mapToJson(expected_extraVars) {
    t.Fatalf("First play: expected 'extra_vars' %v but received %v.", expected_extraVars, firstPlayArgs.ExtraVars)
  }
  if firstPlayArgs.StartAtTask != expected_startAtTask {
    t.Fatalf("First play: expected 'start_at_task' %v but received %v.", expected_startAtTask, firstPlayArgs.StartAtTask)
  }
  if firstPlayArgs.Limit != expected_limit {
    t.Fatalf("First play: expected 'limit' %v but received %v.", expected_limit, firstPlayArgs.Limit)
  }
  if firstPlayArgs.Forks != expected_forks {
    t.Fatalf("First play: expected 'forks' %v but received %v.", expected_forks, firstPlayArgs.Forks)
  }
  if firstPlayArgs.Verbose != expected_verbose {
    t.Fatalf("First play: expected 'verbose' %v but received %v.", expected_verbose, firstPlayArgs.Verbose)
  }
  if firstPlayArgs.ForceHandlers != expected_forceHandlers {
    t.Fatalf("First play: expected 'force_handlers' %v but received %v.", expected_forceHandlers, firstPlayArgs.ForceHandlers)
  }
  if firstPlayArgs.OneLine != expected_oneLine {
    t.Fatalf("First play: expected 'one_line' %v but received %v.", expected_oneLine, firstPlayArgs.OneLine)
  }
  if firstPlayArgs.Become != expected_become {
    t.Fatalf("First play: expected 'become' %v but received %v.", expected_become, firstPlayArgs.Become)
  }
  if firstPlayArgs.BecomeMethod != expected_becomeMethod {
    t.Fatalf("First play: expected 'become_method' %v but received %v.", expected_becomeMethod, firstPlayArgs.BecomeMethod)
  }
  if firstPlayArgs.BecomeUser != expected_becomeUser {
    t.Fatalf("First play: expected 'become_user' %v but received %v.", expected_becomeUser, firstPlayArgs.BecomeUser)
  }
  if firstPlayArgs.VaultPasswordFile != expected_vaultPasswordFile {
    t.Fatalf("First play: expected 'vault_password_file' %v but received %v.", expected_vaultPasswordFile, firstPlayArgs.VaultPasswordFile)
  }

  secondPlayArgs := p.Plays[1].Callable.CallArgs

  if strings.Join(secondPlayArgs.Hosts,"") == strings.Join(expected_hosts,"") {
    t.Fatalf("Second play: expected 'hosts' other than %v.", expected_hosts)
  }
  if strings.Join(secondPlayArgs.Groups,"") == strings.Join(expected_groups,"") {
    t.Fatalf("Second play: expected 'groups' other than %v.", expected_groups)
  }
  if strings.Join(secondPlayArgs.Tags,"") == strings.Join(expected_tags,"") {
    t.Fatalf("Second play: expected 'tags' other than %v.", expected_tags)
  }
  if strings.Join(secondPlayArgs.SkipTags,"") == strings.Join(expected_skipTags,"") {
    t.Fatalf("Second play: expected 'skip_tags' other than %v.", expected_skipTags)
  }
  if mapToJson(secondPlayArgs.ModuleArgs) == mapToJson(expected_moduleArgs) {
    t.Fatalf("Second play: expected 'module_args' other than %v.", expected_moduleArgs)
  }
  if mapToJson(secondPlayArgs.ExtraVars) == mapToJson(expected_extraVars) {
    t.Fatalf("Second play: expected 'extra_vars' other than %v.", expected_extraVars)
  }
  if secondPlayArgs.StartAtTask == expected_startAtTask {
    t.Fatalf("Second play: expected 'start_at_task' other than %v.", expected_startAtTask)
  }
  if secondPlayArgs.Limit == expected_limit {
    t.Fatalf("Second play: expected 'limit' other than %v.", expected_limit)
  }
  if secondPlayArgs.Forks == expected_forks {
    t.Fatalf("Second play: expected 'forks' other than %v.", expected_forks)
  }
  if secondPlayArgs.Verbose == expected_verbose {
    t.Fatalf("Second play: expected 'verbose' other than %v.", expected_verbose)
  }
  if secondPlayArgs.ForceHandlers == expected_forceHandlers {
    t.Fatalf("Second play: expected 'force_handlers' other than %v.", expected_forceHandlers)
  }
  if secondPlayArgs.OneLine == expected_oneLine {
    t.Fatalf("Second play: expected 'one_line' other than %v.", expected_oneLine)
  }
  if secondPlayArgs.Become == expected_become {
    t.Fatalf("Second play: expected 'become' other than %v.", expected_become)
  }
  if secondPlayArgs.BecomeMethod == expected_becomeMethod {
    t.Fatalf("Second play: expected 'become_method' other than %v.", expected_becomeMethod)
  }
  if secondPlayArgs.BecomeUser == expected_becomeUser {
    t.Fatalf("Second play: expected 'become_user' other than %v.", expected_becomeUser)
  }
  if secondPlayArgs.VaultPasswordFile == expected_vaultPasswordFile {
    t.Fatalf("Second play: expected 'vault_password_file' other than %v.", expected_vaultPasswordFile)
  }
}

func mapToJson(m map[string]interface{}) string {
  str, err := json.Marshal(m)
  if err != nil {
    return ""
  }
  return string(str)
}

func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
  r, err := config.NewRawConfig(c)
  if err != nil {
    t.Fatalf("config error: %s", err)
  }
  return terraform.NewResourceConfig(r)
}