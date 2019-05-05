package types

import (
	"os"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
)

// AnsibleSSHSettings represents Ansible process SSH settings.
type AnsibleSSHSettings struct {
	connectTimeoutSeconds                  int
	connectAttempts                        int
	sshKeyscanSeconds                      int
	insecureNoStrictHostKeyChecking        bool
	insecureBastionNoStrictHostKeyChecking bool
	userKnownHostsFile                     string
	bastionUserKnownHostsFile              string
	overrideStrictHostKeyChecking          bool

}

const (
	// default values:
	ansibleSSHDefaultConnectTimeoutSeconds = 10
	ansibleSSHDefaultConnectAttempts       = 10
	ansibleSSHDefaultSSHKeyscanSeconds     = 60
	// attribute names:
	ansibleSSHAttributeConnectTimeoutSeconds                  = "connect_timeout_seconds"
	ansibleSSHAttributeConnectAttempts                        = "connection_attempts"
	ansibleSSHAttributeSSHKeyscanSeconds                      = "ssh_keyscan_timeout"
	ansibleSSHAttributeInsecureNoStrictHostKeyChecking        = "insecure_no_strict_host_key_checking"
	ansibleSSHAttributeInsecureBastionNoStrictHostKeyChecking = "insecure_bastion_no_strict_host_key_checking"
	ansibleSSHAttributeUserKnownHostsFile                     = "user_known_hosts_file"
	ansibleSSHAttributeBastionUserKnownHostsFile              = "bastion_user_known_hosts_file"
	// environment variable names:
	ansibleSSHEnvConnectTimeoutSeconds = "TF_PROVISIONER_ANSIBLE_SSH_CONNECT_TIMEOUT_SECONDS"
	ansibleSSHEnvConnectAttempts       = "TF_PROVISIONER_ANSIBLE_SSH_CONNECTION_ATTEMPTS"
	ansibleSSHEnvSSHKeyscanSeconds     = "TF_PROVISIONER_SSH_KEYSCAN_TIMEOUT_SECONDS"
)

// NewAnsibleSSHSettingsSchema returns a new AnsibleSSHSettings schema.
func NewAnsibleSSHSettingsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				ansibleSSHAttributeConnectTimeoutSeconds: &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
					DefaultFunc: func() (interface{}, error) {
						if val, err := strconv.Atoi(os.Getenv(ansibleSSHEnvConnectTimeoutSeconds)); err == nil {
							return val, nil
						}
						return ansibleSSHDefaultConnectTimeoutSeconds, nil
					},
				},
				ansibleSSHAttributeConnectAttempts: &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
					DefaultFunc: func() (interface{}, error) {
						if val, err := strconv.Atoi(os.Getenv(ansibleSSHEnvConnectAttempts)); err == nil {
							return val, nil
						}
						return ansibleSSHDefaultConnectAttempts, nil
					},
				},
				ansibleSSHAttributeSSHKeyscanSeconds: &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
					DefaultFunc: func() (interface{}, error) {
						if val, err := strconv.Atoi(os.Getenv(ansibleSSHEnvSSHKeyscanSeconds)); err == nil {
							return val, nil
						}
						return ansibleSSHDefaultSSHKeyscanSeconds, nil
					},
				},
				ansibleSSHAttributeInsecureNoStrictHostKeyChecking: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
				ansibleSSHAttributeInsecureBastionNoStrictHostKeyChecking: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
				ansibleSSHAttributeUserKnownHostsFile: &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  "",
				},
				ansibleSSHAttributeBastionUserKnownHostsFile: &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  "",
				},
			},
		},
	}
}

// NewAnsibleSSHSettingsFromInterface reads AnsibleSSHSettings configuration from Terraform schema.
func NewAnsibleSSHSettingsFromInterface(i interface{}, ok bool) *AnsibleSSHSettings {
	v := &AnsibleSSHSettings{
		connectTimeoutSeconds: ansibleSSHDefaultConnectTimeoutSeconds,
		connectAttempts:       ansibleSSHDefaultConnectAttempts,
		sshKeyscanSeconds:     ansibleSSHDefaultSSHKeyscanSeconds,
	}
	if ok {
		vals := mapFromTypeSetList(i.(*schema.Set).List())
		v.connectTimeoutSeconds = vals[ansibleSSHAttributeConnectTimeoutSeconds].(int)
		v.connectAttempts = vals[ansibleSSHAttributeConnectAttempts].(int)
		v.sshKeyscanSeconds = vals[ansibleSSHAttributeSSHKeyscanSeconds].(int)
		v.insecureNoStrictHostKeyChecking = vals[ansibleSSHAttributeInsecureNoStrictHostKeyChecking].(bool)
		v.insecureBastionNoStrictHostKeyChecking = vals[ansibleSSHAttributeInsecureBastionNoStrictHostKeyChecking].(bool)
		v.userKnownHostsFile = vals[ansibleSSHAttributeUserKnownHostsFile].(string)
		v.bastionUserKnownHostsFile = vals[ansibleSSHAttributeBastionUserKnownHostsFile].(string)
	}
	return v
}

// ConnectTimeoutSeconds reutrn Ansible process SSH connection timeout.
func (v *AnsibleSSHSettings) ConnectTimeoutSeconds() int {
	return v.connectTimeoutSeconds
}

// ConnectAttempts reutrn Ansible process SSH connection attempt count.
func (v *AnsibleSSHSettings) ConnectAttempts() int {
	return v.connectAttempts
}

// SSHKeyscanSeconds reutrn Ansible process SSH keyscan timeout.
func (v *AnsibleSSHSettings) SSHKeyscanSeconds() int {
	return v.sshKeyscanSeconds
}

// InsecureNoStrictHostKeyChecking if true, SSH to the target host uses -o StrictHostKeyChecking=no.
func (v *AnsibleSSHSettings) InsecureNoStrictHostKeyChecking() bool {
	if v.overrideStrictHostKeyChecking || v.insecureNoStrictHostKeyChecking {
		return true
	} else {
		return false
	}
}

// SetOverrideStrictHostKeyChecking is used by the provisioner when attached to a null_resource
func (v *AnsibleSSHSettings) SetOverrideStrictHostKeyChecking() {
	v.overrideStrictHostKeyChecking  = true
}


// InsecureBastionNoStrictHostKeyChecking if true, SSH to the bastion host uses -o StrictHostKeyChecking=no.
func (v *AnsibleSSHSettings) InsecureBastionNoStrictHostKeyChecking() bool {
	return v.insecureBastionNoStrictHostKeyChecking
}

// UserKnownHostsFile returns a path to the user known hosts file for the target host.
func (v *AnsibleSSHSettings) UserKnownHostsFile() string {
	return v.userKnownHostsFile
}

// BastionUserKnownHostsFile returns a path to the user known hosts file for the bastion host.
func (v *AnsibleSSHSettings) BastionUserKnownHostsFile() string {
	return v.bastionUserKnownHostsFile
}
