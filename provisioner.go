package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"

	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/go-linereader"
)

const (
	yes                string = "yes"
	no                 string = "no"
	bootstrapDirectory string = "/tmp/ansible-terraform-bootstrap"
	// shared:
	defaultBecome            string = ""
	defaultBecomeSet         string = no
	defaultBecomeMethod      string = ""
	defaultBecomeMethodSet   string = "sudo"
	defaultBecomeUser        string = ""
	defaultBecomeUserSet     string = "root"
	defaultForks             int    = 5
	defaultInventoryFile     string = ""
	defaultLimit             string = ""
	defaultVaultPasswordFile string = ""
	defaultVerbose           string = ""

	defaultPlayEnabled string = yes
	// playbook only:
	defaultForceHandlers string = ""
	defaultStartAtTask   string = ""
	// module only:
	defaultBackground  int    = 0
	defaultHostPattern string = "all"
	defaultOneLine     string = ""
	defaultPoll        int    = 15

	defaultUseSudo     string = yes
	defaultSkipInstall string = no
	defaultSkipCleanup string = no
	defaultLocal       string = no
)

var yesNoStates = map[string]bool{yes: true, no: true}
var becomeMethods = map[string]bool{"sudo": true, "su": true, "pbrun": true, "pfexec": true, "doas": true, "dzdo": true, "ksu": true, "runas": true}

type ansibleInstaller struct {
	AnsibleVersion string
}

type ansibleCallbaleType int

const (
	ansibleCallableUndefined ansibleCallbaleType = iota
	ansibleCallableConflicting
	ansibleCallablePlaybook
	ansibleCallableModule
)

type ansibleInventoryMeta struct {
	Hosts  []string
	Groups []string
}

type ansibleCallArgsShared struct {
	Become            string
	BecomeMethod      string
	BecomeUser        string
	ExtraVars         map[string]interface{}
	Forks             int
	InventoryFile     string
	Limit             string
	VaultPasswordFile string
	Verbose           string
}

type ansibleCallArgs struct {
	// module only:
	Args        map[string]interface{}
	Background  int
	HostPattern string
	OneLine     string
	Poll        int
	// Playbook only:
	ForceHandlers string
	SkipTags      []string
	StartAtTask   string
	Tags          []string
	// shared:
	Shared ansibleCallArgsShared
}

// -- provisioner:

type provisioner struct {
	Plays          []play
	InventoryMeta  ansibleInventoryMeta
	Shared         ansibleCallArgsShared
	useSudo        string
	skipInstall    string
	skipCleanup    string
	installVersion string
	local          string
}

// Provisioner describes this provisioner configuration.
func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{

			"plays": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultPlayEnabled,
						},
						// entity to run:
						"playbook": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"module": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						// meta for temporary inventory:
						"hosts": &schema.Schema{
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
						"groups": &schema.Schema{
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
						// module only:
						"args": &schema.Schema{
							Type:     schema.TypeMap,
							Optional: true,
							Computed: true,
						},
						"background": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  defaultBackground,
						},
						"host_pattern": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultHostPattern,
						},
						"one_line": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultOneLine,
						},
						"poll": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  defaultPoll,
						},
						// playbook only:
						"force_handlers": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultForceHandlers,
						},
						"skip_tags": &schema.Schema{
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
						"start_at_task": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultStartAtTask,
						},
						"tags": &schema.Schema{
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
						// shared:
						"become": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultBecome,
						},
						"become_method": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultBecomeMethod,
						},
						"become_user": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultBecomeUser,
						},
						"extra_vars": &schema.Schema{
							Type:     schema.TypeMap,
							Optional: true,
							Computed: true,
						},
						"forks": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  defaultForks,
						},
						"inventory_file": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultInventoryFile,
						},
						"limit": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultLimit,
						},
						"vault_password_file": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultVaultPasswordFile,
						},
						"verbose": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultVerbose,
						},
					},
				},
			},

			// inventory meta:
			"hosts": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"groups": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},

			// shared:
			"become": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  defaultBecome,
			},
			"become_method": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  defaultBecomeMethod,
			},
			"become_user": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  defaultBecomeUser,
			},
			"extra_vars": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
			},
			"forks": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  defaultForks,
			},
			"inventory_file": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  defaultInventoryFile,
			},
			"limit": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  defaultLimit,
			},
			"vault_password_file": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  defaultVaultPasswordFile,
			},
			"verbose": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  defaultVerbose,
			},

			"use_sudo": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  defaultUseSudo,
			},
			"skip_install": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  defaultSkipInstall,
			},
			"skip_cleanup": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  defaultSkipCleanup,
			},
			"install_version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "", // latest
			},
			"local": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  defaultLocal,
			},
		},
		ApplyFunc:    applyFn,
		ValidateFunc: validateFn,
	}
}

func applyFn(ctx context.Context) error {

	o := ctx.Value(schema.ProvOutputKey).(terraform.UIOutput)
	s := ctx.Value(schema.ProvRawStateKey).(*terraform.InstanceState)
	d := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)

	// Decode the provisioner config
	p, err := decodeConfig(d)
	if err != nil {
		return err
	}

	// Get a new communicator
	comm, err := communicator.New(s)
	if err != nil {
		return err
	}

	if p.local == no {

		// Wait and retry until we establish the connection
		err = retryFunc(comm.Timeout(), func() error {
			return comm.Connect(o)
		})

		if err != nil {
			return err
		}

		defer comm.Disconnect()

		if p.skipInstall == no {
			if err := p.remoteInstallAnsible(o, comm); err != nil {
				return err
			}
		}

		runnables, err := p.remoteDeployAnsibleData(o, comm)

		if err != nil {
			o.Output(fmt.Sprintf("%+v", err))
			return err
		}

		for _, runnable := range runnables {
			command, err := runnable.ToCommand()
			if err != nil {
				return err
			}
			o.Output(fmt.Sprintf("running command: %s", command))
			if err := p.remoteRunCommandSudo(o, comm, command); err != nil {
				return err
			}
		}

	} else {

		connType := s.Ephemeral.ConnInfo["type"]
		switch connType {
		case "ssh", "": // The default connection type is ssh, so if connType is empty use ssh
		default:
			return fmt.Errorf("Currently, only SSH connection is supported")
		}

		connInfo, err := parseConnectionInfo(s)
		if err != nil {
			return err
		}

		if connInfo.User == "" || connInfo.Host == "" {
			return fmt.Errorf("Local mode requires a connection with username and host")
		}

		if connInfo.PrivateKey == "" {
			o.Output(fmt.Sprintf("no private key for %s@%s found, assuming ssh agent...", connInfo.User, connInfo.Host))
		}

		runnables, err := p.localGatherRunnables(o, connInfo)

		if err != nil {
			o.Output(fmt.Sprintf("%+v", err))
			return err
		}

		pemFile := ""
		if connInfo.PrivateKey != "" {
			pemFile, err = p.localWritePem(o, connInfo)
			if err != nil {
				return err
			}
			defer os.Remove(pemFile)
		}

		knownHostsFile, err := p.localEnsureKnownHosts(o, connInfo)
		if err != nil {
			return err
		}
		defer os.Remove(knownHostsFile)

		for _, runnable := range runnables {

			if runnable.InventoryFileTemporary {
				defer os.Remove(runnable.InventoryFile)
			}

			bastionHost := ""
			bastionUsername := connInfo.User
			bastionPemFile := pemFile
			bastionPort := connInfo.Port

			if connInfo.BastionHost != "" {
				bastionHost = connInfo.BastionHost
				if connInfo.BastionUser != "" {
					bastionUsername = connInfo.BastionUser
				}
				if connInfo.BastionPrivateKey != "" {
					bastionPemFile = connInfo.BastionPrivateKey
				}
				if connInfo.BastionPort > 0 {
					bastionPort = connInfo.BastionPort
				}
			}

			command, err := runnable.ToLocalCommand(o, runnablePlayLocalAnsibleArgs{
				Username:        connInfo.User,
				Port:            connInfo.Port,
				PemFile:         pemFile,
				KnownHostsFile:  knownHostsFile,
				BastionHost:     bastionHost,
				BastionPemFile:  bastionPemFile,
				BastionPort:     bastionPort,
				BastionUsername: bastionUsername,
			})

			if err != nil {
				return err
			}

			if connInfo.BastionHost != "" {
				o.Output(fmt.Sprintf("executing ssh-keyscan on bastion: %s@%s", bastionUsername, fmt.Sprintf("%s:%d", bastionHost, bastionPort)))
				bastionSSHKeyScan := NewBastionKeyScan(
					bastionHost,
					bastionPort,
					bastionUsername,
					bastionPemFile)
				if err := bastionSSHKeyScan.Scan(o, connInfo.Host, connInfo.Port); err != nil {
					return err
				}
			}

			o.Output(fmt.Sprintf("running local command: %s", command))

			if err := p.localRunCommand(o, command); err != nil {
				return err
			}
		}

	}

	if p.local == no && p.skipCleanup == no {
		p.remoteCleanupAfterBootstrap(o, comm)
	}

	return nil

}

func validateFn(c *terraform.ResourceConfig) (ws []string, es []error) {

	defer func() {
		if r := recover(); r != nil {
			es = append(es, fmt.Errorf("error while validating the provisioner, reason: %+v", r))
		}
	}()

	becomeMethod, ok := c.Get("become_method")
	if ok {
		if !becomeMethods[becomeMethod.(string)] {
			es = append(es, errors.New(becomeMethod.(string)+" is not a valid become_method"))
		}
	}

	yesNoFields := []string{"verbose", "force_handlers", "one_line", "become", "use_sudo", "skip_install", "skip_cleanup", "local"}
	yesNoFieldsPlay := []string{"enabled"}
	for _, f := range yesNoFields {
		yesNoFieldsPlay = append(yesNoFieldsPlay, f)
	}

	for _, field := range yesNoFields {
		v, ok := c.Get(field)
		if ok && v.(string) != "" {
			if !yesNoStates[v.(string)] {
				es = append(es, errors.New(v.(string)+" is not a valid "+field+" value, must be yes/no"))
			}
		}
	}

	for _, ftt := range []string{"inventory_file", "vault_password_file"} {
		value, ok := c.Get(ftt)
		if ok && len(value.(string)) > 0 {
			if _, err := resolvePath(value.(string)); err != nil {
				es = append(es, errors.New("file "+value.(string)+" does not exist"))
			}
		}
	}

	// Validate plays configs
	validPlaysCount := 0
	plays, ok := c.Get("plays")
	if ok {
		for _, p := range plays.([]map[string]interface{}) {

			currentErrorCount := len(es)

			playbook, okp := p["playbook"]
			module, okm := p["module"]

			if okp && okm && len(playbook.(string)) > 0 && len(module.(string)) > 0 {
				es = append(es, errors.New("playbook and module can't be used together"))
			} else {

				isPlaybook := okp && len(playbook.(string)) > 0
				isModule := okm && len(module.(string)) > 0

				if !okp && !okm {
					es = append(es, errors.New("playbook or module must be set"))
				}

				if isPlaybook {
					disallowedFields := []string{"args", "background", "host_pattern", "one_line", "poll"}
					for _, df := range disallowedFields {
						if _, ok = p[df]; ok {
							es = append(es, fmt.Errorf("%s must not be used with playbook", df))
						}
					}
				}

				if isModule {
					disallowedFields := []string{"force_handlers", "skip_tags", "start_at_task", "tags"}
					for _, df := range disallowedFields {
						if _, ok = p[df]; ok {
							es = append(es, fmt.Errorf("%s must not be used with module", df))
						}
					}
				}

			}

			becomeMethodPlay, ok := p["become_method"]
			if ok {
				if !becomeMethods[becomeMethodPlay.(string)] {
					es = append(es, errors.New(becomeMethodPlay.(string)+" is not a valid become_method"))
				}
			}

			for _, fieldPlay := range yesNoFieldsPlay {
				v, ok := p[fieldPlay]
				if ok && v.(string) != "" {
					if !yesNoStates[v.(string)] {
						es = append(es, errors.New(v.(string)+" is not a valid "+fieldPlay+" value, must be yes/no"))
					}
				}
			}

			for _, ftt := range []string{"inventory_file", "playbook", "vault_password_file"} {
				value, ok := p[ftt]
				if ok && len(value.(string)) > 0 {
					if strings.Index(value.(string), "${path.module}") > -1 {
						ws = append(ws, "I could not reliably determine the existence of '"+ftt+"', most likely because of https://github.com/hashicorp/terraform/issues/17439. If the file does not exist, you'll experience a failure at runtime.")
					} else {
						if _, err := resolvePath(value.(string)); err != nil {
							es = append(es, errors.New("file "+value.(string)+" does not exist"))
						}
					}
				}
			}

			if currentErrorCount == len(es) {
				validPlaysCount++
			}

		}

		if validPlaysCount == 0 {
			ws = append(ws, "nothing to play")
		}

	} else {
		ws = append(ws, "nothing to play")
	}

	local, ok := c.Get("local")
	if ok {
		if local.(string) == yes {
			for _, cf := range []string{"use_sudo", "install_version", "skip_cleanup", "skip_install"} {
				if _, ok := c.Get(cf); ok {
					es = append(es, errors.New(cf+" must not be used when local = true"))
				}
			}
		}
	}

	return ws, es
}

// -- UTILITY:

func (p *provisioner) copyOutput(o terraform.UIOutput, r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}

func resolvePath(path string) (string, error) {
	expandedPath, _ := homedir.Expand(path)
	if _, err := os.Stat(expandedPath); err == nil {
		return expandedPath, nil
	}
	return "", fmt.Errorf("Ansible module not found at path: [%s]", path)
}

// retryFunc is used to retry a function for a given duration
func retryFunc(timeout time.Duration, f func() error) error {
	finish := time.After(timeout)
	for {
		err := f()
		if err == nil {
			return nil
		}
		log.Printf("Retryable error: %v", err)

		select {
		case <-finish:
			return err
		case <-time.After(3 * time.Second):
		}
	}
}

func decodeConfig(d *schema.ResourceData) (*provisioner, error) {
	p := &provisioner{
		useSudo:        d.Get("use_sudo").(string),
		skipInstall:    d.Get("skip_install").(string),
		skipCleanup:    d.Get("skip_cleanup").(string),
		installVersion: d.Get("install_version").(string),
		local:          d.Get("local").(string),
		Plays:          make([]play, 0),
		InventoryMeta: ansibleInventoryMeta{
			Hosts:  getStringList(d.Get("hosts")),
			Groups: getStringList(d.Get("groups")),
		},
		Shared: ansibleCallArgsShared{
			Become:            d.Get("become").(string),
			BecomeMethod:      d.Get("become_method").(string),
			BecomeUser:        d.Get("become_user").(string),
			ExtraVars:         getStringMap(d.Get("extra_vars")),
			Forks:             d.Get("forks").(int),
			InventoryFile:     d.Get("inventory_file").(string),
			Limit:             d.Get("limit").(string),
			VaultPasswordFile: d.Get("vault_password_file").(string),
			Verbose:           d.Get("verbose").(string),
		},
	}
	p.InventoryMeta = ensureLocalhostInCallArgsHosts(p.InventoryMeta)
	p.Plays = decodePlays(d.Get("plays").([]interface{}), p.InventoryMeta, p.Shared)
	return p, nil
}

func decodePlays(v []interface{}, fallbackInventoryMeta ansibleInventoryMeta, fallbackArgs ansibleCallArgsShared) []play {
	plays := make([]play, 0, len(v))
	for _, rawPlayData := range v {

		callable := ""
		callableType := ansibleCallableUndefined
		playData := rawPlayData.(map[string]interface{})
		playbook := (playData["playbook"].(string))
		module := (playData["module"].(string))

		if len(playbook) > 0 && len(module) > 0 {
			callableType = ansibleCallableConflicting
		} else {
			if len(playbook) > 0 {
				callable = playbook
				callableType = ansibleCallablePlaybook
			} else if len(module) > 0 {
				callable = module
				callableType = ansibleCallableModule
			} else {
				callableType = ansibleCallableUndefined
			}
		}

		playToAppend := play{
			Callable:     callable,
			CallableType: callableType,
			Enabled:      playData["enabled"].(string),
			InventoryMeta: ansibleInventoryMeta{
				Hosts:  withStringListFallback(getStringList(playData["hosts"]), fallbackInventoryMeta.Hosts),
				Groups: withStringListFallback(getStringList(playData["groups"]), fallbackInventoryMeta.Groups),
			},
			CallArgs: ansibleCallArgs{
				Shared: ansibleCallArgsShared{
					Become:            withStringFallback(playData["become"].(string), defaultBecome, fallbackArgs.Become),
					BecomeMethod:      withStringFallback(playData["become_method"].(string), defaultBecomeMethod, fallbackArgs.BecomeMethod),
					BecomeUser:        withStringFallback(playData["become_user"].(string), defaultBecomeUser, fallbackArgs.BecomeUser),
					ExtraVars:         withStringInterfaceMapFallback(getStringMap(playData["extra_vars"]), fallbackArgs.ExtraVars),
					Forks:             withIntFallback(playData["forks"].(int), defaultForks, fallbackArgs.Forks),
					InventoryFile:     withStringFallback(playData["inventory_file"].(string), defaultInventoryFile, fallbackArgs.InventoryFile),
					Limit:             withStringFallback(playData["limit"].(string), defaultLimit, fallbackArgs.Limit),
					VaultPasswordFile: withStringFallback(playData["vault_password_file"].(string), defaultVaultPasswordFile, fallbackArgs.VaultPasswordFile),
					Verbose:           withStringFallback(playData["verbose"].(string), defaultVerbose, fallbackArgs.Verbose),
				},
				// module only:
				Args:        getStringMap(playData["args"]),
				Background:  playData["background"].(int),
				HostPattern: playData["host_pattern"].(string),
				OneLine:     playData["one_line"].(string),
				Poll:        playData["poll"].(int),
				// playbook only:
				ForceHandlers: playData["force_handlers"].(string),
				SkipTags:      getStringList(playData["skip_tags"]),
				StartAtTask:   playData["start_at_task"].(string),
				Tags:          getStringList(playData["tags"]),
			},
		}
		playToAppend.InventoryMeta = ensureLocalhostInCallArgsHosts(playToAppend.InventoryMeta)

		plays = append(plays, playToAppend)
	}
	return plays
}

func ensureLocalhostInCallArgsHosts(inventoryMeta ansibleInventoryMeta) ansibleInventoryMeta {
	lc := "localhost"
	found := false
	for _, v := range inventoryMeta.Hosts {
		if v == lc {
			found = true
			break
		}
	}
	if !found {
		inventoryMeta.Hosts = append(inventoryMeta.Hosts, lc)
	}
	return inventoryMeta
}

func getStringList(v interface{}) []string {
	var result []string
	switch v := v.(type) {
	case nil:
		return result
	case []interface{}:
		for _, vv := range v {
			if vv, ok := vv.(string); ok {
				result = append(result, vv)
			}
		}
		return result
	default:
		panic(fmt.Sprintf("Unsupported type: %T", v))
	}
}

func getStringMap(v interface{}) map[string]interface{} {
	switch v := v.(type) {
	case nil:
		return make(map[string]interface{})
	case map[string]interface{}:
		return v
	default:
		panic(fmt.Sprintf("Unsupported type: %T", v))
	}
}

func withStringFallback(intended string, defaultValue string, fallback string) string {
	if intended == defaultValue {
		return fallback
	}
	return intended
}

func withIntFallback(intended int, defaultValue int, fallback int) int {
	if intended == defaultValue {
		return fallback
	}
	return intended
}

func withStringListFallback(intended []string, fallback []string) []string {
	if len(intended) == 0 {
		return fallback
	}
	return intended
}
func withStringInterfaceMapFallback(intended map[string]interface{}, fallback map[string]interface{}) map[string]interface{} {
	if len(intended) == 0 {
		return fallback
	}
	return intended
}

func getMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
