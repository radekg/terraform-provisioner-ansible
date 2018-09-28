package types

import (
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	ansiblePlaybookAttributeForceHandlers = "force_handlers"
	ansiblePlaybookAttributeSkipTags      = "skip_tags"
	ansiblePlaybookAttributeStartAtTask   = "start_at_task"
	ansiblePlaybookAttributeTags          = "tags"
	ansiblePlaybookAttributeFilePath      = "file_path"
	ansiblePlaybookAttributeIncludeRoles  = "include_roles"
)

// Playbook represents playbook settings.
type Playbook struct {
	forceHandlers bool
	skipTags      []string
	startAtTask   string
	tags          []string
	filePath      string
	includeRoles  []string

	// when running a remote provisioner, the path will changed to the remote path:
	runnableFilePath string
}

// NewPlaybookSchema returns a new Ansible playbook schema.
func NewPlaybookSchema() *schema.Schema {
	return &schema.Schema{
		Type:          schema.TypeSet,
		Optional:      true,
		ConflictsWith: []string{"plays.module"},
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				// Ansible parameters:
				ansiblePlaybookAttributeForceHandlers: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				ansiblePlaybookAttributeSkipTags: &schema.Schema{
					Type:     schema.TypeList,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Optional: true,
				},
				ansiblePlaybookAttributeStartAtTask: &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				ansiblePlaybookAttributeTags: &schema.Schema{
					Type:     schema.TypeList,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Optional: true,
				},
				// operational:
				ansiblePlaybookAttributeFilePath: &schema.Schema{
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: vfPath,
				},
				ansiblePlaybookAttributeIncludeRoles: &schema.Schema{
					Type:     schema.TypeList,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Optional: true,
				},
			},
		},
	}
}

// NewPlaybookFromInterface reads Playbook configuration from Terraform schema.
func NewPlaybookFromInterface(i interface{}) *Playbook {
	vals := mapFromTypeSetList(i.(*schema.Set).List())
	return &Playbook{
		filePath:      vals[ansiblePlaybookAttributeFilePath].(string),
		forceHandlers: vals[ansiblePlaybookAttributeForceHandlers].(bool),
		skipTags:      listOfInterfaceToListOfString(vals[ansiblePlaybookAttributeSkipTags].([]interface{})),
		startAtTask:   vals[ansiblePlaybookAttributeStartAtTask].(string),
		tags:          listOfInterfaceToListOfString(vals[ansiblePlaybookAttributeTags].([]interface{})),
		includeRoles:  listOfInterfaceToListOfString(vals[ansiblePlaybookAttributeIncludeRoles].([]interface{})),
	}
}

// FilePath represents a path to the Ansible playbook to be executed.
func (v *Playbook) FilePath() string {
	if v.runnableFilePath == "" {
		return v.filePath
	}
	return v.filePath
}

// ForceHandlers represents Ansible Playbook --force-handlers flag.
func (v *Playbook) ForceHandlers() bool {
	return v.forceHandlers
}

// SkipTags represents Ansible Playbook --skip-tags flag.
func (v *Playbook) SkipTags() []string {
	return v.skipTags
}

// StartAtTask represents Ansible Playbook --start-at-task flag.
func (v *Playbook) StartAtTask() string {
	return v.startAtTask
}

// Tags represents Ansible Playbook --tags flag.
func (v *Playbook) Tags() []string {
	return v.tags
}

// IncludeRoles returns a list of paths to additional roles to be uploaded
// with the playbook and included in the run. Use this argument when roles
// reside outside of the playbook directory.
func (v *Playbook) IncludeRoles() []string {
	return v.includeRoles
}

// SetRunnableFilePath is used by the remote provisioner to reference the correct
// playbook location after the upload to the provisioned machine.
func (v *Playbook) SetRunnableFilePath(path string) {
	v.runnableFilePath = path
}
