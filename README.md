# Ansible provisioner for Terraform

[![CircleCI](https://circleci.com/gh/radekg/terraform-provisioner-ansible.svg?style=svg)](https://circleci.com/gh/radekg/terraform-provisioner-ansible)

Ansible with Terraform 0.12.x - `remote` and `local` provisioners.

## General overview

The purpose of the provisioner is to provide an easy method for running Ansible to configure hosts created with Terraform.

This provisioner, however, is not designed to handle all possible Ansible use cases. Lets consider what's possible and what's not possible with this provisioner.

For after provisioning, you may find the following Ansible module useful if you use AWS S3 for state storage: [terraform-state-ansible-module](https://github.com/radekg/terraform-state-ansible-module).

### What's possible

- `compute resource local provisioner`
  - configured on a compute resource e.g. aws_instance, ibm_compute_vm_instance
  - runs Ansible installed on the same machine where Terraform is executed
  - the provisioner will create a temporary inventory and execute Ansible only against hosts created with Terraform `resource`
  - If `count` is used with the compute resource and is greater than 1, the provisioner runs after each resource instance is created, passing the host information for that instance only. 
  - Ansible Vault password file / Vault ID files can be used
  - the temporary inventory uses `ansible_connection=ssh`, the host `alias` is resolved from the `resource.connection` attribute, it is possible to specify an `ansible_host` using `plays.hosts`

- `compute resource remote provisioner`
  - configured on a compute resource e.g. aws_instance, ibm_compute_vm_instance
  - runs Ansible on the hosts created with Terraform `resource`
  - if Ansible is not installed on the newly created hosts, the provisioner can install one
  - the provisioner will create a temporary inventory and execute Ansible only against hosts created with Terraform `resource`
  - playbooks, roles, Vault password file / Vault ID files and the temporary inventory file will be uploaded to the each host prior to Ansible run
  - hosts are provisioned using `ansible_connection=local`
  - an alias can be provided using `hosts`, each `host` will be included in every `group` provided with `groups` but each of them will use `ansible_connection=local`

- `null_resource local provisioner`
  - configured on a null_resouce
  - runs Ansible installed on the same machine where Terraform is executed
  - Executes Ansible against the hosts defined by a list of IP addresses passed by interpolation on the `plays.hosts` attribute. The host group is defined by `plays.groups`. 
  - Executes the Ansible provisioner once against all hosts defined in `plays.hosts`, triggered by the availability of the interpolated vars.
  - Alternatively an inventory file (staticly defined or dynamically templated) can be passed to Ansible to specify a list of Terraform provisioned hosts and groups to be passed to Ansible to execute against in a single run.  
  - Inventory file can also be used with Ansible dynamic inventory and inventory plugins. 
  - The Terraform depends_on attribute can be used to determine when the Ansible provisioner is executed in relation to the provisioning of other Terraform resources
  - If the Terraform host is on the same network (cloud hosted or VPN) as the provisioned hosts, private IP addresses can be passed eliminating the requirement for bastion hosts or public SSH access. 
  - Ansible Vault password file / Vault ID files can be used


### What's not possible

The provisioner by no means attempts to implement all Ansible use cases. The provisioner is not intended to be used as a `jump host`. For example, the `remote mode` does not allow provisioning hosts other than the one where Ansible is executed. The number of use cases and possibilities covered by Ansible is so wide that having to strive for full support is a huge undertaking for one person. Using the provisioner with a null_resource provides further options for passing the Ansible inventory, including dynamic inventory, to meet use cases not addressed when used with a compute resource. 

If you find yourself in need of executing Ansible against well specified, complex inventories, either follow the regular process of provisoning hosts via Terraform and executing Ansible against them as a separate step, or initate the Ansible execution as the last Terraform task using null_resource and depends_on. Of course, pull requests are always welcomed!

## Installation

### Using Docker

```console
$ cd /my-terraform-project
$ docker run -it --rm -v $PWD:$PWD -w $PWD radekg/terraform-ansible:latest init
$ docker run -it --rm -v $PWD:$PWD -w $PWD radekg/terraform-ansible:latest apply
```

### Local Installation

Note that although `terraform-provisioner-ansible` is in the [terraform registry](https://registry.terraform.io/modules/radekg/ansible/provisioner/), it cannot be installed using a `module` terraform stanza, as such a configuration will not cause terraform to download the `terraform-provisioner-ansible` binary.

[Prebuilt releases are available on GitHub](https://github.com/radekg/terraform-provisioner-ansible/releases). Download a release for the version you require and place it in `~/.terraform.d/plugins` directory, as [documented here](https://www.terraform.io/docs/plugins/basics.html).

**Caution: you will need to rename the file to match the pattern recognized by Terraform: `terraform-provisioner-ansible_v<version>`.**

Alternatively, you can download and deploy an existing release using the following script:

    curl -sL \
      https://raw.githubusercontent.com/radekg/terraform-provisioner-ansible/master/bin/deploy-release.sh \
      --output /tmp/deploy-release.sh
    chmod +x /tmp/deploy-release.sh
    /tmp/deploy-release.sh -v <version number>
    rm -rf /tmp/deploy-release.sh

## Configuration

Example:

```tf
resource "aws_instance" "test_box" {
  # ...
  connection {
    host = "..."
    user = "centos"
  }
  provisioner "ansible" {
    plays {
      playbook {
        file_path = "/path/to/playbook/file.yml"
        roles_path = ["/path1", "/path2"]
        force_handlers = false
        skip_tags = ["list", "of", "tags", "to", "skip"]
        start_at_task = "task-name"
        tags = ["list", "of", "tags"]
      }
      # shared attributes
      enabled = true
      hosts = ["zookeeper"]
      groups = ["consensus"]
      become = false
      become_method = "sudo"
      become_user = "root"
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
      vault_id = ["/vault/password/file/path"]
      verbose = false
    }
    plays {
      module {
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
      become_user = "root"
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
      vault_id = ["/vault/password/file/path"]
    }
    ansible_ssh_settings {
      connect_timeout_seconds = 10
      connection_attempts = 10
      ssh_keyscan_timeout = 60
      insecure_no_strict_host_key_checking = false
      insecure_bastion_no_strict_host_key_checking = false
      user_known_hosts_file = ""
      bastion_user_known_hosts_file = ""
    }
    remote {
      use_sudo = true
      skip_install = false
      skip_cleanup = false
      install_version = ""
      local_installer_path = ""
      remote_installer_directory = "/tmp"
      bootstrap_directory = "/tmp"
    }
  }
}
```

```tf
resource "aws_instance" "test_box" {
  # ...
}

resource "null_resource" "test_box" {
  depends_on = "aws_instance.test_box"
  connection {
    host = "${aws_instance.test_box.0.public_ip}"
    private_key = "${file("./test_box")}"
  }
  provisioner "ansible" {
    plays {
      playbook {
        file_path = "/path/to/playbook/file.yml"
        roles_path = ["/path1", "/path2"]
        force_handlers = false
        skip_tags = ["list", "of", "tags", "to", "skip"]
        start_at_task = "task-name"
        tags = ["list", "of", "tags"]
      }
      hosts = ["aws_instance.test_box.*.public_ip"]
      groups = ["consensus"]
    }
  }
}
```

### Plays

#### Selecting what to run

Each `plays` must contain exactly one `playbook` or `module`. Define multiple `plays` when more than one Ansible action shall be executed against a host.

#### Playbook attributes

- `plays.playbook.file_path`: full path to the playbook YAML file; *remote provisioning*: a complete parent directory will be uploaded to the host
- `plays.playbook.roles_path`: `ansible-playbook --roles-path`, list of full paths to directories containing your roles; *remote provisioning*: all directories will be uploaded to the host; string list, default `empty list` (not applies)
- `plays.playbook.force_handlers`: `ansible-playbook --force-handlers`, boolean, default `false`
- `plays.playbook.skip_tags`: `ansible-playbook --skip-tags`, string list, default `empty list` (not applied)
- `plays.playbook.start_at_task`: `ansible-playbook --start-at-task`, string, default `empty string` (not applied)
- `plays.playbook.tags`: `ansible-playbook --tags`, string list, default `empty list` (not applied)

#### Module attributes

- `plays.module.args`: `ansible --args`, map, default `empty map` (not applied); values of type list and map will be converted to strings using `%+v`, avoid using those unless you really know what you are doing
- `plays.module.background`: `ansible --background`, int, default `0` (not applied)
- `plays.module.host_pattern`: `ansible <host-pattern>`, string, default `all`
- `plays.module.one_line`: `ansible --one-line`, boolean , default `false` (not applied)
- `plays.module.poll`: `ansible --poll`, int, default `15` (applied only when `background > 0`)

#### Plays attributes

- `plays.hosts`: list of hosts to include in auto-generated inventory file when `inventory_file` not given, string list, default `empty list`; When used with nulll_resource this can be an interpolated list of host IP address public or private; more details below
- `plays.groups`: list of groups to include in auto-generated inventory file when `inventory_file` not given, string list, default `empty list`; more details below
- `plays.enabled`: boolean, default `true`; set to `false` to skip execution
- `plays.become`: `ansible[-playbook] --become`, boolean, default `false` (not applied)
- `plays.become_method`: `ansible[-playbook] --become-method`, string, default `sudo`, only takes effect when `become = true`
- `plays.become_user`: `ansible[-playbook] --become-user`, string, default `root`, only takes effect when `become = true`
- `plays.diff`: `ansible[-playbook] --diff`, boolean, default `false` (not applied)
- `plays.extra_vars`: `ansible[-playbook] --extra-vars`, map, default `empty map` (not applied); will be serialized to a JSON string, supports values of different types, including lists and maps
- `plays.forks`: `ansible[-playbook] --forks`, int, default `5`
- `plays.inventory_file`: full path to an inventory file, `ansible[-playbook] --inventory-file`, string, default `empty string`; if `inventory_file` attribute is not given or empty, a temporary inventory using `hosts` and `groups` will be generated; when specified, `hosts` and `groups` are not in use
- `plays.limit`: `ansible[-playbook] --limit`, string, default `empty string` (not applied)
- `plays.vault_id`: `ansible[-playbook] --vault-id`, list of full paths to vault password files; *remote provisioning*: files will be uploaded to the server, string list, default `empty list` (not applied); takes precedence over `plays.vault_password_file`
- `plays.vault_password_file`: `ansible[-playbook] --vault-password-file`, full path to the vault password file; *remote provisioning*:  file will be uploaded to the server, string, default `empty string` (not applied)
- `plays.verbose`: `ansible[-playbook] --verbose`, boolean, default `false` (not applied)

#### Defaults

Some of the `plays` settings might be common across multiple `plays`. Such settings can be provided using the `defaults` attribute. Any setting from the following list can be specified in defaults:

- `defaults.hosts`
- `defaults.groups`
- `defaults.become_method`
- `defaults.become_user`
- `defaults.extra_vars`
- `defaults.forks`
- `defaults.inventory_file`
- `defaults.limit`
- `defaults.vault_id`
- `defaults.vault_password_file`

None of the boolean attributes can be specified in `defaults`. Neither `playbook` nor `module` can be specified in `defaults`.

#### Ansible SSH settings

- `ansible_ssh_settings.connect_timeout_seconds`: SSH `ConnectTimeout`, default `10` seconds
- `ansible_ssh_settings.connection_attempts`: SSH `ConnectionAttempts`, default `10`
- `ansible_ssh_settings.ssh_keyscan_timeout`: when `ssh-keyscan` is used, how long to try fetching the host key until failing, default `60` seconds

Following settings apply to `local provisioning` only:

- `ansible_ssh_settings.insecure_no_strict_host_key_checking`: if `true`, host key checking will be disabled when connecting to the target host, default `false`; when connecting via bastion, bastion will not execute any SSH keyscan
- `ansible_ssh_settings.insecure_bastion_no_strict_host_key_checking`: if `true`, host key checking will be disabled when connecting to the bastion host, default `false`
- `ansible_ssh_settings.user_known_hosts_file`: used only when `ansible_ssh_settings.insecure_no_strict_host_key_checking=false`; if set, the provided path will be used instead of an auto-generate known hosts file; when executing via bastion host, it allows the administrator to provide a known hosts file, no SSH keyscan will be executed on the bastion; default `empty string`
- `ansible_ssh_settings.bastion_user_known_hosts_file`: used only when `ansible_ssh_settings.insecure_bastion_no_strict_host_key_checking=false`; if set, the provided path will be used instead of an auto-generate known hosts file

#### Remote

The existence of this resource enables `remote provisioning`. To use remote provisioner with its default settings, simply add `remote {}` to your provisioner.

- `remote.use_sudo`: should `sudo` be used for bootstrap commands, boolean, default `true`, `become` does not make much sense; this attribute has no relevance to Ansible `--sudo` flag
- `remote.skip_install`: if set to `true`, Ansible installation on the server will be skipped, assume Ansible is already installed, boolean, default `false`
- `remote.skip_cleanup`: if set to `true`, Ansible bootstrap data will be left on the server after bootstrap, boolean, default `false`
- `remote.install_version`: Ansible version to install when `skip_install = false` and default installer is in ude, string, default `empty string` (latest version available in respective repositories)
- `remote.local_installer_path`: full path to the custom Ansible installer on the local machine, used when `skip_install = false`, string, default `empty string`; when empty and `skip_install = false`, the default installer is used
- `remote.remote_installer_directory`: full path to the remote directory where custom Ansible installer will be deployed to and executed from, used when `skip_install = false`, string, default `/tmp`; any intermediate directories will be created; the program will be executed with `sh`, use shebang if program requires a non-shell interpreter; the installer will be saved as `tf-ansible-installer` under the given directory; for `/tmp`, the path will be `/tmp/tf-ansible-installer`
- `remote.bootstrap_directory`: full path to the remote directory where playbooks, roles, password files and such will be uploaded to, used when `skip_install = false`, string, default `/tmp`; the final directory will have `tf-ansible-bootstrap` appended to it; for `/tmp`, the directory will be `/tmp/tf-ansible-bootstrap`

## Examples

[Working examples](https://github.com/radekg/terraform-provisioner-ansible/tree/master/examples).

## Usage

The provisioner does not support passwords. It is possible to add password support for:

- remote provisioner without bastion: host passwords reside in the inventory file
- remote provisioner with bastion: host passwords reside in the inventory file, bastion is handled by Terraform, password is never visible
- local provisioner without bastion: host passwords reside in the inventory file

However, local provisioner with bastion currently rely on executing an Ansible command with SSH `-o ProxyCommand`, this would require putting the password on the terminal. For consistency, consider no password support.

### Local provisioner: SSH details

Local provisioner requires the `resource.connection` with, at least, the `user` defined. After the bootstrap, the plugin will inspect the connection info, check if the `user` and `private_key` are set and that provisioning succeeded, indeed, by checking the host (which should be an ip address of the newly created instance). If the connection info does not provide the SSH private key, `ssh agent` mode is assumed.

In the process of doing so, a temporary inventory will be created for the newly created host, the pem file will be written to a temp file and a temporary `known_hosts` file will be created. Temporary `known_hosts` and temporary pem are per provisioner run, inventory is created for each `plays`. Files are cleaned up after the provisioner finishes or fails. Inventory will be removed only if not supplied with `inventory_file`.

### Local provisioner: host and bastion host keys

Because the provisioner executes SSH commands outside of itself, via Ansible command line tools, the provisioner must construct a temporary SSH `known_hosts` file to feed to Ansible. There are two possible scenarios.

#### Host without a bastion

1. If `connection.host_key` is used, the provisioner will use the provided host key to construct the temporary `known_hosts` file.
2. If `connection.host_key` is not given or empty, the provisioner will attempt a connection to the host and retrieve first host key returned during the handshake (similar to `ssh-keyscan` but using Golang SSH).

#### Host with bastion

This is a little bit more involved than the previous case.

1. If `connection.bastion_host_key` is provided, the provisioner will use the provided bastion host key for the `known_hosts` file.
2. If `connection.bastion_host_key` is not given or empty, the provisioner will attempt a connection to the bastion host and retrieve first host key returned during the handshake (similar to `ssh-keyscan` but using Golang SSH).

However, Ansible must know the host key of the target host where the bootstrap actually happens. If `connection.host_key` is provided, the provisioner will simply use the provieded value. But, if no `connection.host_key` is given (or empty), the provisioner will open an SSH connection to the bastion host and perform an `ssh-keyscan` operation against the target host on the bastion host.

In the `ssh-keyscan` case, the bastion host must:

- be a Linux / BSD based system
- **unless `bastion_host_key` is used**:
  - have `cat`, `echo`, `grep`, `mkdir`, `rm`, `ssh-keyscan` commands available on the `$PATH` for the SSH `user`
  - have `$HOME` enviornment variable set for the SSH `user`

### Compute resource local provisioner: hosts and groups

The `plays.hosts` and `defaults.hosts` attributes can be used with local provisioner. When used with a compute resource only the first defined host will be used when generating the inventory file and additional hosts will be ignored. If `plays.hosts` or `defaults.hosts` is not specified, the provisioner uses the public IP address of the Terraform provisioned resource instance. The inventory file is generated in the following format with a single host:

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

### Null_resource local provisioner: hosts and groups

The `plays.hosts` and `defaults.hosts` can be used with local provisioner on a null_resource. All passed hosts are used when generating the inventory file. The inventory file is generated in the following format:

```
<firstHost IP> 
<secondHost IP>
```

For each group, additional ini section will be added, where each section is:

```
[groupName]
<firstHost IP> 
<secondHost IP>
```

For a host list `["firstHost IP", "secondHost IP"]` and a group list of `["group1", "group2"]`, the inventory would be:

```
<firstHost IP> 
<secondHost IP>

[group1]
<firstHost IP> 
<secondHost IP>

[group2]
<firstHost IP> 
<secondHost IP>
```


### Remote provisioner: running on hosts created by Terraform

Remote provisioner can be enabled by adding `remote {}` resource to the `provisioner` resource.

```tf
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
    
    # enable remote provisioner
    remote {}
    
  }
}
```

Unless `remote.skip_install = true`, the provisioner will install Ansible on the bootstrapped machine. Next, a temporary inventory file is created and uploaded to the host, any playbooks, roles, Vault password files are uploaded to the host.

Remote provisioning works with a Linux target host only.

## Supported Ansible repository layouts

This provisioner supports two main repository layouts.

1. Roles nested under the playbook directory:
    
    ```
    .
    ├── install-tree.yml
    └── roles
        └── tree
            └── tasks
                └── main.yml
    ```

2. Roles and playbooks directories separate:

    ```
    .
    ├── playbooks
    │   └── install-tree.yml
    └── roles
        └── tree
            └── tasks
                └── main.yml
    ```
    
In the first case, to reference the roles, it is necessary to use `plays.playbook.roles_path` attribute:

```tf
    plays {
      playbook {
        file_path = ".../playbooks/install-tree.yml"
        roles_path = [
            ".../ansible-data/roles"
        ]
      }
    }
```

In the second case, it is sufficient to use only the `plays.playbook.file_path`, roles are nested, thus available to Ansible:

```tf
    plays {
      playbook {
        file_path = ".../playbooks/install-tree.yml"
      }
    }
```

### Remote provisioning directory upload

A remark regardng remote provisioning. Remote provisioner must upload referenced playbooks and role paths to the remote server. In case of a playbook, the complete parent directory of the YAML file will be uploaded. Remote provisioner attempts to deduplicate uploads, if multiple `plays` reference the same playbook, the playbook will be uploaded only once. This is achieved by generating an MD5 hash of the absolute path to the playbook's parent directory and storing your playbooks at `${remote.bootstrap_direcotry}/${md5-hash}` on the remote server.

For the roles path, the complete directory as referenced in `roles_path` will be uploaded to the remote server. Same deduplication method applies but the MD5 hash is the `roles_path` itself.

## Tests

Integration tests require `ansible` and `ansible-playbook` on the `$PATH`. To run tests:

```sh
make test-verbose
```

## Creating releases

To cut a release, run: 

    curl -sL https://raw.githubusercontent.com/radekg/git-release/master/git-release --output /tmp/git-release
    chmod +x /tmp/git-release
    /tmp/git-release --repository-path=$GOPATH/src/github.com/radekg/terraform-provisioner-ansible
    rm -rf /tmp/git-release

After the release is cut, build the binaries for the release:

    git checkout v${RELEASE_VERSION}
    ./bin/build-release-binaries.sh

Handle Docker image:

    git checkout v${RELEASE_VERSION}
    docker build --build-arg TAP_VERSION=$(cat .version) -t radekg/terraform-ansible:$(cat .version) .
    docker login --username=radekg
    docker tag radekg/terraform-ansible:$(cat .version) radekg/terraform-ansible:latest
    docker push radekg/terraform-ansible:$(cat .version)
    docker push radekg/terraform-ansible:latest

Note that the version is hardcoded in the [Dockerfile](Dockerfile). You may wish to update it after release.
