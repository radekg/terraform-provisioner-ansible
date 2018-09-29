package mode

// BastionHost encapsulates bastion host metadata.
type BastionHost struct {
	host    string
	port    int
	user    string
	pemFile string
}

// NewBastionHostFromConnectionInfo returns a bastion host instance extracted from connection info.
func NewBastionHostFromConnectionInfo(connInfo *connectionInfo, pemFile string) *BastionHost {
	bastionHost := &BastionHost{
		host:    "",
		port:    connInfo.Port,
		user:    connInfo.User,
		pemFile: pemFile,
	}
	if connInfo.BastionHost != "" {
		bastionHost.host = connInfo.BastionHost
		if connInfo.BastionUser != "" {
			bastionHost.user = connInfo.BastionUser
		}
		if connInfo.BastionPrivateKey != "" {
			bastionHost.pemFile = connInfo.BastionPrivateKey
		}
		if connInfo.BastionPort > 0 {
			bastionHost.port = connInfo.BastionPort
		}
	}
	return bastionHost
}

// Host returns bastion's host.
func (v *BastionHost) Host() string {
	return v.host
}

// Port returns bastion's port.
func (v *BastionHost) Port() int {
	return v.port
}

// User returns bastion's user.
func (v *BastionHost) User() string {
	return v.user
}

// PemFile returns bastion's PEM file.
func (v *BastionHost) PemFile() string {
	return v.pemFile
}
