# Ansible provisioner for terraform

**Work in progress**.

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
        playbook = "/full/path/to/an/ansible/playbook.yaml"
        hosts = ["${self.public_hostname}"]
        groups = ["leaders"]
      }
    }

## Arguments

**To be added**.