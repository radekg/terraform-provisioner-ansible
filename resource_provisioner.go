package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
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
	bootstrapDirectory string = "/tmp/ansible-terraform-bootstrap"
	// shared:
	defaultBecomeMethod      string = ""
	defaultBecomeMethodSet   string = "sudo"
	defaultBecomeUser        string = ""
	defaultBecomeUserSet     string = "root"
	defaultForks             int    = 5
	defaultInventoryFile     string = ""
	defaultLimit             string = ""
	defaultVaultPasswordFile string = ""
	defaultVerbose           string = ""

	// playbook only:
	defaultForceHandlers string = ""
	defaultStartAtTask   string = ""
	// module only:
	defaultBackground  int    = 0
	defaultHostPattern string = "all"
	defaultOneLine     string = ""
	defaultPoll        int    = 15
)

var becomeMethods = map[string]bool{"sudo": true, "su": true, "pbrun": true, "pfexec": true, "doas": true, "dzdo": true, "ksu": true, "runas": true}

type ansibleInstaller struct {
	AnsibleVersion string
}

type ansibleCallbaleType int

type ansibleInventoryMeta struct {
	Hosts  []string
	Groups []string
}

type ansibleCallArgsShared struct {
	Become            bool
	BecomeMethod      string
	BecomeUser        string
	ExtraVars         map[string]interface{}
	Forks             int
	InventoryFile     string
	Limit             string
	VaultPasswordFile string
	Verbose           bool
}

type ansibleCallArgs struct {
	Shared ansibleCallArgsShared
}

type ansiblePlaybook struct {
	ForceHandlers bool
	SkipTags      []string
	StartAtTask   string
	Tags          []string
	FilePath      string
	IncludeRoles  []string
}

type ansibleModule struct {
	Name        string
	Args        map[string]interface{}
	Background  int
	HostPattern string
	OneLine     bool
	Poll        int
}

// -- validation functions

func vfBecomeMethod(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	if !becomeMethods[v] {
		errs = append(errs, fmt.Errorf("%s is not a valid become_method", v))
	}
	return
}

func vfPath(val interface{}, key string) (warns []string, errs []error) {
	v := val.(string)
	if strings.Index(v, "${path.module}") > -1 {
		warns = append(warns, fmt.Sprintf("I could not reliably determine the existence of '%s', most likely because of https://github.com/hashicorp/terraform/issues/17439. If the file does not exist, you'll experience a failure at runtime.", v))
	} else {
		if _, err := resolvePath(v); err != nil {
			errs = append(errs, fmt.Errorf("file '%s' does not exist", v))
		}
	}
	return
}

// -- provisioner:

type provisioner struct {
	Plays          []play
	InventoryMeta  ansibleInventoryMeta
	Shared         ansibleCallArgsShared
	useSudo        bool
	skipInstall    bool
	skipCleanup    bool
	installVersion string
	local          bool
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
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},

						// entity to run:
						"playbook": &schema.Schema{
							Type:          schema.TypeSet,
							Optional:      true,
							ConflictsWith: []string{"plays.module"},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									// Ansible parameters:
									"force_handlers": &schema.Schema{
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
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
									// operational:
									"file_path": &schema.Schema{
										Type:         schema.TypeString,
										Required:     true,
										ValidateFunc: vfPath,
									},
									"include_roles": &schema.Schema{
										Type:     schema.TypeList,
										Elem:     &schema.Schema{Type: schema.TypeString},
										Optional: true,
									},
								},
							},
						},

						"module": &schema.Schema{
							Type:          schema.TypeSet,
							Optional:      true,
							ConflictsWith: []string{"plays.playbook"},
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									// Ansible parameters:
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
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
									"poll": &schema.Schema{
										Type:     schema.TypeInt,
										Optional: true,
										Default:  defaultPoll,
									},
									// operational:
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},

						"hosts": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"host": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"ansible_properties": &schema.Schema{
										Type:     schema.TypeMap,
										Optional: true,
										Computed: true,
									},
								},
							},
						},
						"groups": &schema.Schema{
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
						"become": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"become_method": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      defaultBecomeMethod,
							ValidateFunc: vfBecomeMethod,
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
							Type:         schema.TypeString,
							Optional:     true,
							Default:      defaultInventoryFile,
							ValidateFunc: vfPath,
						},
						"limit": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultLimit,
						},
						"vault_password_file": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      defaultVaultPasswordFile,
							ValidateFunc: vfPath,
						},
						"verbose": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultVerbose,
						},
					},
				},
			},

			"defaults": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hosts": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"host": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"ansible_properties": &schema.Schema{
										Type:     schema.TypeMap,
										Optional: true,
										Computed: true,
									},
								},
							},
						},
						"groups": &schema.Schema{
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
						"become": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  false,
						},
						"become_method": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      defaultBecomeMethod,
							ValidateFunc: vfBecomeMethod,
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
							Type:         schema.TypeString,
							Optional:     true,
							Default:      defaultInventoryFile,
							ValidateFunc: vfPath,
						},
						"limit": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultLimit,
						},
						"vault_password_file": &schema.Schema{
							Type:         schema.TypeString,
							Optional:     true,
							Default:      defaultVaultPasswordFile,
							ValidateFunc: vfPath,
						},
						"verbose": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  defaultVerbose,
						},
					},
				},
			},

			"remote":               newRemoteSchema(),
			"ansible_ssh_settings": newAnsibleSSHSettingsSchema(),
		},
		ValidateFunc: validateFn,
		ApplyFunc:    applyFn,
	}
}

func validateFn(c *terraform.ResourceConfig) (ws []string, es []error) {

	defer func() {
		if r := recover(); r != nil {
			es = append(es, fmt.Errorf("error while validating the provisioner, reason: %+v", r))
		}
	}()

	_, isRemoteProvisioning := c.Get("remote")

	validPlaysCount := 0

	if plays, hasPlays := c.Get("plays"); hasPlays {
		for _, vPlay := range plays.([]map[string]interface{}) {

			currentErrorCount := len(es)

			vPlaybook, playHasPlaybook := vPlay["playbook"]
			_, playHasModule := vPlay["module"]

			if playHasPlaybook && playHasModule {
				es = append(es, fmt.Errorf("playbook and module can't be used together"))
			} else if !playHasPlaybook && !playHasModule {
				es = append(es, fmt.Errorf("playbook or module must be set"))
			} else {

				// a local provisioning play playbook include_roles shall be ignored
				if playHasPlaybook {
					if !isRemoteProvisioning {
						vPlaybookTyped := vPlaybook.([]map[string]interface{})
						playbookRoles, hasIncludeRoles := vPlaybookTyped[0]["include_roles"]
						if hasIncludeRoles && len(playbookRoles.([]string)) > 0 {
							playbookFilePath, _ := vPlaybookTyped[0]["file_path"]
							ws = append(ws, fmt.Sprintf("include_roles omited for playbook '%s' when local provisioning is used", playbookFilePath))
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

	return ws, es
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

	if p.local {

		// Wait and retry until we establish the connection
		err = retryFunc(comm.Timeout(), func() error {
			return comm.Connect(o)
		})

		if err != nil {
			return err
		}

		defer comm.Disconnect()

		if p.skipInstall {
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

	if p.local && p.skipCleanup {
		p.remoteCleanupAfterBootstrap(o, comm)
	}

	return nil

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
	vRemoteSettings := newRemoteSettingsFromInterface(d.GetOk("remote"))
	fmt.Printf(" ==============> Remote\n\t\t%+v\n", vRemoteSettings)
	vAnsibleSSHSettings := newAnsibleSSHSettingsFromInterface(d.GetOk("ansible_ssh_settings"))
	fmt.Printf(" ==============> Ansible SSH settings\n\t\t%+v\n", vAnsibleSSHSettings)
	/*
		p := &provisioner{
			useSudo:        d.Get("use_sudo").(bool),
			skipInstall:    d.Get("skip_install").(bool),
			skipCleanup:    d.Get("skip_cleanup").(bool),
			installVersion: d.Get("install_version").(string),
			local:          d.Get("local").(bool),
			Plays:          make([]play, 0),
			InventoryMeta: ansibleInventoryMeta{
				Hosts:  getStringList(d.Get("hosts")),
				Groups: getStringList(d.Get("groups")),
			},
			Shared: ansibleCallArgsShared{
				Become:            d.Get("become").(bool),
				BecomeMethod:      d.Get("become_method").(string),
				BecomeUser:        d.Get("become_user").(string),
				ExtraVars:         getStringMap(d.Get("extra_vars")),
				Forks:             d.Get("forks").(int),
				InventoryFile:     d.Get("inventory_file").(string),
				Limit:             d.Get("limit").(string),
				VaultPasswordFile: d.Get("vault_password_file").(string),
				Verbose:           d.Get("verbose").(bool),
			},
		}
		p.InventoryMeta = ensureLocalhostInCallArgsHosts(p.InventoryMeta)
		p.Plays = decodePlays(d.Get("plays").([]interface{}), p.InventoryMeta, p.Shared)
		return p, nil
	*/
	return nil, nil
}

func decodePlays(v []interface{}, fallbackInventoryMeta ansibleInventoryMeta, fallbackArgs ansibleCallArgsShared) []play {
	plays := make([]play, 0, len(v))
	/* TODO: FIX me
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
			WithRoles:    getStringList(playData["with_roles"]),
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
	*/
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
