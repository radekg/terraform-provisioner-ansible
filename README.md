# Ansible provisioner for terraform

[![Build Status](https://travis-ci.org/radekg/terraform-provisioner-ansible.svg?branch=master)](https://travis-ci.org/radekg/terraform-provisioner-ansible)

Ansible with Terraform - `remote` and `local` modes.

## Installation

[Prebuilt releases are available on GitHub](https://github.com/radekg/terraform-provisioner-ansible/releases). Download a release for your version and place it in `~/.terraform.d/plugins` directory.

## Arguments

### Inventory meta

These are used only with remote provisioner and only when an explicit `inventory_file` isn't specified. Used to generate a runtime temporary inventory.

- `hosts`: list of hosts to append to the inventory, each host will be decorated with `ansible_connection=local`, `localhost` is added automatically
- `groups`: list of groups to append to the inventory, each group will contain all hosts specified in `hosts`

### Plays

#### Selecting what to run:

- `plays.playbook`: full path to the playbook yaml file; the complete directory containing the yaml file will be uploaded, string, no default
- `plays.module`: module to run, string, no default

#### Playbook arguments

- `plays.force_handlers`: `ansible-playbook --force-handlers`, string `yes/no`, default `empty string` (not applied)
- `plays.skip_tags`: `ansible-playbook --skip-tags`, list of strings, default `empty list` (not applied)
- `plays.start_at_task`: `ansible-playbook --start-at-task`, string, default `empty string` (not applied)
- `plays.tags`: `ansible-playbook --tags`, list of strings, default `empty list` (not applied)

#### Module arguments

- `plays.args`: `ansible --args`, map, default `empty map` (not applied)
- `plays.background`: `ansible --background`, int, default `0` (not applied)
- `plays.host_pattern`: `ansible <host-pattern>`, string, default `all`
- `plays.one_line`: `ansible --one-line`, string `yes/no`, default `empty string` (not applied)
- `plays.poll`: `ansible --poll`, int, default `15` (applied only when `background > 0`)

#### Disabling a play

It is possible that one may be testing a playbook or a module while the state of the changes in Ansible may not be known, thus - potentially - breaking the provisioning process. One might still need the provisioning process to succeed so the Ansible changes can be tested manually against the machine. In such case, instead of commenting the play out in Terraform file, use:

- `plays.enabled`: string `yes/no`, default `yes`; set to `no` to skip execution

#### Shared arguments

These arguments can be set on the `provisioner` level or individual `plays`. When an argument is specified on the `provisioner` level and on `plays`, the `plays` value takes precedence.

- `become`: `ansible-playbook --become`, string `yes/no`, default `empty string` (not applied)
- `become_user`: `ansible-playbook --become-user`, string, default `root`, only takes effect when `become = yes`
- `become_method`: `ansible-playbook --become-method`, string, default `sudo`, only takes effect when `become = yes`
- `extra_vars`: `ansible-playbook --extra-vars`, map, default `empty map` (not applied); will be serialized to a json string
- `forks`: `ansible-playbook --forks`, integer, default `5`
- `inventory_file`: full path to an inventory file, `ansible-playbook --inventory-file`, string, default `empty string` (not applied); when using in remote mode, if `inventory_file` argument is not specified, a temporary inventory using `hosts` and `groups` will be generated; when specified, `hosts` and `groups` are not in use
- `limit`: `ansible-playbook --limit`, string, default `empty string` (not applied)
- `vault_password_file`: `ansible-playbook --vault-password-file`, full path to the vault password file; file file will be uploaded to the server, string, default `empty string` (not applied)
- `verbose`: `ansible-playbook --verbose`, string `yes/no`, default `empty string` (not applied)

### Provioner arguments

These affect provisioner only. Not related to `plays`.

- `use_sudo`: should `sudo` be used for bootstrap commands, string `yes/no`, default `yes`, `become` does not make much sense
- `skip_install`: if set to `true`, Ansible installation on the server will be skipped, assume Ansible is already installed, string `yes/no`, default `no`
- `skip_cleanup`: if set to `true`, Ansible bootstrap data will be left on the server after bootstrap, string `yes/no`, default `no`
- `install_version`: Ansible version to install when `skip_install = false`, string, default `empty string` (latest available version)
- `local`: string `yes/no`, default `no`; if `yes`, Ansible will run on the host where Terraform command is executed; if `no`, Ansible will be installed on the bootstrapped host

## Usage

### Remote: running on a bootstrapped target

If `provisioner.local` is not set or `no` (the default), the provisioner will attempt a so-called `remote provisioning`. The provisioner will install Ansible on the bootstrapped machine, create a temporary inventory (if `inventory_file` not given), upload playbooks to the remote host and execute Ansible on the remote host.

    resource "aws_instance" "ansible_test" {
      ...
      connection {
        user = "centos"
        private_key = "${file("${path.module}/keys/centos.pem")}"
      }
      provisioner "ansible" {
        
        plays {
          playbook = "/full/path/to/an/ansible/playbook.yaml"
          hosts = ["override.example.com"]
          groups = ["override","groups"]
          extra_vars {
            override = "vars"
          }
        }
        
        plays {
          module = "some-module"
          hosts = ["override.example.com"]
          groups = ["override","groups"]
          extra_vars {
            override = "vars"
          }
          args {
            arg1 = "arg value"
          }
        }
        hosts = ["${self.public_hostname}"]
        groups = ["leaders"]
        extra_vars {
          var1 = "some value"
          var2 = 5
        }
      }
    }

**Remote provisioning works with a Linux target only.**

### Local: provisioning using local Ansible

If `provisioner.local = yes`, Ansible will be executed on the same host where Terraform is executed. However, currently only `connection` type `ssh` is supported and the assumption is that the `connection` uses a `private_key`. If you are not using private keys, provisioning will fail. SSH Agent is supported.

When using `provisioner.local = yes`, do not set any of these: `use_sudo`, `skip_install`, `skip_cleanup` or `install_version`.

`hosts` are not taken into account.

    resource "aws_instance" "ansible_test" {
      ...
      connection {
        user = "centos"
        private_key = "${file("${path.module}/keys/centos.pem")}"
      }
      
      provisioner "ansible" {
        
        plays {
          playbook = "/full/path/to/an/ansible/playbook.yaml"
          hosts = ["override.example.com"]
          groups = ["override","groups"]
          extra_vars {
            override = "vars"
          }
        }
        
        become = "yes"
        local = "yes"
        
      }
    }

#### Local mode SSH: details

The local mode requires the provisioner connection to use, at least, the username. After the bootstrap, the plugin will inspect the connection info, check that the username and private key are set and that provisioning succeeded, indeed, by checking the host (which should be an ip address of the newly created instance). If the connection info does not provide the SSH private key, `ssh agent` mode is assumed. When the state validates correctly, the provisioner will execute `ssh-keyscan` against the newly created instance and proceed only when `ssh-keyscan` succeedes. You will see plenty of `ssh-keyscan` errors in the output before provisioning starts.

In the process of doing so, a temporary inventory will be created for the newly created host, the pem file will be written to a temp file and a temporary `known_hosts` file will be created. Temporary `known_hosts` and temporary pem are per provisioner run, inventory is created for each `plays`. Files should be cleaned up after the provisioner finishes or fails. Inventory will be removed only if not supplied with `inventory_file`.

##### `ssh-keyscan` timeout

`ssh-keyscan` can take some time until the host key is resolved. The provisioner will continue trying obtaining the key for the period `60 seconds` by default. The value can be changed using `TF_PROVISIONER_SSH_KEYSCAN_TIMEOUT_SECONDS` environment variable.

##### Local mode bastion host

Bastion host must fulfill the following criteria:

- Linux / BSD based system
- `mkdir`, `touch`, `ssh-keyscan`, `echo`, `cat` and `rm` commands must be available to the SSH user
- `$HOME` enviornment variable must be set for the SSH user

###### SSH connect timeout and connection attempts

The local mode specifies a number of `ssh extra options` to be passed to Ansible. Most of them are implied from the run context, however, `ConnectTimeout` and `ConnectionAttempts` aren't available via `ansible` / `ansible-playbook`. These can be set using environment variables. Details:

- `ConnectTimeout`: default `10` seconds, change by setting `TF_PROVISIONER_ANSIBLE_SSH_CONNECT_TIMEOUT_SECONDS` environment variable
- `ConnectionAttempts`: default `10`, change by setting `TF_PROVISIONER_ANSIBLE_SSH_CONNECTION_ATTEMPTS` environment variable

## yes/no? Why not boolean?

The `yes/no` exists because of the fallback mechanism for `become` and `verbose`, other arguments use `yes/no` for consistency. With boolean values, there is no easy way to specify `undefined` state.

## Creating releases

To cut a release, run: 

    ./bin/release.sh

After the release is cut, build the binaries for the release:

    git checkout v${RELEASE_VERSION}
    ./bin/build-release-binaries.sh

After the binaries are built, upload the to GitHub release.