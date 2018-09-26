package types

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// Defaults represents default settings for each consequent play.
type Defaults struct {
	hosts             []Host
	groups            []string
	becomeMethod      string
	becomeUser        string
	extraVars         map[string]interface{}
	forks             int
	inventoryFile     string
	limit             string
	vaultPasswordFile string
	//
	hostsIsSet             bool
	groupsIsSet            bool
	becomeMethodIsSet      bool
	becomeUserIsSet        bool
	extraVarsIsSet         bool
	forksIsSet             bool
	inventoryFileIsSet     bool
	limitIsSet             bool
	vaultPasswordFileIsSet bool
}

const (
	// attribute names:
	defaultsAttributeHosts             = "hosts"
	defaultsAttributeGroups            = "groups"
	defaultsAttributeBecomeMethod      = "become_method"
	defaultsAttributeBecomeUser        = "become_user"
	defaultsAttributeExtraVars         = "extra_vars"
	defaultsAttributeForks             = "forks"
	defaultsAttributeInventoryFile     = "inventory_file"
	defaultsAttributeLimit             = "limit"
	defaultsAttributeVaultPasswordFile = "vault_password_file"
)

// NewDefaultsSchema returns a new defaults schema.
func NewDefaultsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				defaultsAttributeHosts: NewHostSchema(),
				defaultsAttributeGroups: &schema.Schema{
					Type:     schema.TypeList,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Optional: true,
				},
				defaultsAttributeBecomeMethod: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: VfBecomeMethod,
				},
				defaultsAttributeBecomeUser: &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				defaultsAttributeExtraVars: &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
					Computed: true,
				},
				defaultsAttributeForks: &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},
				defaultsAttributeInventoryFile: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: VfPath,
				},
				defaultsAttributeLimit: &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				defaultsAttributeVaultPasswordFile: &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: VfPath,
				},
			},
		},
	}
}

// NewDefaultsFromInterface reads Defaults configuration from Terraform schema.
func NewDefaultsFromInterface(i interface{}, ok bool) *Defaults {
	v := &Defaults{}
	if ok {
		vals := mapFromTypeSetList(i.(*schema.Set).List())
		if val, ok := vals[defaultsAttributeHosts]; ok {
			hosts := make([]Host, 0)
			hostSchema := NewHostSchema()
			for _, iface := range val.([]interface{}) {
				hosts = append(hosts, *NewHostFromInterface(schema.NewSet(schema.HashResource(hostSchema.Elem.(*schema.Resource)), []interface{}{iface})))
			}
			v.hosts = hosts
			v.hostsIsSet = len(v.hosts) > 0
		}
		if val, ok := vals[defaultsAttributeGroups]; ok {
			v.groups = listOfInterfaceToListOfString(val.([]interface{}))
			v.groupsIsSet = len(v.groups) > 0
		}
		if val, ok := vals[defaultsAttributeBecomeMethod]; ok {
			v.becomeMethod = val.(string)
			v.becomeMethodIsSet = v.becomeMethod != ""
		}
		if val, ok := vals[defaultsAttributeBecomeUser]; ok {
			v.becomeUser = val.(string)
			v.becomeUserIsSet = v.becomeUser != ""
		}
		if val, ok := vals[defaultsAttributeExtraVars]; ok {
			v.extraVars = mapFromTypeMap(val)
			v.extraVarsIsSet = len(v.extraVars) > 0
		}
		if val, ok := vals[defaultsAttributeForks]; ok {
			v.forks = val.(int)
			v.forksIsSet = v.forks > 0
		}
		if val, ok := vals[defaultsAttributeInventoryFile]; ok {
			v.inventoryFile = val.(string)
			v.inventoryFileIsSet = v.inventoryFile != ""
		}
		if val, ok := vals[defaultsAttributeLimit]; ok {
			v.limit = val.(string)
			v.limitIsSet = v.limit != ""
		}
		if val, ok := vals[defaultsAttributeVaultPasswordFile]; ok {
			v.vaultPasswordFile = val.(string)
			v.vaultPasswordFileIsSet = v.vaultPasswordFile != ""
		}
	}
	return v
}
