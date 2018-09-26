package mode

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/radekg/terraform-provisioner-ansible/types"

	"github.com/hashicorp/terraform/terraform"
	linereader "github.com/mitchellh/go-linereader"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

const (
	bastionHostKnownHostsFile = "~/.ssh/known_hosts"
)

type cleanup func()

// BastionKeyScan holds the ssh-keyscan metadata.
type BastionKeyScan struct {
	host           string
	port           int
	username       string
	privateKeyFile string
}

// NewBastionKeyScan create an ssh-keyscan operation wrapper.
func NewBastionKeyScan(host string, port int, username string, privateKeyFile string) *BastionKeyScan {
	return &BastionKeyScan{
		host:           host,
		port:           port,
		username:       username,
		privateKeyFile: privateKeyFile,
	}
}

func (b *BastionKeyScan) publicKeyFile() ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(b.privateKeyFile)
	if err != nil {
		return nil
	}
	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

func (b *BastionKeyScan) sshAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

func (b *BastionKeyScan) sshModes() ssh.TerminalModes {
	return ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
}

func (b *BastionKeyScan) sshConfig() *ssh.ClientConfig {
	authMethods := make([]ssh.AuthMethod, 0)
	if b.privateKeyFile == "" {
		authMethods = append(authMethods, b.sshAgent())
	} else {
		authMethods = append(authMethods, b.publicKeyFile())
	}

	return &ssh.ClientConfig{
		User: b.username,
		Auth: authMethods,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
}

func (b *BastionKeyScan) makeError(pattern string, e error) error {
	if e == nil {
		return fmt.Errorf("Bastion ssh-keyscan: %s", pattern)
	}
	return fmt.Errorf("Bastion ssh-keyscan: %s", fmt.Sprintf(pattern, e))
}

func (b *BastionKeyScan) output(o terraform.UIOutput, message string) {
	o.Output(fmt.Sprintf("Bastion host: %s", message))
}

func (b *BastionKeyScan) copyOutput(o terraform.UIOutput, r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}

func (b *BastionKeyScan) redirectOutputs(o terraform.UIOutput, s *ssh.Session) (cleanup, error) {
	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	outDoneCh := make(chan struct{})
	errDoneCh := make(chan struct{})
	go b.copyOutput(o, outR, outDoneCh)
	go b.copyOutput(o, errR, errDoneCh)
	stdout, err := s.StdoutPipe()

	cleanupF := func() {
		outW.Close()
		errW.Close()
		<-outDoneCh
		<-errDoneCh
	}

	if err != nil {
		cleanupF()
		return nil, fmt.Errorf("Unable to setup stdout for session: %v", err)
	}
	go io.Copy(outW, stdout)

	stderr, err := s.StderrPipe()
	if err != nil {
		cleanupF()
		return nil, fmt.Errorf("Unable to setup stderr for session: %v", err)
	}
	go io.Copy(errW, stderr)

	return cleanupF, nil
}

func (b *BastionKeyScan) execute(command string, connection *ssh.Client, o terraform.UIOutput) error {
	b.output(o, fmt.Sprintf("running command: %s", command))
	session, err := connection.NewSession()
	if err != nil {
		return b.makeError("failed to create session: %s.", err)
	}
	defer session.Close()
	if err := session.RequestPty("xterm", 80, 40, b.sshModes()); err != nil {
		return b.makeError("request for pseudo terminal failed: %s.", err)
	}
	cleanupF, err := b.redirectOutputs(o, session)
	if err != nil {
		return err
	}
	defer cleanupF()
	commandResult := session.Run(command)
	return commandResult
}

// Scan executes an ssh-keyscan operation.
func (b *BastionKeyScan) Scan(o terraform.UIOutput, host string, port int, ansibleSSHSettings *types.AnsibleSSHSettings) error {
	b.output(o, fmt.Sprintf("connecting using SSH to %s@%s:%d...", b.username, b.host, b.port))
	connection, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", b.host, b.port), b.sshConfig())
	if err != nil {
		return b.makeError("failed to dial: %s.", err)
	}
	defer connection.Close()

	b.output(o, fmt.Sprintf("ensuring the existence of a known hosts file at %s...", bastionHostKnownHostsFile))
	if err := b.execute(
		fmt.Sprintf(
			"mkdir -p \"%s\" && touch \"%s\"",
			b.quotedSSHKnownFileDir(),
			b.quotedSSHKnownFilePath()),
		connection, o); err != nil {
		return err
	}

	u1 := uuid.Must(uuid.NewV4())
	targetPath := filepath.Join(b.quotedSSHKnownFileDir(), u1.String())

	timeoutMs := ansibleSSHSettings.SSHKeyscanSeconds() * 1000
	timeSpentMs := 0
	intervalMs := 500

	for {
		sshKeyScanCommand := fmt.Sprintf("ssh-keyscan -p %d %s 2>/dev/null | head -n1 > \"%s\"", port, host, targetPath)
		keyScanError := b.execute(sshKeyScanCommand, connection, o)
		if err := b.execute(fmt.Sprintf("stat \"%s\"", targetPath), connection, o); err == nil {
			break
		} else {
			b.output(o, fmt.Sprintf("ssh-keyscan hasn't succeeded yet (last error: %s); retrying...", keyScanError))
			time.Sleep(time.Duration(intervalMs) * time.Millisecond)
			timeSpentMs = timeSpentMs + intervalMs
			if timeSpentMs > timeoutMs {
				b.execute(fmt.Sprintf("rm -rf \"%s\"", targetPath), connection, o) // cleanup, just in case
				return b.makeError(
					fmt.Sprintf(
						"failed receive target ssh key for %s:%d within time specified period of %d seconds.",
						host, port, timeoutMs/1000), nil)
			}
		}
	}

	b.execute(
		fmt.Sprintf(
			"echo $(cat \"%s\") >> \"%s\" && rm -rf \"%s\"",
			targetPath,
			b.quotedSSHKnownFilePath(),
			targetPath),
		connection, o)

	return nil
}

func (b *BastionKeyScan) quotedSSHKnownFileDir() string {
	return strings.Replace(filepath.Dir(bastionHostKnownHostsFile), "~/", "$HOME/", 1)
}
func (b *BastionKeyScan) quotedSSHKnownFilePath() string {
	return strings.Replace(bastionHostKnownHostsFile, "~/", "$HOME/", 1)
}
