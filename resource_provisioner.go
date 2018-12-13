package main

import (
	"context"
	"fmt"

	"github.com/radekg/terraform-provisioner-ansible/mode"
	"github.com/radekg/terraform-provisioner-ansible/types"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

type provisioner struct {
	defaults           *types.Defaults
	plays              []*types.Play
	ansibleSSHSettings *types.AnsibleSSHSettings
	remote             *types.RemoteSettings
	globalPlays        []*types.Play
}

// Provisioner describes this provisioner configuration.
func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{
			"plays":                types.NewPlaySchema(),
			"defaults":             types.NewDefaultsSchema(),
			"remote":               types.NewRemoteSchema(),
			"ansible_ssh_settings": types.NewAnsibleSSHSettingsSchema(),
			"global_plays":         types.NewPlaySchema(),
		},
		ValidateFunc: validateFn,
		ApplyFunc:    applyFn,
	}
}

func validatePlays(play map[string]interface{}, validPlaysCount *int, ws *[]string, es *[]error) {
	currentErrorCount := len(*es)

	_, hasModule := play["module"]
	if p, ok := play["playbook"]; ok {
		if hasModule {
			*es = append(*es, fmt.Errorf("a play cannot have both a playbook and module"))
		}
		//schema supports multiple playbooks within same play, but only implement the first playbook
		var firstPlaybook map[string]interface{}
		switch p.(type) {
		case *schema.Set:
			firstPlaybook = p.(*schema.Set).List()[0].(map[string]interface{})
		case []map[string]interface{}:
			firstPlaybook = p.([]map[string]interface{})[0]
		default:
			firstPlaybook = p.([]interface{})[0].(map[string]interface{})
		}

		if rolesPath, hasRolesPath := firstPlaybook["roles_path"]; hasRolesPath {
			for _, singlePath := range rolesPath.([]interface{}) {
				vws, ves := types.VfPathDirectory(singlePath, "roles_path")

				for _, w := range vws {
					*ws = append(*ws, w)
				}
				for _, e := range ves {
					*es = append(*es, e)
				}
			}
		}
	}

	if currentErrorCount == len(*es) {
		*validPlaysCount++
	}
	return
}
func validateFn(c *terraform.ResourceConfig) (ws []string, es []error) {

	defer func() {
		if r := recover(); r != nil {
			es = append(es, fmt.Errorf("error while validating the provisioner, reason: %+v", r))
		}
	}()
	validPlaysCount := 0
	validGlobalPlaysCount := 0

	if p, hasPlays := c.Get("plays"); hasPlays {
		switch p.(type) {
		case *schema.Set:
			for _, vPlay := range p.(*schema.Set).List() {
				validatePlays(vPlay.(map[string]interface{}), &validPlaysCount, &ws, &es)
			}
		case []map[string]interface{}:
			for _, vPlay := range p.([]map[string]interface{}) {
				validatePlays(vPlay, &validPlaysCount, &ws, &es)
			}
		default:
			for _, vPlay := range p.([]interface{}) {
				validatePlays(vPlay.(map[string]interface{}), &validPlaysCount, &ws, &es)
			}
		}
	}

	if p, hasGlobalPlays := c.Get("global_plays"); hasGlobalPlays {
		switch p.(type) {
		case *schema.Set:
			for _, vPlay := range p.(*schema.Set).List() {
				validatePlays(vPlay.(map[string]interface{}), &validGlobalPlaysCount, &ws, &es)
			}
		case []map[string]interface{}:
			for _, vPlay := range p.([]map[string]interface{}) {
				validatePlays(vPlay, &validGlobalPlaysCount, &ws, &es)
			}
		default:
			for _, vPlay := range p.([]interface{}) {
				validatePlays(vPlay.(map[string]interface{}), &validGlobalPlaysCount, &ws, &es)
			}
		}
	}

	if validPlaysCount == 0 && validGlobalPlaysCount == 0 {
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

	if p.remote.IsRemoteInUse() {
		remoteMode, err := mode.NewRemoteMode(o, s, p.remote)
		if err != nil {
			o.Output(fmt.Sprintf("%+v", err))
			return err
		}
		return remoteMode.Run(p.plays)
	}

	localMode, err := mode.NewLocalMode(o, s)
	if err != nil {
		o.Output(fmt.Sprintf("%+v", err))
		return err
	}
	return localMode.Run(p.plays, p.globalPlays, p.ansibleSSHSettings)

}

func decodeConfig(d *schema.ResourceData) (*provisioner, error) {

	vRemoteSettings := types.NewRemoteSettingsFromInterface(d.GetOk("remote"))
	vAnsibleSSHSettings := types.NewAnsibleSSHSettingsFromInterface(d.GetOk("ansible_ssh_settings"))
	vDefaults := types.NewDefaultsFromInterface(d.GetOk("defaults"))

	plays := make([]*types.Play, 0)
	if rawPlays, ok := d.GetOk("plays"); ok {
		playSchema := types.NewPlaySchema()
		for _, iface := range rawPlays.(*schema.Set).List() {
			plays = append(plays, types.NewPlayFromInterface(schema.NewSet(schema.HashResource(playSchema.Elem.(*schema.Resource)), []interface{}{iface}), vDefaults))
		}

	}

	globalPlays := make([]*types.Play, 0)

	if rawPlays, ok := d.GetOk("global_plays"); ok {
		playSchema := types.NewPlaySchema()
		for _, iface := range rawPlays.(*schema.Set).List() {
			globalPlays = append(globalPlays, types.NewPlayFromInterface(schema.NewSet(schema.HashResource(playSchema.Elem.(*schema.Resource)), []interface{}{iface}), vDefaults))
		}
	}
	return &provisioner{
		defaults:           vDefaults,
		remote:             vRemoteSettings,
		ansibleSSHSettings: vAnsibleSSHSettings,
		plays:              plays,
		globalPlays:        globalPlays,
	}, nil
}
