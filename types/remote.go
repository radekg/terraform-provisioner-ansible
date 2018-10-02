package types

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

// RemoteSettings represents remote settings.
type RemoteSettings struct {
	isRemoteInUse      bool
	useSudo            bool
	skipInstall        bool
	skipCleanup        bool
	installVersion     string
	localInstallerPath string
}

const (
	// default values:
	remoteDefaultUseSudo        = true
	remoteDefaultInstallVersion = "" // latest
	// attribute names:
	remoteAttributeUseSudo            = "use_sudo"
	remoteAttributeSkipInstall        = "skip_install"
	remoteAttributeSkipCleanup        = "skip_cleanup"
	remoteAttributeInstallVersion     = "install_version"
	remoteAttributeLocalInstallerPath = "local_installer_path"
)

// NewRemoteSchema returns a new remote schema.
func NewRemoteSchema() *schema.Schema {
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
				},
				remoteAttributeSkipCleanup: &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				remoteAttributeInstallVersion: &schema.Schema{
					Type:          schema.TypeString,
					Optional:      true,
					Default:       remoteDefaultInstallVersion,
					ConflictsWith: []string{fmt.Sprintf("remote.%s", remoteAttributeLocalInstallerPath)},
				},
				remoteAttributeLocalInstallerPath: &schema.Schema{
					Type:          schema.TypeString,
					Optional:      true,
					ValidateFunc:  vfPath,
					ConflictsWith: []string{fmt.Sprintf("remote.%s", remoteAttributeInstallVersion)},
				},
			},
		},
	}
}

// NewRemoteSettingsFromInterface reads Remote configuration from Terraform schema.
func NewRemoteSettingsFromInterface(i interface{}, ok bool) *RemoteSettings {
	v := &RemoteSettings{
		isRemoteInUse: false,
		useSudo:       remoteDefaultUseSudo,
	}
	if ok {
		vals := mapFromTypeSetList(i.(*schema.Set).List())
		v.isRemoteInUse = true
		v.useSudo = vals[remoteAttributeUseSudo].(bool)
		v.skipInstall = vals[remoteAttributeSkipInstall].(bool)
		v.skipCleanup = vals[remoteAttributeSkipCleanup].(bool)
		v.installVersion = vals[remoteAttributeInstallVersion].(string)
		v.localInstallerPath = vals[remoteAttributeLocalInstallerPath].(string)
	}
	return v
}

// IsRemoteInUse returns true remote provisioning is in use.
func (v *RemoteSettings) IsRemoteInUse() bool {
	return v.isRemoteInUse
}

// UseSudo returns true is sudo should be use, false otherwise.
func (v *RemoteSettings) UseSudo() bool {
	return v.useSudo
}

// SkipInstall returns true is Ansible installation should be skipped during remote provisioning, false otherwise.
func (v *RemoteSettings) SkipInstall() bool {
	return v.skipInstall
}

// SkipCleanup returns true is Ansible installation should be cleaned up during remote provisioning, false otherwise.
func (v *RemoteSettings) SkipCleanup() bool {
	return v.skipCleanup
}

// InstallVersion returns Ansible version to install, empty string means latest.
func (v *RemoteSettings) InstallVersion() string {
	return v.installVersion
}

// LocalInstallerPath returns a path to the custom Ansible installer.
func (v *RemoteSettings) LocalInstallerPath() string {
	return v.localInstallerPath
}
