package types

import "github.com/hashicorp/terraform/helper/schema"

const (
	// default values:
	ansibleModuleDefaultHostPattern = "all"
	ansibleModuleDefaultPoll        = 15
	// attribute names:
	ansibleModuleAttributeArgs        = "args"
	ansibleModuleAttributeBackground  = "background"
	ansibleModuleAttributeHostPattern = "host_pattern"
	ansibleModuleAttributeOneLine     = "one_line"
	ansibleModuleAttributePoll        = "poll"
	ansibleModuleAttributeModule      = "module"
)

// Module represents module settings.
type Module struct {
	args        map[string]interface{}
	background  int
	hostPattern string
	oneLine     bool
	poll        int
	module      string
}

// NewPlaybookSchema return a new Ansible module schema.
func NewModuleSchema() *schema.Schema {
	return &schema.Schema{
		Type:          schema.TypeSet,
		Optional:      true,
		ConflictsWith: []string{"plays.playbook"},
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				// Ansible parameters:
				ansibleModuleAttributeArgs: &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
					Computed: true,
				},
				ansibleModuleAttributeBackground: &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
					Default:  0,
				},
				ansibleModuleAttributeHostPattern: &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  ansibleModuleDefaultHostPattern,
				},
				ansibleModuleAttributeOneLine: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
				ansibleModuleAttributePoll: &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
					Default:  ansibleModuleDefaultPoll,
				},
				// operational:
				ansibleModuleAttributeModule: &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
	}
}

// NewModuleFromInterface reads Module configuration from Terraform schema.
func NewModuleFromInterface(i interface{}) *Module {
	vals := mapFromTypeSetList(i.(*schema.Set).List())
	return &Module{
		module:      vals[ansibleModuleAttributeModule].(string),
		args:        mapFromTypeMap(vals[ansibleModuleAttributeArgs]),
		background:  vals[ansibleModuleAttributeBackground].(int),
		hostPattern: vals[ansibleModuleAttributeHostPattern].(string),
		oneLine:     vals[ansibleModuleAttributeOneLine].(bool),
		poll:        vals[ansibleModuleAttributePoll].(int),
	}
}

func (v *Module) Module() string {
	return v.module
}

func (v *Module) Args() map[string]interface{} {
	return v.args
}

func (v *Module) Background() int {
	return v.background
}

func (v *Module) HostPattern() string {
	return v.hostPattern
}

func (v *Module) OneLine() bool {
	return v.oneLine
}

func (v *Module) Poll() int {
	return v.poll
}
