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
        "playbook":      "ansible/test.yaml",
        "force_handlers": "no",
        "skip_tags":      []string{"tag2"},
        "start_at_task":  "test task",
        "tags":           []string{"tag1", "tag2"},
      },
      map[string]interface{}{
        "module":       "some_module",
        "args":         map[string]interface{}{"ARG1": "value 1", "ARG2": "value 2"},
        "background":   10,
        "host_pattern": "all-tests",
        "one_line":     "no",
        "poll":         15,
      },
    },
    "use_sudo":           false,
    "skip_install":       true,
    "skip_cleanup":       true,
    "install_version":    "2.3.0.0",

    "hosts":              []string{"localhost1", "localhost2"},
    "groups":             []string{"group1", "group2"},

    "become":             "no",
    "become_method":      "sudo",
    "become_user":        "test",
    "extra_vars":         map[string]interface{}{"VAR1": "value 1", "VAR2": "value 2"},
    "forks":              10,
    "limit":              "a=b",
    "vault_password_file": "~/.vault_password_file",
    "verbose":            "no",
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
        "module":   "some_module",
      },
      map[string]interface{}{
        "module":   "some_module",
        "one_line": "unknown",
      },
    },
    "become":             "yes",
    "become_method":      "test",
  })

  warn, errs := Provisioner().Validate(c)
  if len(warn) > 0 {
    t.Fatalf("Warnings: %v", warn)
  }
  if len(errs) != 3 {
    t.Fatalf("Should have three errors but have: %v", errs)
  }
}

func TestResourceProvisioner_Validate_bad_playbook_config(t *testing.T) {
  // Errors:
  // - all 5 fields which can't be used with playbook
  c := testConfig(t, map[string]interface{}{
    "plays": []map[string]interface{}{
      map[string]interface{}{
        "playbook":     "ansible/test.yaml",
        "args":         map[string]interface{}{"arg1": "string value"},
        "background":   10,
        "host_pattern": "all",
        "one_line":     "yes",
        "poll":         15,
      },
    },
  })

  warn, errs := Provisioner().Validate(c)
  if len(warn) > 0 {
    t.Fatalf("Warnings: %v", warn)
  }
  if len(errs) != 5 {
    t.Fatalf("Should have five errors but have: %v", errs)
  }
}

func TestResourceProvisioner_Validate_bad_module_config(t *testing.T) {
  // Errors:
  // - all 4 fields which can't be used with module
  c := testConfig(t, map[string]interface{}{
    "plays": []map[string]interface{}{
      map[string]interface{}{
        "module":         "some-module",
        "force_handlers": "yes",
        "skip_tags":      []string{"tag1", "tag2"},
        "start_at_task":  "some task",
        "tags":           []string{"tag0"},
      },
    },
  })

  warn, errs := Provisioner().Validate(c)
  if len(warn) > 0 {
    t.Fatalf("Warnings: %v", warn)
  }
  if len(errs) != 4 {
    t.Fatalf("Should have four errors but have: %v", errs)
  }
}

func TestResourceProvisioner_Verify_fallbacks(t *testing.T) {

  expected_hosts :=             []string{"localhost1", "localhost2", "localhost"}
  expected_groups :=            []string{"group1", "group2"}
  expected_become :=            "yes"
  expected_becomeMethod :=      "su"
  expected_becomeUser :=        "unit_test_user"
  expected_extraVars :=         map[string]interface{}{"VAR1": "value 1", "VAR2": "value 2"}
  expected_forks :=             10
  expected_limit :=             "a=b"
  expected_vaultPasswordFile := "~/test/.vault_password_file"
  expected_verbose :=           "yes"

  c := map[string]interface{}{
    "plays": []map[string]interface{}{
      map[string]interface{}{
        "playbook":       "ansible/test.yaml",
        "force_handlers": "yes",
        "skip_tags":      []string{"tag2"},
        "start_at_task":  "some_test_task",
        "tags":           []string{"tag1"},
      },
      map[string]interface{}{
        "playbook":            "ansible/test.yaml",
        "force_handlers":      "yes",
        "skip_tags":           []string{"tag2"},
        "start_at_task":       "some_test_task",
        "tags":                []string{"tag1"},
        // fallback test:
        "hosts":               []string{"localhost3", "localhost"},
        "groups":              []string{"group3"},
        "become":              "no",
        "become_method":       "sudo",
        "become_user":         "root",
        "extra_vars":          map[string]interface{}{"VAR3": "value 1", "VAR4": "value 2"},
        "forks":               6,
        "limit":               "b=c",
        "vault_password_file": "~/another/.vault_password_file",
        "verbose":             "no",
      },
    },

    "hosts":               expected_hosts,
    "groups":              expected_groups,

    "become":              expected_become,
    "become_method":       expected_becomeMethod,
    "become_user":         expected_becomeUser,
    "extra_vars":          expected_extraVars,
    "forks":               expected_forks,
    "limit":               expected_limit,
    "vault_password_file": expected_vaultPasswordFile,
    "verbose":             expected_verbose,
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

  firstPlayInventory := p.Plays[0].InventoryMeta
  firstPlayArgs := p.Plays[0].CallArgs.Shared

  if strings.Join(firstPlayInventory.Hosts,"") != strings.Join(expected_hosts,"") {
    t.Fatalf("First play: expected 'hosts' %v but received %v.", expected_hosts, firstPlayInventory.Hosts)
  }
  if strings.Join(firstPlayInventory.Groups,"") != strings.Join(expected_groups,"") {
    t.Fatalf("First play: expected 'groups' %v but received %v.", expected_groups, firstPlayInventory.Groups)
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
  if mapToJson(firstPlayArgs.ExtraVars) != mapToJson(expected_extraVars) {
    t.Fatalf("First play: expected 'extra_vars' %v but received %v.", expected_extraVars, firstPlayArgs.ExtraVars)
  }
  if firstPlayArgs.Forks != expected_forks {
    t.Fatalf("First play: expected 'forks' %v but received %v.", expected_forks, firstPlayArgs.Forks)
  }
  if firstPlayArgs.Limit != expected_limit {
    t.Fatalf("First play: expected 'limit' %v but received %v.", expected_limit, firstPlayArgs.Limit)
  }
  if firstPlayArgs.VaultPasswordFile != expected_vaultPasswordFile {
    t.Fatalf("First play: expected 'vault_password_file' %v but received %v.", expected_vaultPasswordFile, firstPlayArgs.VaultPasswordFile)
  }
  if firstPlayArgs.Verbose != expected_verbose {
    t.Fatalf("First play: expected 'verbose' %v but received %v.", expected_verbose, firstPlayArgs.Verbose)
  }

  secondPlayInventory := p.Plays[1].InventoryMeta
  secondPlayArgs := p.Plays[1].CallArgs.Shared

  if strings.Join(secondPlayInventory.Hosts,"") == strings.Join(expected_hosts,"") {
    t.Fatalf("Second play: expected 'hosts' other than %v.", expected_hosts)
  }
  if strings.Join(secondPlayInventory.Groups,"") == strings.Join(expected_groups,"") {
    t.Fatalf("Second play: expected 'groups' other than %v.", expected_groups)
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
  if mapToJson(secondPlayArgs.ExtraVars) == mapToJson(expected_extraVars) {
    t.Fatalf("Second play: expected 'extra_vars' other than %v.", expected_extraVars)
  }
  if secondPlayArgs.Forks == expected_forks {
    t.Fatalf("Second play: expected 'forks' other than %v.", expected_forks)
  }
  if secondPlayArgs.Limit == expected_limit {
    t.Fatalf("Second play: expected 'limit' other than %v.", expected_limit)
  }
  if secondPlayArgs.VaultPasswordFile == expected_vaultPasswordFile {
    t.Fatalf("Second play: expected 'vault_password_file' other than %v.", expected_vaultPasswordFile)
  }
  if secondPlayArgs.Verbose == expected_verbose {
    t.Fatalf("Second play: expected 'verbose' other than %v.", expected_verbose)
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