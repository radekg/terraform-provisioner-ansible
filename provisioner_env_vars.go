package main

import (
	"os"
	"strconv"
)

func AnsibleSSHConnecTimeoutSeconds() int {
	sshConnectTimeout := 10
	if val, err := strconv.Atoi(os.Getenv("TF_PROVISIONER_ANSIBLE_SSH_CONNECT_TIMEOUT_SECONDS")); err == nil {
		sshConnectTimeout = val
	}
	return sshConnectTimeout
}

func AnsibleSSHConnecionAttempts() int {
	sshConnectionAttempts := 10
	if val, err := strconv.Atoi(os.Getenv("TF_PROVISIONER_ANSIBLE_SSH_CONNECTION_ATTEMPTS")); err == nil {
		sshConnectionAttempts = val
	}
	return sshConnectionAttempts
}

func SSHKeyScanTimeoutSeconds() int {
	sshKeyscanTimeoutSeconds := 60
	if val, err := strconv.Atoi(os.Getenv("TF_PROVISIONER_SSH_KEYSCAN_TIMEOUT_SECONDS")); err == nil {
		sshKeyscanTimeoutSeconds = val
	}
	return sshKeyscanTimeoutSeconds
}
