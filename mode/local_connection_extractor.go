package mode

import (
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hashicorp/terraform/communicator/shared"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/crypto/ssh"
)

const (
	// DefaultUser is used if there is no user given
	DefaultUser = "root"

	// DefaultPort is used if there is no port given
	DefaultPort = 22

	// DefaultScriptPath is used as the path to copy the file to
	// for remote execution if not provided otherwise.
	DefaultScriptPath = "/tmp/terraform_%RAND%.sh"

	// DefaultTimeout is used if there is no timeout given
	DefaultTimeout = 5 * time.Minute
)

type connectionInfo struct {
	User       string
	Password   string
	PrivateKey string `mapstructure:"private_key"`
	Host       string
	HostKey    string `mapstructure:"host_key"`
	Port       int
	Agent      bool
	Timeout    string
	ScriptPath string        `mapstructure:"script_path"`
	TimeoutVal time.Duration `mapstructure:"-"`

	BastionUser       string `mapstructure:"bastion_user"`
	BastionPassword   string `mapstructure:"bastion_password"`
	BastionPrivateKey string `mapstructure:"bastion_private_key"`
	BastionHost       string `mapstructure:"bastion_host"`
	BastionHostKey    string `mapstructure:"bastion_host_key"`
	BastionPort       int    `mapstructure:"bastion_port"`

	AgentIdentity string `mapstructure:"agent_identity"`
}

func parseConnectionInfo(s *terraform.InstanceState) (*connectionInfo, error) {
	connInfo := &connectionInfo{}
	decConf := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           connInfo,
	}
	dec, err := mapstructure.NewDecoder(decConf)
	if err != nil {
		return nil, err
	}
	if err := dec.Decode(s.Ephemeral.ConnInfo); err != nil {
		return nil, err
	}

	// To default Agent to true, we need to check the raw string, since the
	// decoded boolean can't represent "absence of config".
	//
	// And if SSH_AUTH_SOCK is not set, there's no agent to connect to, so we
	// shouldn't try.
	if s.Ephemeral.ConnInfo["agent"] == "" && os.Getenv("SSH_AUTH_SOCK") != "" {
		connInfo.Agent = true
	}

	if connInfo.User == "" {
		connInfo.User = DefaultUser
	}

	// Format the host if needed.
	// Needed for IPv6 support.
	connInfo.Host = shared.IpFormat(connInfo.Host)

	if connInfo.Port == 0 {
		connInfo.Port = DefaultPort
	}
	if connInfo.ScriptPath == "" {
		connInfo.ScriptPath = DefaultScriptPath
	}
	if connInfo.Timeout != "" {
		connInfo.TimeoutVal = safeDuration(connInfo.Timeout, DefaultTimeout)
	} else {
		connInfo.TimeoutVal = DefaultTimeout
	}

	if connInfo.PrivateKey != "" {
		if err := validatePrivateKey(&connInfo.PrivateKey); err != nil {
			return nil, err
		}
	}
	// Default all bastion config attrs to their non-bastion counterparts
	if connInfo.BastionHost != "" {
		// Format the bastion host if needed.
		// Needed for IPv6 support.
		connInfo.BastionHost = shared.IpFormat(connInfo.BastionHost)

		if connInfo.BastionUser == "" {
			connInfo.BastionUser = connInfo.User
		}
		if connInfo.BastionPassword == "" {
			connInfo.BastionPassword = connInfo.Password
		}
		if connInfo.BastionPrivateKey == "" {
			connInfo.BastionPrivateKey = connInfo.PrivateKey
		} else {
			if err := validatePrivateKey(&connInfo.BastionPrivateKey); err != nil {
				return nil, err
			}
		}
		if connInfo.BastionPort == 0 {
			connInfo.BastionPort = connInfo.Port
		}
	}

	return connInfo, nil
}

func safeDuration(dur string, defaultDur time.Duration) time.Duration {
	d, err := time.ParseDuration(dur)
	if err != nil {
		log.Printf("Invalid duration '%s', using default of %s", dur, defaultDur)
		return defaultDur
	}
	return d
}

func validatePrivateKey(key *string) error {
	pk := []byte(*key)
	block, _ := pem.Decode(pk)
	if block == nil {
		return fmt.Errorf("Failed to decode private key %q: no key found", pk)
	}
	// from https://github.com/hashicorp/terraform/blob/d4ac68423c4998279f33404db46809d27a5c2362/communicator/ssh/provisioner.go#L257
	// ... preferably Terraform exposes some public interface for these operations.
	if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
		return fmt.Errorf(
			"Failed to read key %q: password protected keys are "+
				"not supported; please decrypt the key prior to use", pk)
	}
	if _, err := ssh.ParsePrivateKey([]byte(pk)); err != nil {
		return fmt.Errorf("Failed to parse private key file %q: %s", pk, err)
	}
	// end from
	*key = string(pem.EncodeToMemory(block))
	return nil
}
