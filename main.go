package main

import (
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/terraform"
	"github.com/radekg/terraform-provisioner-ansible/v2/provisioner"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProvisionerFunc: func() terraform.ResourceProvisioner {
			return provisioner.Provisioner()
		},
	})
}
