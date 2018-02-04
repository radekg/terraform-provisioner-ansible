package main

import (
  "fmt"
  "testing"

  "github.com/hashicorp/terraform/helper/schema"
)

func TestResourceProvider_configuration(t *testing.T) {

  testConfig := map[string]interface{}{
    "plays": make(map[string]interface{}),
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
      "VAR1": "value 1",
      "VAR2": "value 2",
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
    schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, testConfig),
  )

  if err != nil {
    t.Fatalf("Error: %v", err)
  }

  fmt.Printf(fmt.Sprintf(" ================> %+v", p))

}