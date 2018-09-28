package types

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// Host represents Ansible inventory host.
type Host struct {
	fqdn              string
	ansibleConnection string
	ansiblePort       int
	ansibleUser       string
}

const (
	// attribute names:
	hostAttributeFqdn              = "fqdn"
	hostAttributeAnsibleConnection = "ansible_connection"
	hostAttributeAnsiblePort       = "ansible_port"
	hostAttributeAnsibleUser       = "ansible_user"
)

// NewHostSchema returns a new Host schema.
func NewHostSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				hostAttributeFqdn: &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
				hostAttributeAnsibleConnection: &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				hostAttributeAnsiblePort: &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},
				hostAttributeAnsibleUser: &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}

// NewHostFromInterface reads Host configuration from Terraform schema.
func NewHostFromInterface(i interface{}) *Host {
	vals := mapFromTypeSetList(i.(*schema.Set).List())
	return &Host{
		fqdn:              vals[hostAttributeFqdn].(string),
		ansibleConnection: vals[hostAttributeAnsibleConnection].(string),
		ansiblePort:       vals[hostAttributeAnsiblePort].(int),
		ansibleUser:       vals[hostAttributeAnsibleUser].(string),
	}
}

// FQDN returns host fqdn.
func (v *Host) FQDN() string {
	return v.fqdn
}

// AnsibleConnection returns host's ansible_connection.
func (v *Host) AnsibleConnection() string {
	return v.ansibleConnection
}

// AnsiblePort returns host's ansible_port.
func (v *Host) AnsiblePort() int {
	return v.ansiblePort
}

// AnsibleUser returns host's ansible_user.
func (v *Host) AnsibleUser() string {
	return v.ansibleUser
}
