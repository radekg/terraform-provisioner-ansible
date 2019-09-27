provider "aws" {
  region  = "eu-central-1"
  profile = "terraform-provisioner-ansible"
}

variable "ami_id" {}


## -- security groups:

resource "aws_security_group" "ssh_box" {
  name        = "ssh_box"
  description = "SSH"

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    self        = true
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

## -- machine:

resource "aws_instance" "test_box" {
  ami           = "${var.ami_id}"
  count         = "1"
  instance_type = "m3.medium"

  security_groups = ["${aws_security_group.ssh_box.name}"]

  connection {
    host = "${self.public_ip}"
    user = "centos"
  }

  provisioner "ansible" {
    plays {
      galaxy_install {
        role_file = "${path.module}/ansible-data/requirements.yml"
        verbose = true
      }
    }
    plays {
      playbook {
        file_path = "${path.module}/ansible-data/playbooks/install-ntp.yml"
        roles_path = [
            # our galaxy_install does not deine roles_path, default values are being used:
            "galaxy_install:/tmp/tf-ansible-bootstrap/galaxy-roles"
        ]
      }
      hosts = ["testBoxToBootstrap"]
    }
    remote {}
  }

  root_block_device {
    delete_on_termination = true
    volume_size           = 8
    volume_type           = "gp2"
  }
}
