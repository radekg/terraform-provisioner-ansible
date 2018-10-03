package mode

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

type bastionHost struct {
	connInfo *connectionInfo
}

// NewBastionHostFromConnectionInfo returns a bastion host instance extracted from connection info.
func newBastionHostFromConnectionInfo(connInfo *connectionInfo) *bastionHost {
	return &bastionHost{
		connInfo: connInfo,
	}
}

func (v *bastionHost) agent() bool {
	return v.connInfo.Agent
}

func (v *bastionHost) inUse() bool {
	return v.connInfo.BastionHost != ""
}

func (v *bastionHost) host() string {
	return v.connInfo.BastionHost
}

func (v *bastionHost) port() int {
	return v.connInfo.BastionPort
}

func (v *bastionHost) user() string {
	return v.connInfo.BastionUser
}

func (v *bastionHost) pemFile() string {
	return v.connInfo.BastionPrivateKey
}

func (v *bastionHost) hostKey() string {
	return v.connInfo.BastionHostKey
}

func (v *bastionHost) timeout() time.Duration {
	return v.connInfo.TimeoutVal
}

func (v *bastionHost) connect() (*ssh.Client, error) {
	sshConfig, err := v.sshConfig()
	if err != nil {
		return nil, err
	}
	return ssh.Dial("tcp", fmt.Sprintf("%s:%d", v.host(), v.port()), sshConfig)
}

func (v *bastionHost) sshConfig() (*ssh.ClientConfig, error) {
	authMethods := make([]ssh.AuthMethod, 0)
	if v.pemFile() != "" {
		authMethods = append(authMethods, v.publicKeyFile())
	}
	if v.agent() {
		authMethods = append(authMethods, v.sshAgent())
	}

	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		v.connInfo.BastionHostKey = string(ssh.MarshalAuthorizedKey(key))
		return nil
	}

	if v.hostKey() != "" {
		// from terraform/communicator/ssh/provisioner.go
		// ----------------------------------------------
		// The knownhosts package only takes paths to files, but terraform
		// generally wants to handle config data in-memory. Rather than making
		// the known_hosts file an exception, write out the data to a temporary
		// file to create the HostKeyCallback.
		tf, err := ioutil.TempFile("", "tf-provisioner-known_hosts")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp known_hosts file: %s", err)
		}
		defer tf.Close()
		defer os.RemoveAll(tf.Name())

		// we mark this as a CA as well, but the host key fallback will still
		// use it as a direct match if the remote host doesn't return a
		// certificate.
		if _, err := tf.WriteString(fmt.Sprintf("@cert-authority %s %s\n", v.host(), v.hostKey())); err != nil {
			return nil, fmt.Errorf("failed to write temp known_hosts file: %s", err)
		}
		tf.Sync()

		hostKeyCallback, err = knownhosts.New(tf.Name())
		if err != nil {
			return nil, err
		}
	}

	return &ssh.ClientConfig{
		User:            v.user(),
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         v.timeout(),
	}, nil
}

func (v *bastionHost) sshAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

func (v *bastionHost) publicKeyFile() ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(v.pemFile())
	if err != nil {
		return nil
	}
	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}
