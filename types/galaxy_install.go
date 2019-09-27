package types

import "github.com/hashicorp/terraform/helper/schema"

// ansible-galaxy install -r requirements.yml

const (
	ansibleGalaxyAttributeForce        = "force"
	ansibleGalaxyAttributeIgnoreCerts  = "ignore_certs"
	ansibleGalaxyAttributeIgnoreErrors = "ignore_errors"
	ansibleGalaxyAttributeKeepScmMeta  = "keep_scm_meta"
	ansibleGalaxyAttributeNoDeps       = "no_deps"
	ansibleGalaxyAttributeRoleFile     = "role_file"
	ansibleGalaxyAttributeRolesPath    = "roles_path"
	ansibleGalaxyAttributeServer       = "server"
	ansibleGalaxyAttributeVerbose      = "verbose"
)

// GalaxyInstall represents ansible-galaxy settings.
type GalaxyInstall struct {
	force        bool
	ignoreCerts  bool
	ignoreErrors bool
	keepScmMeta  bool
	noDeps       bool
	roleFile     string
	rolesPath    string
	server       string
	verbose      bool
}

// NewGalaxyInstallSchema returns a new Ansible Galaxy schema for the install operation.
func NewGalaxyInstallSchema() *schema.Schema {
	return &schema.Schema{
		Type:          schema.TypeSet,
		Optional:      true,
		ConflictsWith: []string{"plays.module", "plays.playbook"},
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				// Ansible Galaxy parameters:
				ansibleGalaxyAttributeForce: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
				ansibleGalaxyAttributeIgnoreCerts: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
				ansibleGalaxyAttributeIgnoreErrors: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
				ansibleGalaxyAttributeKeepScmMeta: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
				ansibleGalaxyAttributeNoDeps: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
				ansibleGalaxyAttributeRoleFile: &schema.Schema{
					Type:         schema.TypeString,
					Required:     true,
					ValidateFunc: vfPath,
				},
				ansibleGalaxyAttributeRolesPath: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: vfPath,
				},
				ansibleGalaxyAttributeServer: &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				ansibleGalaxyAttributeVerbose: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  false,
				},
			},
		},
	}
}

// NewGalaxyInstallFromInterface reads Ansible Galaxy install configuration from Terraform schema.
func NewGalaxyInstallFromInterface(i interface{}) *GalaxyInstall {
	vals := mapFromTypeSetList(i.(*schema.Set).List())
	return &GalaxyInstall{
		force:        vals[ansibleGalaxyAttributeForce].(bool),
		ignoreCerts:  vals[ansibleGalaxyAttributeIgnoreCerts].(bool),
		ignoreErrors: vals[ansibleGalaxyAttributeIgnoreErrors].(bool),
		keepScmMeta:  vals[ansibleGalaxyAttributeKeepScmMeta].(bool),
		noDeps:       vals[ansibleGalaxyAttributeNoDeps].(bool),
		roleFile:     vals[ansibleGalaxyAttributeRoleFile].(string),
		rolesPath:    vals[ansibleGalaxyAttributeRolesPath].(string),
		server:       vals[ansibleGalaxyAttributeServer].(string),
		verbose:      vals[ansibleGalaxyAttributeVerbose].(bool),
	}
}

// Force is the ansible-galaxy install --force flag.
func (v *GalaxyInstall) Force() bool {
	return v.force
}

// IgnoreCerts is the ansible-galaxy --ignore-certs flag.
func (v *GalaxyInstall) IgnoreCerts() bool {
	return v.ignoreCerts
}

// IgnoreErrors is the ansible-galaxy install --ignore-errors flag.
func (v *GalaxyInstall) IgnoreErrors() bool {
	return v.ignoreErrors
}

// KeepScmMeta is the ansible-galaxy install --keep-scm-meta flag.
func (v *GalaxyInstall) KeepScmMeta() bool {
	return v.keepScmMeta
}

// NoDeps is the ansible-galaxy install --no-deps flag.
func (v *GalaxyInstall) NoDeps() bool {
	return v.noDeps
}

// RoleFile ansible-galaxy install --role-file.
func (v *GalaxyInstall) RoleFile() string {
	return v.roleFile
}

// SetRoleFile is used by the remote provisioner to set calculated role file path.
func (v *GalaxyInstall) SetRoleFile(p string) {
	v.roleFile = p
}

// RolesPath ansible-galaxy install --roles-path.
func (v *GalaxyInstall) RolesPath() string {
	return v.rolesPath
}

// SetRolesPath is used by the remote provisioner to set calculated roles path.
func (v *GalaxyInstall) SetRolesPath(p string) {
	v.rolesPath = p
}

// Server is the ansible-galaxy --server.
func (v *GalaxyInstall) Server() string {
	return v.server
}

// Verbose is the ansible-galaxy --verbose flag.
func (v *GalaxyInstall) Verbose() bool {
	return v.verbose
}
