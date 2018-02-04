package main

import (
  "fmt"
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
  c := map[string]interface{}{
    "plays": []map[string]interface{}{
      map[string]interface{}{
        "playbook": "ansible/test.yaml",
      },
    },
    "use_sudo":    false,
    "skip_install": true,
    "skip_cleanup": true,
    "install_version": "2.3.0.0",
    "hosts": []string{"localhost1", "localhost2"},
    "group": []string{"group1", "group2"},
    "tags": []string{"tag1", "tag2"},
    "skip_tags": []string{"tag2"},
    "start_at_task": "test task",
    "limit": "a=b",
    "forks": 10,
    "extra_vars": map[string]interface{}{
      "VAR1": "value 1",
      "VAR2": "value 2",
    },
    "module_args": map[string]interface{}{
      "ARG1": "value 1",
      "ARG2": "value 2",
    },
    "verbose": false,
    "force_handlers": false,
    "one_line": false,
    "become": true,
    "become_method": "test",
    "become_user": "test",
    "vault_password_file": "~/.vault_password_file",
  }

  p, err := decodeConfig(
      schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, c),
  )
  if err != nil {
    t.Fatalf("Error: %v", err)
  }

  fmt.Println(fmt.Sprintf(" =======================> %+v", p))
}

func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
  r, err := config.NewRawConfig(c)
  if err != nil {
    t.Fatalf("config error: %s", err)
  }
  return terraform.NewResourceConfig(r)
}