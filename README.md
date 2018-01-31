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
