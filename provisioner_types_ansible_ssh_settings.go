package main

import (
	"os"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
)

type ansibleSSHSettings struct {
	connectTimeoutSeconds int
	connectAttempts       int
	sshKeyscanSeconds     int
}

const (
	// default values:
	ansibleSSHDefaultConnectTimeoutSeconds = 10
	ansibleSSHDefaultConnectAttempts       = 10
	ansibleSSHDefaultSSHKeyscanSeconds     = 60
	// attribute names:
	ansibleSSHAttributeConnectTimeoutSeconds = "connect_timeout_seconds"
	ansibleSSHAttributeConnectAttempts       = "connection_attempts"
	ansibleSSHAttributeSSHKeyscanSeconds     = "ssh_keyscan_timeout"
	// environment variable names:
	ansibleSSHEnvConnectTimeoutSeconds = "TF_PROVISIONER_ANSIBLE_SSH_CONNECT_TIMEOUT_SECONDS"
	ansibleSSHEnvConnectAttempts       = "TF_PROVISIONER_ANSIBLE_SSH_CONNECTION_ATTEMPTS"
	ansibleSSHEnvSSHKeyscanSeconds     = "TF_PROVISIONER_SSH_KEYSCAN_TIMEOUT_SECONDS"
)

func newAnsibleSSHSettingsSchema() *schema.Schema {
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
			},
		},
	}
}

func newAnsibleSSHSettingsFromInterface(i interface{}, ok bool) *ansibleSSHSettings {
	v := &ansibleSSHSettings{
		connectTimeoutSeconds: ansibleSSHDefaultConnectTimeoutSeconds,
		connectAttempts:       ansibleSSHDefaultConnectAttempts,
		sshKeyscanSeconds:     ansibleSSHDefaultSSHKeyscanSeconds,
	}
	if ok {
		vals := mapFromSetList(i.(*schema.Set).List())
		if val, ok := vals[ansibleSSHAttributeConnectTimeoutSeconds]; ok {
			v.connectTimeoutSeconds = val.(int)
		}
		if val, ok := vals[ansibleSSHAttributeConnectAttempts]; ok {
			v.connectAttempts = val.(int)
		}
		if val, ok := vals[ansibleSSHAttributeSSHKeyscanSeconds]; ok {
			v.sshKeyscanSeconds = val.(int)
		}
	}
	return v
}
