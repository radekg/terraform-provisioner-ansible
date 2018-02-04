# Ansible provisioner for terraform

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
          module_args {
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

### Plays

- `plays.playbook`: full path to the playbook yaml file; the complete directory containing the yaml file will be uploaded, string, no default
- `plays.module`: module to run, string, no default
- `plays.*`: any call argument, documented below, if not set but defined on the provisioner level, the value will fall back to the provisioner value

### Call arguments

- `hosts`: list of hosts to append to the inventory, each host will be decorated with `ansible_connection=local`, `localhost` is added automatically
- `groups`: list of groups to append to the inventory, each group will contain all hosts specified in `hosts`
- `tags`: `ansible-playbook --tags`, list of strings, default `empty list` (not applied)
- `skip_tags`: `ansible-playbook --skip-tags`, list of strings, default `empty list` (not applied)
- `start_at_task`: `ansible-playbook --start-at-task`, string, default `empty string` (not applied)
- `limit`: `ansible-playbook --limit`, string, default `empty string` (not applied)
- `forks`: `ansible-playbook --forks`, integer, default `5`
- `verbose`: `ansible-playbook --verbose`, string `yes/no`, default `empty string` (not applied)
- `one_line`: `ansible-playbook --one-line`, string `yes/no`, default `empty string` (not applied)
- `force_handlers`: `ansible-playbook --force-handlers`, string `yes/no`, default `empty string` (not applied)
- `extra_vars`: `ansible-playbook --extra-vars`, map, default `empty map` (not applied); will be serialized to a json string
- `module_args`: `ansible-playbook --args`, map, default `empty map` (not applied)
- `become`: `ansible-playbook --become`, string `yes/no`, default `empty string` (not applied)
- `become_user`: `ansible-playbook --become-user`, string, default `root`, only takes effect when `become = true`
- `become_method`: `ansible-playbook --become-method`, string, default `sudo`, only takes effect when `become = true`
- `vault_password_file`: `ansible-playbook --vault-password-file`, full path to the vault password file; file file will be uploaded to the server, string, default `empty string` (not applied)

### Provioner arguments

- `use_sudo`: should `sudo` be used for bootstrap commands, boolean, default `true`; when `true`, `become` does not make much sense
- `skip_install`: if set to `true`, ansible installation on the server will be skipped, assume ansible is already installed, boolean, default `false`
- `skip_cleanup`: if set to `true`, ansible bootstrap data will be left on the server after bootstrap, boolean, default `false`
- `install_version`: ansible version to install when `skip_install = false`, string, default `empty string` (latest available version)
