package provisioner

import (
	"context"
	"fmt"

	"github.com/radekg/terraform-provisioner-ansible/v2/mode"
	"github.com/radekg/terraform-provisioner-ansible/v2/types"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

type terraformVersion int

const (
	terraform011 terraformVersion = iota
	terraform012
)

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

	validPlaysCount := 0

	// Workaround to enable backward compatibility
	var computedTfVersion terraformVersion

	if plays, hasPlays := c.Get("plays"); hasPlays {

		var sanitizedPlays []interface{}

		switch plays.(type) {
		case []interface{}: // Terraform 0.12.x
			computedTfVersion = terraform012
			sanitizedPlays = plays.([]interface{})
		case []map[string]interface{}: // Terraform 0.11.x
			computedTfVersion = terraform011
			for _, v := range plays.([]map[string]interface{}) {
				sanitizedPlays = append(sanitizedPlays, v)
			}
		default:
			es = append(es, fmt.Errorf("could not establish Terrafrom version from plays type: %T", plays))
			return ws, es // return early
		}

		for _, rawVPlay := range sanitizedPlays {
			vPlay := rawVPlay.(map[string]interface{})

			currentErrorCount := len(es)

			vPlaybook, playHasPlaybook := vPlay["playbook"]
			_, playHasModule := vPlay["module"]
			_, playHasGalaxyInstall := vPlay["galaxy_install"]

			if types.HasMoreThanOneTrue([]bool{playHasPlaybook, playHasModule, playHasGalaxyInstall}...) {
				es = append(es, fmt.Errorf("play can have only one of: galaxy_install, playbook or module"))
			} else if !playHasPlaybook && !playHasModule && !playHasGalaxyInstall {
				es = append(es, fmt.Errorf("galaxy_install, playbook or module must be set"))
			} else {

				if playHasPlaybook {

					var rolesPath []interface{}
					var hasRolesPath bool

					switch computedTfVersion {
					case terraform012:
						vPlaybookTyped := vPlaybook.([]interface{})
						rolesPath, hasRolesPath = vPlaybookTyped[0].(map[string]interface{})["roles_path"].([]interface{})
					case terraform011:
						vPlaybookTyped := vPlaybook.([]map[string]interface{})
						rolesPath, hasRolesPath = vPlaybookTyped[0]["roles_path"].([]interface{})
					default:
						es = append(es, fmt.Errorf("unsupported Terrafrom version detected: %d", computedTfVersion))
						return ws, es // return early
					}

					if hasRolesPath {
						for _, singlePath := range rolesPath {
							vws, ves := types.VfPathDirectory(singlePath, "roles_path")

							for _, w := range vws {
								ws = append(ws, w)
							}
							for _, e := range ves {
								es = append(es, e)
							}
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
	}

	localMode, err := mode.NewLocalMode(o, s)
	if err != nil {
		o.Output(fmt.Sprintf("%+v", err))
		return err
	}
	return localMode.Run(p.plays, p.ansibleSSHSettings)

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
