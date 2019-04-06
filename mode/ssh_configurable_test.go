package mode

import (
	"testing"
	"time"

	"github.com/radekg/terraform-provisioner-ansible/test"
)

type testingSSHConfigurable struct {
	hostKeyVaule string
}

func (p *testingSSHConfigurable) agent() bool {
	return true
}
func (p *testingSSHConfigurable) host() string {
	return "127.0.0.1"
}
func (p *testingSSHConfigurable) port() int {
	return 2022
}
func (p *testingSSHConfigurable) user() string {
	return "test-user"
}
func (p *testingSSHConfigurable) pemFile() string {
	return test.TestSSHHostKeyPrivate
}
func (p *testingSSHConfigurable) hostKey() string {
	return p.hostKeyVaule
}
func (p *testingSSHConfigurable) timeout() time.Duration {
	return time.Duration(time.Second * 1)
}
func (p *testingSSHConfigurable) receiveHostKey(hk string) {
	p.hostKeyVaule = hk
}

func TestSSHConfigurable(t *testing.T) {
	configurator := sshConfigurator{
		provider: &testingSSHConfigurable{
			hostKeyVaule: test.TestSSHHostKeyPublic,
		},
	}
	_, err := configurator.sshConfig()
	if err != nil {
		t.Fatal("Expected SSH config but received an error", err)
	}
}
