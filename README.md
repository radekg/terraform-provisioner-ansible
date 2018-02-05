# Ansible provisioner for terraform

[![Build Status](https://travis-ci.org/radekg/terraform-provisioner-ansible.svg?branch=master)](https://travis-ci.org/radekg/terraform-provisioner-ansible)

Run ansible on the provisioned host to bootstrap that host.

## Install

    mkdir -p $GOPATH/src/github.com/radekg
    cd $GOPATH/src/github.com/radekg
    git clone https://github.com/radekg/terraform-provisioner-ansible.git
    cd terraform-provisioner-ansible
    make install

    # to build for linux
    make build-linux
    # to build for darwin
    make build-darwin

The binary will be deployed to your `~/.terraform.d/plugins` directory so it is ready to use immediately.

## Usage

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

#### Shared arguments

These arguments can be set on the `provisioner` level or individual `plays`. When an argument is specified on the `provisioner` level and on `plays`, the `plays` value takes precedence.

- `become`: `ansible-playbook --become`, string `yes/no`, default `empty string` (not applied)
- `become_user`: `ansible-playbook --become-user`, string, default `root`, only takes effect when `become = true`
- `become_method`: `ansible-playbook --become-method`, string, default `sudo`, only takes effect when `become = true`
- `extra_vars`: `ansible-playbook --extra-vars`, map, default `empty map` (not applied); will be serialized to a json string
- `forks`: `ansible-playbook --forks`, integer, default `5`
- `limit`: `ansible-playbook --limit`, string, default `empty string` (not applied)
- `vault_password_file`: `ansible-playbook --vault-password-file`, full path to the vault password file; file file will be uploaded to the server, string, default `empty string` (not applied)
- `verbose`: `ansible-playbook --verbose`, string `yes/no`, default `empty string` (not applied)

### Provioner arguments

These affect provisioner only. Not related to `plays`.

- `use_sudo`: should `sudo` be used for bootstrap commands, boolean, default `true`; when `true`, `become` does not make much sense
- `skip_install`: if set to `true`, ansible installation on the server will be skipped, assume ansible is already installed, boolean, default `false`
- `skip_cleanup`: if set to `true`, ansible bootstrap data will be left on the server after bootstrap, boolean, default `false`
- `install_version`: ansible version to install when `skip_install = false`, string, default `empty string` (latest available version)
