package main

import (
	"context"
	"fmt"

	"github.com/radekg/terraform-provisioner-ansible/mode"
	"github.com/radekg/terraform-provisioner-ansible/types"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// $$ TODO: localhost in hosts

type provisioner struct {
	defaults           *types.Defaults
	plays              []*types.Play
	ansibleSSHSettings *types.AnsibleSSHSettings
	remote             *types.RemoteSettings
}

// Provisioner describes this provisioner configuration.
func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{
			"plays":                types.NewPlaySchema(),
			"defaults":             types.NewDefaultsSchema(),
			"remote":               types.NewRemoteSchema(),
			"ansible_ssh_settings": types.NewAnsibleSSHSettingsSchema(),
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

	if p.remote.IsRemoteInUse() {
		remoteMode, err := mode.NewRemoteMode(o, s, p.remote)
		if err != nil {
			o.Output(fmt.Sprintf("%+v", err))
			return err
		}
		return remoteMode.Run(p.plays)
	} else {

		localMode, err := mode.NewLocalMode(o, s)
		if err != nil {
			o.Output(fmt.Sprintf("%+v", err))
			return err
		}
		return localMode.Run(p.plays, p.ansibleSSHSettings)
	}

	return nil

}

func decodeConfig(d *schema.ResourceData) (*provisioner, error) {

	vRemoteSettings := types.NewRemoteSettingsFromInterface(d.GetOk("remote"))
	vAnsibleSSHSettings := types.NewAnsibleSSHSettingsFromInterface(d.GetOk("ansible_ssh_settings"))
	vDefaults := types.NewDefaultsFromInterface(d.GetOk("defaults"))

	plays := make([]*types.Play, 0)
	if rawPlays, ok := d.GetOk("plays"); ok {
		playSchema := types.NewPlaySchema()
		for _, iface := range rawPlays.([]interface{}) {
			plays = append(plays, types.NewPlayFromInterface(schema.NewSet(schema.HashResource(playSchema.Elem.(*schema.Resource)), []interface{}{iface}), vDefaults))
		}
	}
	return &provisioner{
		defaults:           vDefaults,
		remote:             vRemoteSettings,
		ansibleSSHSettings: vAnsibleSSHSettings,
		plays:              plays,
	}, nil
}
