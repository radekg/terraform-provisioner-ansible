# Ansible provisioner for Terraform

[![Build Status](https://travis-ci.org/radekg/terraform-provisioner-ansible.svg?branch=master)](https://travis-ci.org/radekg/terraform-provisioner-ansible)

Ansible with Terraform - `remote` and `local` modes.

## General overview

The purpose of the provisioner is to provide an easy method for running Ansible to provision hosts created with Terraform.

This provisioner, however, is not designed to handle all possible responsibilities of Ansible. To better understand the distinction, consider what's possible and what's not possible with this provisioner.

### What's possible

- `local mode`
  - runs Ansible installed on the same machine where Terraform is executed
  - the provisioner will create a temporary inventory and execute Ansible only against the hosts created with Terraform `resource`
  - Ansible Vault password file can be used
  - the temporary inventory uses `ansible_connection=ssh`, the `ansible_host` is resolved from the `resource.connection` resource, it is possible to specify an `alias` using `hosts`
- `remote mode`
  - runs Ansible on the hosts created with Terraform `resource`
  - if Ansible is not installed on the newly created hosts, the provisioner can install one
  - the provisioner will create a temporary inventory and execute Ansible only against the hosts created with Terraform `resource`
  - playbooks, roles, vault password file and the temporary inventory file will be uploaded to the each host prior to Ansible run
  - hosts are provisioned using `ansible_connection=local`
  - an alias can be provided using `hosts`, each `host` will be included in every `group` provided with `groups` but each of them will use `ansible_connection=local`

### What's not possible

The provisioner by no means attempts to implement all Ansible use cases. The provisioner is not intended to be used as a `jump host`. For example, the `remote mode` does not allow provisioning hosts other than the one where Ansible is executed. The multitude of use cases Ansible covers is so wide that having to strive for full support is a huge undertaking.

If you find yourself in need of executing Ansible against well specified, complex inventories, it might be, indeed, easier to follow the regular process of provisoning hosts via Terraform and executing Ansible against them as a separate step.

## Installation

[Prebuilt releases are available on GitHub](https://github.com/radekg/terraform-provisioner-ansible/releases). Download a release for the version you require and place it in `~/.terraform.d/plugins` directory, as [documented here](https://www.terraform.io/docs/plugins/basics.html).

**Caution: you will need to rename the file to match the pattern recognized by Terraform: `terraform-provisioner-ansible_v<version>`.**

Alternatively, you can download and deploy an existing release using the following script:

    curl -sL https://raw.githubusercontent.com/radekg/terraform-provisioner-ansible/master/bin/deploy-release.sh \
      --output /tmp/deploy-release.sh
    chmod +x /tmp/deploy-release.sh
    /tmp/deploy-release.sh -v <version number>
    rm -rf /tmp/deploy-release.sh

## Configuration

Example:

```
resource "aws_instance" "test_box" {
  # ...
  connection {
    user = "centos"
  }
  provisioner "ansible" {
    plays {
      playbook = {
        file_path = "/path/to/playbook/file.yml"
        roles_path = ["/path1", "/path2"]
        force_handlers = false
        skip_tags = ["list", "of", "tags", "to", "skip"]
        start_at_task = "task-name"
        skip_tags = ["list", "of", "tags"]
      }
      # shared attributes
      enabled = true
      hosts = ["zookeeper"]
      groups = ["consensus"]
      become = true
      become_method = "sudo"
      become_user = "${aws_instance.test_box.connection.user}" # default behaviour
      diff = false
      extra_vars = {
        extra = {
          variables = {
            to = "pass"
          }
        }
      }
      forks = 5
      inventory_file = "/optional/inventory/file/path"
      limit = "limit"
      vault_password_file = "/vault/password/file/path"
      verbose = false
    }
    plays {
      module = {
        module = "module-name"
        args = {
          "arbitrary" = "arguments"
        }
        background = 0
        host_pattern = "string host pattern"
        one_line = false
        poll = 15
      }
      # shared attributes
      # enabled = ...
      # ...
    }
    defaults {
      hosts = ["eu-central-1"]
      groups = ["platform"]
      become_method = "sudo"
      become_user = "${aws_instance.test_box.connection.user}" # default behaviour
      extra_vars = {
        extra = {
          variables = {
            to = "pass"
          }
        }
      }
      forks = 5
      inventory_file = "/optional/inventory/file/path"
      limit = "limit"
      vault_password_file = "/vault/password/file/path"
    }
    ansible_ssh_settings {
      connect_timeout_seconds = 10
      connection_attempts = 10
      ssh_keyscan_timeout = 60
    }
    remote {
      use_sudo = true
      skip_install = false
      skip_cleanup = false
      install_version = ""
    }
  }
}
```

### Plays

#### Selecting what to run

Each `plays` may contain at most one `playbook` or `module`. Define multiple `plays` when more than one Ansible action shall be executed against a host.

#### Playbook attributes

- `plays.playbook.file_path`: full path to the playbook file
- `plays.playbook.force_handlers`: `ansible-playbook --force-handlers`, boolean, default `false`
- `plays.playbook.skip_tags`: `ansible-playbook --skip-tags`, list of strings, default `empty list` (not applied)
- `plays.playbook.start_at_task`: `ansible-playbook --start-at-task`, string, default `empty string` (not applied)
- `plays.playbook.tags`: `ansible-playbook --tags`, list of strings, default `empty list` (not applied)

#### Module attributes

- `plays.module.args`: `ansible --args`, map, default `empty map` (not applied)
- `plays.module.background`: `ansible --background`, int, default `0` (not applied)
- `plays.module.host_pattern`: `ansible <host-pattern>`, string, default `all`
- `plays.module.one_line`: `ansible --one-line`, boolean , default `false` (not applied)
- `plays.module.poll`: `ansible --poll`, int, default `15` (applied only when `background > 0`)

#### Plays attributes

- `plays.hosts`: list of hosts to include in auto-generated inventory file when `inventory_file` not given, `string list`, default `empty list`; more details below
- `plays.groups`: list of groups to include in auto-generated inventory file when `inventory_file` not given, `string list`, default `empty list`; more details below
- `plays.enabled`: boolean, default `true`; set to `false` to skip execution
- `plays.become`: `ansible-playbook --become`, boolean, default `false` (not applied)
- `plays.become_method`: `ansible-playbook --become-method`, string, default `sudo`, only takes effect when `become = true`
- `plays.become_user`: `ansible-playbook --become-user`, string, default `root`, only takes effect when `become = true`
- `plays.diff`: `ansible-playbook --diff`, boolean, default `false` (not applied)
- `plays.extra_vars`: `ansible-playbook --extra-vars`, map, default `empty map` (not applied); will be serialized to a json string
- `plays.forks`: `ansible-playbook --forks`, integer, default `5`
- `plays.inventory_file`: full path to an inventory file, `ansible-playbook --inventory-file`, string, default `empty string`; if `inventory_file` attribute is not given or empty, a temporary inventory using `hosts` and `groups` will be generated; when specified, `hosts` and `groups` are not in use
- `plays.limit`: `ansible-playbook --limit`, string, default `empty string` (not applied)
- `plays.vault_password_file`: `ansible-playbook --vault-password-file`, full path to the vault password file; file file will be uploaded to the server, string, default `empty string` (not applied)
- `plays.verbose`: `ansible-playbook --verbose`, boolean, default `false` (not applied)

#### Defaults

Some of the `plays` settings might be common along multiple `plays`. Such settings can be provided using the `defaults` attribute. Any setting from the following list can be specified in defaults:

- `defaults.hosts`
- `defaults.groups`
- `defaults.become_method`
- `defaults.become_user`
- `defaults.extra_vars`
- `defaults.forks`
- `defaults.inventory_file`
- `defaults.limit`
- `defaults.vault_password_file`

None of the boolean attributes can be specified in `defaults`. Neither `playbook` nor `module` can be specified in `defaults`.

#### Ansible SSH settings

Only used when `local provisioner` is used.

- `ansible_ssh_settings.connect_timeout_seconds`: SSH `ConnectTimeout`, default `10` seconds
- `ansible_ssh_settings.connect_timeout_seconds`: SSH `ConnectionAttempts`, default `10`
- `ansible_ssh_settings.ssh_keyscan_timeout`: when `ssh-keyscan` is used, how long to try fetching the host key until failing, default `60` seconds

#### Remote

The existence of this attribute enables `remote provisioning`. To use the defaults with remote provisioner, simply add `remote {}` to your provisioner.

- `use_sudo`: should `sudo` be used for bootstrap commands, boolean, default `true`, `become` does not make much sense; this attribute has no relevance to Ansible `--sudo` flag
- `skip_install`: if set to `true`, Ansible installation on the server will be skipped, assume Ansible is already installed, boolean, default `false`
- `skip_cleanup`: if set to `true`, Ansible bootstrap data will be left on the server after bootstrap, boolean, default `false`
- `install_version`: Ansible version to install when `skip_install = false`, string, default `empty string` (latest version available in respective repositories)

## Usage

### Local provisioner: SSH details

Local provisioner requires the `resource.connection` with, at least, the `user` defined. After the bootstrap, the plugin will inspect the connection info, check if the `user` and `private_key` are set and that provisioning succeeded, indeed, by checking the host (which should be an ip address of the newly created instance). If the connection info does not provide the SSH private key, `ssh agent` mode is assumed. When the state validates correctly, the provisioner will execute `ssh-keyscan` against the newly created instance and proceed only when `ssh-keyscan` succeedes. You will see plenty of `ssh-keyscan` errors in the output before provisioning starts.

In the process of doing so, a temporary inventory will be created for the newly created host, the pem file will be written to a temp file and a temporary `known_hosts` file will be created. Temporary `known_hosts` and temporary pem are per provisioner run, inventory is created for each `plays`. Files should be cleaned up after the provisioner finishes or fails. Inventory will be removed only if not supplied with `inventory_file`.

### Local provisioner: bastion host

If the `resource.connection` specifies a `bastion_host`, bastion host will be used. Bastion host must fulfill the same criteria as the host itself: `bastion_user` must be set, `bastion_private_key` may be set, if no `bastion_private_key` is specified, `ssh agent` is assumed.

Bastion host must:

- be a Linux / BSD based system
- have `mkdir`, `touch`, `ssh-keyscan`, `echo`, `cat` and `rm` commands available on the `$PATH` for the SSH `user`
- `$HOME` enviornment variable must be set for the SSH `user`

### Local provisioner: hosts and groups

The `plays.hosts` and `defaults.hosts` can be used with local provisioner. However, only the first defined host will be used when generating the inventory file. When `plays.hosts` or `defaults.hosts` is set to a non-empty list, the first host will be used to generate an inventory in the following format:

```
aFirstHost ansible_host=<ip address of the host> ansible_connection-ssh
```

For each group, additional ini section will be added, where each section is:

```
[groupName]
aFirstHost ansible_host=<ip address of the host> ansible_connection-ssh
```

For a host list `["someHost"]` and a group list of `["group1", "group2"]`, the inventory would be:

```
someHost ansible_host=<ip> ansible_connection-ssh

[group1]
someHost ansible_host=<ip> ansible_connection-ssh

[group2]
someHost ansible_host=<ip> ansible_connection-ssh
```

If `hosts` is an empty list or not given, the resulting generated inventory is:

```
<ip> ansible_connection-ssh

[group1]
<ip> ansible_connection-ssh

[group2]
<ip> ansible_connection-ssh
```

### Remote provisioner: running on hosts created by Terraform

Remote provisioner can be enabled by adding `remote {}` resource to the `provisioner` resource.

    resource "aws_instance" "ansible_test" {
      # ...
      connection {
        user = "centos"
        private_key = "${file("${path.module}/keys/centos.pem")}"
      }
      provisioner "ansible" {
        
        plays {
          # ...
        }
        
        remote {}
      }
    }

Unless `remote.skip_install = true`, the provisioner will install Ansible on the bootstrapped machine. Next, a temporary inventory file is created and uploaded to the host, playbooks and roles referenced by `plays.playbook.file_path` and `plays.playbook.roles_path` are uploaded, Ansible Vault password file is uploaded (unless no vault password file is given). Finally, Ansible will be executed.

Remote provisioning works with a Linux target host only.

## Changes from 1.0.0

- change `plays.playbook` and `plays.module` to a resource
- remove `yes/no`, boolean values are used instead
- **local provisioning becomes the default**, remote provisioning enabled with `remote {}` resource
- default values now provided using the `defaults` resource
- added `ansible_ssh_settings {}` resource
- `diff`, `become` and `verbose` can be set only on `plays`, no default override for boolean values

## Creating releases

To cut a release, run: 

    ./bin/release.sh

After the release is cut, build the binaries for the release:

    git checkout v${RELEASE_VERSION}
    ./bin/build-release-binaries.sh

After the binaries are built, upload the to GitHub release.