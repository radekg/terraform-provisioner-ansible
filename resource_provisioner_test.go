package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var vaultPasswordFile string
var alternativeVaultPasswordFile string
var playbookFile string

func TestMain(m *testing.M) {

	tempVaultPasswordFile, _ := ioutil.TempFile("", "vault-password-file")
	tempAlternativeVaultPasswordFile, _ := ioutil.TempFile("", "vault-password-file")
	tempPlaybookFile, _ := ioutil.TempFile("", "playbook-file")

	vaultPasswordFile = tempVaultPasswordFile.Name()
	alternativeVaultPasswordFile = tempAlternativeVaultPasswordFile.Name()
	playbookFile = tempPlaybookFile.Name()

	result := m.Run()

	os.Remove(vaultPasswordFile)
	os.Remove(alternativeVaultPasswordFile)
	os.Remove(playbookFile)

	os.Exit(result)
}

func TestResourceProvisioner_impl(t *testing.T) {
	var _ terraform.ResourceProvisioner = Provisioner()
}

func TestProvisioner(t *testing.T) {
	if err := Provisioner().(*schema.Provisioner).InternalValidate(); err != nil {
		t.Fatalf("error: %s", err)
	}
}

func TestBadConfig(t *testing.T) {
	// play.0.playbook with no file_path
	// play.0.module with no module
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"playbook": []map[string]interface{}{
					map[string]interface{}{},
				},
			},
			map[string]interface{}{
				"module": []map[string]interface{}{
					map[string]interface{}{},
				},
			},
		},

		"remote": []map[string]interface{}{
			map[string]interface{}{
				"use_sudo":        false,
				"skip_install":    true,
				"skip_cleanup":    true,
				"install_version": "2.3.0.0",
			},
		},

		"defaults": []map[string]interface{}{
			map[string]interface{}{
				"hosts": []interface{}{},
			},
		},
	})
	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) != 2 {
		t.Fatalf("Expected 2 errors but got: %v", errs)
	}
}

func TestGoodAndCompleteRemoteConfig(t *testing.T) {
	// warnings:
	// = plays.0.playbook.roles_path
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"playbook": []map[string]interface{}{
					map[string]interface{}{
						"file_path":      playbookFile,
						"roles_path":     []interface{}{"${path.module}/path/to/a/role/directory"},
						"force_handlers": false,
						"skip_tags":      []string{"tag2"},
						"start_at_task":  "test task",
						"tags":           []string{"tag1", "tag2"},
					},
				},
			},
			map[string]interface{}{
				"module": []map[string]interface{}{
					map[string]interface{}{
						"module":       "some_module",
						"args":         map[string]interface{}{"ARG1": "value 1", "ARG2": "value 2"},
						"background":   10,
						"host_pattern": "all-tests",
						"one_line":     false,
						"poll":         15,
					},
				},
			},
		},

		"remote": []map[string]interface{}{
			map[string]interface{}{
				"use_sudo":        false,
				"skip_install":    true,
				"skip_cleanup":    true,
				"install_version": "2.3.0.0",
			},
		},

		"defaults": []map[string]interface{}{
			map[string]interface{}{
				"hosts":               []interface{}{"localhost"},
				"groups":              []interface{}{"group1", "group2"},
				"become_method":       "sudo",
				"become_user":         "test",
				"extra_vars":          map[string]interface{}{"VAR1": "value 1", "VAR2": "value 2"},
				"forks":               10,
				"limit":               "a=b",
				"vault_password_file": vaultPasswordFile,
			},
		},

		"ansible_ssh_settings": []map[string]interface{}{
			map[string]interface{}{
				"connect_timeout_seconds": 5,
				"connection_attempts":     5,
				"ssh_keyscan_timeout":     30,
			},
		},
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) != 1 {
		t.Fatalf("Expected one warning.")
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestGoodLocalConfigWithoutPlaybookWarnings(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"playbook": []map[string]interface{}{
					map[string]interface{}{
						"file_path": playbookFile,
					},
				},
			},
		},
	})
	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", errs)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestRequirePlaybookFilePath(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"playbook": []map[string]interface{}{
					map[string]interface{}{},
				},
			},
		},
	})
	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", errs)
	}
	if len(errs) != 1 {
		t.Fatalf("Expected 1 error.")
	}
}

func TestRequireModuleName(t *testing.T) {
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"module": []map[string]interface{}{
					map[string]interface{}{},
				},
			},
		},
	})
	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", errs)
	}
	if len(errs) != 1 {
		t.Fatalf("Expected 1 error.")
	}
}

func TestConfigWithoutPlaysFails(t *testing.T) {
	// no plays gives a warning:
	c := testConfig(t, map[string]interface{}{})

	warn, errs := Provisioner().Validate(c)
	if len(warn) != 1 {
		t.Fatalf("Should have 1 warning.")
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %v", errs)
	}
}

func TestConfigWithPlaysbookAndModuleFails(t *testing.T) {
	// no plays gives a warning:
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"playbook": []map[string]interface{}{
					map[string]interface{}{
						"file_path": playbookFile,
					},
				},
				"module": []map[string]interface{}{
					map[string]interface{}{
						"module": "module-name",
					},
				},
			},
		},
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) != 1 {
		t.Fatalf("Should have 1 warning.")
	}
	if len(errs) != 1 {
		t.Fatalf("Should have 1 error.")
	}
}

func TestConfigWithInvalidValueTypeFailes(t *testing.T) {
	// file_path is set to a boolean instead of a string
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"playbook": []map[string]interface{}{
					map[string]interface{}{
						"file_path": true,
					},
				},
			},
		},
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) > 0 {
		t.Fatalf("Warnings: %v", warn)
	}
	if len(errs) != 1 {
		t.Fatalf("Expected one error but received: %v", errs)
	}
}

func TestConfigProvisionerParserDecoder(t *testing.T) {
	c := map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"playbook": []map[string]interface{}{
					map[string]interface{}{
						"file_path": playbookFile,
					},
				},
				"hosts": []interface{}{"host.to.play"},
			},
			map[string]interface{}{
				"module": []map[string]interface{}{
					map[string]interface{}{
						"module": "some-module",
					},
				},
			},
		},

		"remote": []map[string]interface{}{
			map[string]interface{}{
				"use_sudo":        false,
				"skip_install":    true,
				"skip_cleanup":    true,
				"install_version": "2.3.0.0",
			},
		},

		"defaults": []map[string]interface{}{
			map[string]interface{}{
				"hosts":               []interface{}{"localhost"},
				"groups":              []interface{}{"group1", "group2"},
				"become_method":       "sudo",
				"become_user":         "test",
				"extra_vars":          map[string]interface{}{"VAR1": "value 1", "VAR2": "value 2"},
				"forks":               10,
				"limit":               "a=b",
				"vault_password_file": vaultPasswordFile,
			},
		},

		"ansible_ssh_settings": []map[string]interface{}{
			map[string]interface{}{
				"connect_timeout_seconds": 5,
				"connection_attempts":     5,
				"ssh_keyscan_timeout":     30,
			},
		},
	}

	warn, errs := Provisioner().Validate(testConfig(t, c))
	if len(warn) > 0 {
		t.Fatalf("Warnings: %+v", warn)
	}
	if len(errs) > 0 {
		t.Fatalf("Errors: %+v", errs)
	}

	/*p, _ := */
	decodeConfig(
		schema.TestResourceDataRaw(t, Provisioner().(*schema.Provisioner).Schema, c),
	)

	//t.Fatalf(" =================> %+v", p)
}

/*

func TestResourceProvisioner_Validate_bad_config(t *testing.T) {
	// Errors:
	// - plays contains both playbook and module
	// - become_method is not one of the supported methods
	// - one_line invalid value
	// Warnings:
	// - nothing to play
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"playbook": playbookFile,
				"module":   "some_module",
			},
			map[string]interface{}{
				"module":   "some_module",
				"one_line": "unknown",
			},
		},
		"become":        "yes",
		"become_method": "test",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) != 1 {
		t.Fatalf("Should have one warning but have: %v", warn)
	}
	if len(errs) != 3 {
		t.Fatalf("Should have three errors but have: %v", errs)
	}
}

func TestResourceProvisioner_Validate_bad_playbook_config(t *testing.T) {
	// Errors:
	// - all 5 fields which can't be used with playbook
	// - enabled not yes/no
	// Warnings:
	// - nothing to play
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"enabled":      "invalid value",
				"playbook":     playbookFile,
				"args":         map[string]interface{}{"arg1": "string value"},
				"background":   10,
				"host_pattern": "all",
				"one_line":     "yes",
				"poll":         15,
			},
		},
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) != 1 {
		t.Fatalf("Should have one warning but have: %v", warn)
	}
	if len(errs) != 6 {
		t.Fatalf("Should have six errors but have: %v", errs)
	}
}

func TestResourceProvisioner_Validate_bad_module_config(t *testing.T) {
	// Errors:
	// - all 4 fields which can't be used with module
	// Warnings:
	// - nothing to play
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
	if len(warn) != 1 {
		t.Fatalf("Should have one warning but have: %v", warn)
	}
	if len(errs) != 4 {
		t.Fatalf("Should have four errors but have: %v", errs)
	}
}

func TestResourceProvisioner_Validate_file_existence_checks(t *testing.T) {
	// Errors:
	// - all 3 files do not exist
	// Warnings:
	// - nothing to play
	c := testConfig(t, map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"playbook": "/tmp/non-existing-playbook.yaml",
			},
		},
		"inventory_file":      "/tmp/non-existing-inventory-file",
		"vault_password_file": "/tmp/non-existing-vault-password-file",
	})

	warn, errs := Provisioner().Validate(c)
	if len(warn) != 1 {
		t.Fatalf("Should have one warning but have: %v", warn)
	}
	if len(errs) != 3 {
		t.Fatalf("Should have three errors but have: %v", errs)
	}
}

func TestResourceProvisioner_Verify_fallbacks(t *testing.T) {

	expectedHosts := []string{"localhost1", "localhost2", "localhost"}
	expectedGroups := []string{"group1", "group2"}
	expectedBecome := "yes"
	expectedBecomeMethod := "su"
	expectedBecomeUser := "unit_test_user"
	expectedExtraVars := map[string]interface{}{"VAR1": "value 1", "VAR2": "value 2"}
	expectedForks := 10
	expectedLimit := "a=b"
	expectedVaultPasswordFile := vaultPasswordFile
	expectedVerbose := "yes"

	c := map[string]interface{}{
		"plays": []map[string]interface{}{
			map[string]interface{}{
				"enabled":        "yes",
				"playbook":       playbookFile,
				"force_handlers": "yes",
				"skip_tags":      []string{"tag2"},
				"start_at_task":  "some_test_task",
				"tags":           []string{"tag1"},
			},
			map[string]interface{}{
				"enabled":        "no",
				"playbook":       playbookFile,
				"force_handlers": "yes",
				"skip_tags":      []string{"tag2"},
				"start_at_task":  "some_test_task",
				"tags":           []string{"tag1"},
				// fallback test:
				"hosts":               []string{"localhost3", "localhost"},
				"groups":              []string{"group3"},
				"become":              "no",
				"become_method":       "sudo",
				"become_user":         "root",
				"extra_vars":          map[string]interface{}{"VAR3": "value 1", "VAR4": "value 2"},
				"forks":               6,
				"limit":               "b=c",
				"vault_password_file": alternativeVaultPasswordFile,
				"verbose":             "no",
			},
		},

		"hosts":  expectedHosts,
		"groups": expectedGroups,

		"become":              expectedBecome,
		"become_method":       expectedBecomeMethod,
		"become_user":         expectedBecomeUser,
		"extra_vars":          expectedExtraVars,
		"forks":               expectedForks,
		"limit":               expectedLimit,
		"vault_password_file": expectedVaultPasswordFile,
		"verbose":             expectedVerbose,
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

	if p.Plays[0].Enabled != "yes" {
		t.Fatalf("First play: enabled should be yes")
	}
	if p.Plays[1].Enabled != "no" {
		t.Fatalf("Second play: enabled should be no")
	}

	if strings.Join(firstPlayInventory.Hosts, "") != strings.Join(expectedHosts, "") {
		t.Fatalf("First play: expected 'hosts' %v but received %v.", expectedHosts, firstPlayInventory.Hosts)
	}
	if strings.Join(firstPlayInventory.Groups, "") != strings.Join(expectedGroups, "") {
		t.Fatalf("First play: expected 'groups' %v but received %v.", expectedGroups, firstPlayInventory.Groups)
	}
	if firstPlayArgs.Become != expectedBecome {
		t.Fatalf("First play: expected 'become' %v but received %v.", expectedBecome, firstPlayArgs.Become)
	}
	if firstPlayArgs.BecomeMethod != expectedBecomeMethod {
		t.Fatalf("First play: expected 'become_method' %v but received %v.", expectedBecomeMethod, firstPlayArgs.BecomeMethod)
	}
	if firstPlayArgs.BecomeUser != expectedBecomeUser {
		t.Fatalf("First play: expected 'become_user' %v but received %v.", expectedBecomeUser, firstPlayArgs.BecomeUser)
	}
	if mapToJSON(firstPlayArgs.ExtraVars) != mapToJSON(expectedExtraVars) {
		t.Fatalf("First play: expected 'extra_vars' %v but received %v.", expectedExtraVars, firstPlayArgs.ExtraVars)
	}
	if firstPlayArgs.Forks != expectedForks {
		t.Fatalf("First play: expected 'forks' %v but received %v.", expectedForks, firstPlayArgs.Forks)
	}
	if firstPlayArgs.Limit != expectedLimit {
		t.Fatalf("First play: expected 'limit' %v but received %v.", expectedLimit, firstPlayArgs.Limit)
	}
	if firstPlayArgs.VaultPasswordFile != expectedVaultPasswordFile {
		t.Fatalf("First play: expected 'vault_password_file' %v but received %v.", expectedVaultPasswordFile, firstPlayArgs.VaultPasswordFile)
	}
	if firstPlayArgs.Verbose != expectedVerbose {
		t.Fatalf("First play: expected 'verbose' %v but received %v.", expectedVerbose, firstPlayArgs.Verbose)
	}

	secondPlayInventory := p.Plays[1].InventoryMeta
	secondPlayArgs := p.Plays[1].CallArgs.Shared

	if strings.Join(secondPlayInventory.Hosts, "") == strings.Join(expectedHosts, "") {
		t.Fatalf("Second play: expected 'hosts' other than %v.", expectedHosts)
	}
	if strings.Join(secondPlayInventory.Groups, "") == strings.Join(expectedGroups, "") {
		t.Fatalf("Second play: expected 'groups' other than %v.", expectedGroups)
	}
	if secondPlayArgs.Become == expectedBecome {
		t.Fatalf("Second play: expected 'become' other than %v.", expectedBecome)
	}
	if secondPlayArgs.BecomeMethod == expectedBecomeMethod {
		t.Fatalf("Second play: expected 'become_method' other than %v.", expectedBecomeMethod)
	}
	if secondPlayArgs.BecomeUser == expectedBecomeUser {
		t.Fatalf("Second play: expected 'become_user' other than %v.", expectedBecomeUser)
	}
	if mapToJSON(secondPlayArgs.ExtraVars) == mapToJSON(expectedExtraVars) {
		t.Fatalf("Second play: expected 'extra_vars' other than %v.", expectedExtraVars)
	}
	if secondPlayArgs.Forks == expectedForks {
		t.Fatalf("Second play: expected 'forks' other than %v.", expectedForks)
	}
	if secondPlayArgs.Limit == expectedLimit {
		t.Fatalf("Second play: expected 'limit' other than %v.", expectedLimit)
	}
	if secondPlayArgs.VaultPasswordFile == expectedVaultPasswordFile {
		t.Fatalf("Second play: expected 'vault_password_file' other than %v.", expectedVaultPasswordFile)
	}
	if secondPlayArgs.Verbose == expectedVerbose {
		t.Fatalf("Second play: expected 'verbose' other than %v.", expectedVerbose)
	}
}

func mapToJSON(m map[string]interface{}) string {
	str, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(str)
}
*/
func testConfig(t *testing.T, c map[string]interface{}) *terraform.ResourceConfig {
	r, err := config.NewRawConfig(c)
	if err != nil {
		t.Fatalf("config error: %s", err)
	}
	return terraform.NewResourceConfig(r)
}
