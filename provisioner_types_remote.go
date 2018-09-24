package main

import (
	"github.com/hashicorp/terraform/helper/schema"
)

type remoteSettings struct {
	useSudo        bool
	skipInstall    bool
	skipCleanup    bool
	installVersion string
}

const (
	// default values:
	remoteDefaultUseSudo        = true
	remoteDefaultSkipInstall    = false
	remoteDefaultSkipCleanup    = false
	remoteDefaultInstallVersion = "" // latest
	// attribute names:
	remoteAttributeUseSudo        = "use_sudo"
	remoteAttributeSkipInstall    = "skip_install"
	remoteAttributeSkipCleanup    = "skip_cleanup"
	remoteAttributeInstallVersion = "install_version"
)

func newRemoteSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				remoteAttributeUseSudo: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  remoteDefaultUseSudo,
				},
				remoteAttributeSkipInstall: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  remoteDefaultSkipInstall,
				},
				remoteAttributeSkipCleanup: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
					Default:  remoteDefaultSkipCleanup,
				},
				remoteAttributeInstallVersion: &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  remoteDefaultInstallVersion,
				},
			},
		},
	}
}

func newRemoteSettingsFromInterface(i interface{}, ok bool) *remoteSettings {
	v := &remoteSettings{
		useSudo:        remoteDefaultUseSudo,
		skipInstall:    remoteDefaultSkipInstall,
		skipCleanup:    remoteDefaultSkipCleanup,
		installVersion: remoteDefaultInstallVersion,
	}
	if ok {
		vals := mapFromSetList(i.(*schema.Set).List())
		if val, ok := vals[remoteAttributeUseSudo]; ok {
			v.useSudo = val.(bool)
		}
		if val, ok := vals[remoteAttributeSkipInstall]; ok {
			v.skipInstall = val.(bool)
		}
		if val, ok := vals[remoteAttributeSkipCleanup]; ok {
			v.skipCleanup = val.(bool)
		}
		if val, ok := vals[remoteAttributeInstallVersion]; ok {
			v.installVersion = val.(string)
		}
	}
	return v
}
