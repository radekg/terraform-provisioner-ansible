package mode

import (
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

type targetHost struct {
	connInfo *connectionInfo
}

func newTargetHostFromConnectionInfo(connInfo *connectionInfo) *targetHost {
	return &targetHost{
		connInfo: connInfo,
	}
}

func (v *targetHost) agent() bool {
	return v.connInfo.Agent
}

func (v *targetHost) host() string {
	return v.connInfo.Host
}

func (v *targetHost) port() int {
	return v.connInfo.Port
}

func (v *targetHost) user() string {
	return v.connInfo.User
}

func (v *targetHost) pemFile() string {
	return v.connInfo.PrivateKey
}

func (v *targetHost) hostKey() string {
	return v.connInfo.HostKey
}

func (v *targetHost) timeout() time.Duration {
	return v.connInfo.TimeoutVal
}

func (v *targetHost) receiveHostKey(hostKey string) {
	v.connInfo.HostKey = hostKey
}

func (v *targetHost) fetchHostKey() error {
	configurator := &sshConfigurator{
		provider: v,
	}
	sshConfig, err := configurator.sshConfig()
	if err != nil {
		return err
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", v.host(), v.port()), sshConfig)
	defer client.Close()
	if err != nil {
		return err
	}
	return nil
}
