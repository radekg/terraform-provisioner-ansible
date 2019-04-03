package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/radekg/terraform-provisioner-ansible/types"
)

var (
	// TestSSHHostKeyPrivate is an integration test host private key.
	TestSSHHostKeyPrivate = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAACFwAAAAdzc2gtcn
NhAAAAAwEAAQAAAgEAyD7WW7tbpNQR64DqVQFtKUUYTCSm731GZCK9ZGibrnqNXyEwtuWP
++1ooZZsj0ya8iDHBkgRph/vWKeD80vX8uPgW1gT1R6Ju2gWrRu8Y/410l8AhqtpIOQHaa
kojjeH7c1xoHbXa+4u6VfSUWXZjbAfpGL7MJhNnB3BsutqAJqN9ozcRLAbWrEGymVRhi8I
cob9AD0lq5i6OAlhlV81XbJHINdBIwXxM7w9nbkkIbH8EBZ44Qlbk9q02S2CJutadb+OwJ
Skp2qIEpiOQY4hFW100uzefHh+hb9F5ktZxXxWgAsuPvtUY19Xs6zNlLwasYBB/5ePOOgy
5zyqMXyvzkQxWns2tElxAib1rvQwj98cB9np6g9dEXVWmellosG4t7Ff49taJXjNPuswr0
lDORJws+SCQsamfn4ijHfTWyAB44eUKx78fS13xDjHMye5vVNcPvzhIwZal4CXLNLjVoBa
qOXuCtu9gtM1GgA2sXoUaYMN0k8zp0gk4ArfCP8BsvYxjJGucoFePiAHLjZr/RngfGRJBU
2D78e+CStQ0LzQYZUJJkoODxqpxXUND/dPmvehLd7zDYagi+lIJMDDh686zlQYdSi+uJQA
Yr924KEGzo4ZZalJbd+iF6h71zGE6EeY2UjbrWT+e24KxQLrKK23DkuYlIN2HtopYiFUM4
8AAAdIK5+5XiufuV4AAAAHc3NoLXJzYQAAAgEAyD7WW7tbpNQR64DqVQFtKUUYTCSm731G
ZCK9ZGibrnqNXyEwtuWP++1ooZZsj0ya8iDHBkgRph/vWKeD80vX8uPgW1gT1R6Ju2gWrR
u8Y/410l8AhqtpIOQHaakojjeH7c1xoHbXa+4u6VfSUWXZjbAfpGL7MJhNnB3BsutqAJqN
9ozcRLAbWrEGymVRhi8Icob9AD0lq5i6OAlhlV81XbJHINdBIwXxM7w9nbkkIbH8EBZ44Q
lbk9q02S2CJutadb+OwJSkp2qIEpiOQY4hFW100uzefHh+hb9F5ktZxXxWgAsuPvtUY19X
s6zNlLwasYBB/5ePOOgy5zyqMXyvzkQxWns2tElxAib1rvQwj98cB9np6g9dEXVWmellos
G4t7Ff49taJXjNPuswr0lDORJws+SCQsamfn4ijHfTWyAB44eUKx78fS13xDjHMye5vVNc
PvzhIwZal4CXLNLjVoBaqOXuCtu9gtM1GgA2sXoUaYMN0k8zp0gk4ArfCP8BsvYxjJGuco
FePiAHLjZr/RngfGRJBU2D78e+CStQ0LzQYZUJJkoODxqpxXUND/dPmvehLd7zDYagi+lI
JMDDh686zlQYdSi+uJQAYr924KEGzo4ZZalJbd+iF6h71zGE6EeY2UjbrWT+e24KxQLrKK
23DkuYlIN2HtopYiFUM48AAAADAQABAAACABh5ZaWsjpTkvpP0G6/sDrV+lmuoByc6PoI6
pL9C8dQvclvwKI4SHLkD2Uf2pKoXCNETJIAmCtItEQna34u691feditz5mij5N2c6TibLT
ljdpyRs/TBuoWkmStW23gWXWy5MWwVWlr8r4shirkcI6znm9ZxqpXT55hvIp+Fml0chsFd
kgZrJ8y72kKeg4pM8VFeIkoHLzV74za6Hs0s587d3UesR2/KaCKUnUtLt5jOsiodNQT8Kc
82aegpYcDz/whpMz9ia4VyxdLQBoMIpg6CZEbrYH4CFCSwrhBpoT075y6bLznPag8DNirl
sfK90t1i+a4Njhm8d4w/o+WMgcgSlhlQ5VfJknCVd5qpBXDJ5PoIkzomkQgphB4qllEipD
LQPHWON3z+dsMEjWox33TucN8ZiLmTs6kGP2XHYXWOTCnleAvXPOjS3Dld3cn1kYP18jIh
09ug+974WbvmzbDq6jhXQQ/20y1gqIBumRuacqmMzMw6Ge2su7w7PI+bsDdg2SElGdQc/S
fTt90bz6Gb03bf2OQb6MCdb1C44KAgE2IROXHWKnx5sHpTdoukqrIn3XoB274WHI8DIgmy
R/RG9/KNJ6nF2iADa6HY/U0mjkKwgNPIryO14lbLJHPq2yT3fhXkVocYnrUbsZFeFdPO29
RenW9LWCIHNsX7vYNxAAABADT0pfY4EEGbuHnBhMDAP05cudX0JNDOuZO3OfbIDPyXvB/M
iSpnAxTVi0ZHyKtwQI8+qT5XkMxLIkijMAiEBcfRwN1Sg2FEQVut4FmIhs/bAgC6jLRYtB
mLEWMjBz3TFd0HhDaFXDLy6lIb8O+6/kBCgNq7C1JgXupeiRdvsWhH6sf1YjW3aCkRNWMB
g/wX005F1h3xAAKQvpLFljvNn5dijffTAn74cB/i6zHL7bd0N3dUmxX+0tdWWm2FjPZEVG
x62XzHeu+twG5UoMropSHRDC3AfExuoyU/vqNrgEztfkYdpuGFGwqG/ysevYVTzsFtAHBa
zliKzwvGfbTrjv8AAAEBAPTBw5ssWkiZmLAEVHpfjZQnRnEVxqzyRWZOIIWVbU3by6SUZq
u1uzaoHJRgCJF/N9K8OkSIitl3qOYcF6iMe/dLqqFaW0YL2X15WT4grZN+XOi/k1oqbm6d
glRsjvxgc8Rza2xnC2oao/Yf121djzPdb0PM5ZUE2wTqWkKsbblVU3iiG495v1q21wVS+u
5SqF+NKduQ3K0EUfXRhHV9LeDkqphfxerMfoqJAcwu3R77Pt4wRfj8zUW1CrastlJc5/iE
7qzYtk5s+jbTHCxl7LqHQx4URVvEXMfEZZdVH7k9SdPtkcNeNWYsgeRMRpb9ooG5ivOoNQ
Eucw/otj59X1sAAAEBANFxo07K/COrr4YmVs/6PlXEHVdGNmAUCMPrBGTBtTK/63X3eQJ7
NjqcO917Gi4oJFgg9lHvQ5QJ7eLtFEP0k5z9znvttIMy/Is0zn/zbFnjwdkPmKaMmXa1ON
U2bwf/mksMRg6LTjJyWUaedi+YT/3I6D0iFRiWe8QcQe6GQD3x0e/2CI5JoHXPdF8sFocq
GLlldAzZhsF0KHTsIT6SmO3Vh6qq8Mew9PnSRRmH/7vwK7GlhCPokugtoEXQREw2/E3wLy
J65+euj1VTdZmMojcObiNEdjofW3aobbBmeZ4FyXMfwvAIEMqq7aLX1hEv4Ge0B5R6QGs5
HglAkbnyRt0AAAAOcmFkQG5vYW4ubG9jYWwBAgMEBQ==
-----END OPENSSH PRIVATE KEY-----`

	// TestSSHHostKeyPublic is an integration test host public key.
	TestSSHHostKeyPublic = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDIPtZbu1uk1BHrgOpVAW0pRRhMJKbvfUZkIr1kaJuueo1fITC25Y/77WihlmyPTJryIMcGSBGmH+9Yp4PzS9fy4+BbWBPVHom7aBatG7xj/jXSXwCGq2kg5AdpqSiON4ftzXGgdtdr7i7pV9JRZdmNsB+kYvswmE2cHcGy62oAmo32jNxEsBtasQbKZVGGLwhyhv0APSWrmLo4CWGVXzVdskcg10EjBfEzvD2duSQhsfwQFnjhCVuT2rTZLYIm61p1v47AlKSnaogSmI5BjiEVbXTS7N58eH6Fv0XmS1nFfFaACy4++1RjX1ezrM2UvBqxgEH/l4846DLnPKoxfK/ORDFaeza0SXECJvWu9DCP3xwH2enqD10RdVaZ6WWiwbi3sV/j21oleM0+6zCvSUM5EnCz5IJCxqZ+fiKMd9NbIAHjh5QrHvx9LXfEOMczJ7m9U1w+/OEjBlqXgJcs0uNWgFqo5e4K272C0zUaADaxehRpgw3STzOnSCTgCt8I/wGy9jGMka5ygV4+IAcuNmv9GeB8ZEkFTYPvx74JK1DQvNBhlQkmSg4PGqnFdQ0P90+a96Et3vMNhqCL6UgkwMOHrzrOVBh1KL64lABiv3bgoQbOjhllqUlt36IXqHvXMYToR5jZSNutZP57bgrFAusorbcOS5iUg3Ye2iliIVQzjw== rad@noan.local`

	// TestSSHUserKeyPrivate is an integration test user private key.
	TestSSHUserKeyPrivate = `-----BEGIN OPENSSH PRIVATE KEY-----
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
-----END OPENSSH PRIVATE KEY-----`

	// TestSSHUserKeyPublic is an integration test user public key.
	TestSSHUserKeyPublic = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC4xdZRtDIqh/TYbB2U4ZzyIiDoBBd0f9BIE39EB43AD1mlkrex9afxQKpfBNoR3Bvd6j89Ucb3Clklnj1KdEgxsC0kN1mewABkdAt6+5xpOnOl9ZgnUE3Ib959/puMlG8d1G/evWBo2DzVgk6uP/2A7+cjUsTvVM5t/DXeOniebJHDlUZse9FHoltPvL1Ro/ug2o4tZWXucf2PDqD2+aDfzXWRAZiIluXOo8j/by3/8K+DLODwhAPKZ9h9es0wSNWkC9UhOc6/2iY32PaKn9PIz3cEKfl/YHp6GA8gdJnR5KLLxYra6vxGoVSrvqr32Y1K+ktYVmv7V/TrpkwoXqFx540gO1t71PuYxAsyzENZny/L7MxK8cS+9ND4xYQOE8ImIcwk+52Jy5/H7g0M4To0Xjla6FcnCjvgDsXiEH/JRPfGCuyDCZPqVOUv9B0lLJTeKEipsysIAZsAN4kM5Nomv/9DQpaEOIbX9PihEm0RYzVvqvyRBOtvshn5rnLGJKZGyJw2Hr4wSJPgpaYTPnsjTAZ8ZKOiUAAvqF8qB/7mZ8p4mXFGJJMw/lk8NK+2/vH88OpnmWGiOs9rA9BnyWbc4rmpMj/XZBt7oaxyLWmeVfYzeKc6VadjrN21Yv2qSysngvX5BodDZ+Ql6T8Dvd8KeNnoezJL/xOM3cGm74uGwQ== rad@noan.local`

	// CommandWaitTimeoutDuration specifies how long the test awaits for a notification from the SSH server.
	// External CI (Travis, CircleCI) requires long timeouts, locally these will always be much faster.
	CommandWaitTimeoutDuration = time.Duration(60) * time.Second

	// ServerWaitTimeoutDuration specifies how long the test awaits for the SSH server to start.
	ServerWaitTimeoutDuration = time.Duration(10) * time.Second
)

// CommandTest tests an SSH server output channel for a command.
func CommandTest(t *testing.T, sshServer *TestingSSHServer, commandPrefix string) {
	select {
	case event := <-sshServer.Notifications():
		switch tevent := event.(type) {
		case NotificationCommandExecuted:
			if !strings.HasPrefix(tevent.Command, commandPrefix) {
				t.Fatalf("Expected a command starting with '%s' received: '%s'", commandPrefix, tevent.Command)
			}
		default:
			t.Fatal("Expected a command execution but received", tevent)
		}
	case <-time.After(CommandWaitTimeoutDuration):
		t.Fatal("Excepted a notification from the SSH server.")
	}
}

// CreateTempAnsibleBootstrapDir creates a temp Ansible bootstrap directory.
func CreateTempAnsibleBootstrapDir(t *testing.T) string {
	tmp, err := ioutil.TempDir("", ".temp-bootstrap")
	if err != nil {
		t.Fatal("Expected a temp temp bootstrap dir to be created", err)
	}
	return tmp
}

// CreateTempAnsibleDataDirectory creates a temp Ansible data directory.
func CreateTempAnsibleDataDirectory(t *testing.T) string {
	tempAnsibleDataDir, err := ioutil.TempDir("", ".temp-ansible-data")
	if err != nil {
		t.Fatal("Expected a temp data dir to be created", err)
	}
	return tempAnsibleDataDir
}

// CreateTempAnsibleRemoteTmpDir creates a temp Ansible remote_tmp directory.
func CreateTempAnsibleRemoteTmpDir(t *testing.T) string {
	tmp, err := ioutil.TempDir("", ".temp-remote-tmp")
	if err != nil {
		t.Fatal("Expected a temp remote_tmp dir to be created", err)
	}
	return tmp
}

// GetConfiguredAndRunningSSHServer returns a running instance of a *TestingSSHServer.
func GetConfiguredAndRunningSSHServer(t *testing.T, serverID string, localMode bool, instanceState *terraform.InstanceState, output terraform.UIOutput) *TestingSSHServer {
	authUser := &TestingSSHUser{
		Username:  instanceState.Ephemeral.ConnInfo["user"],
		PublicKey: TestSSHUserKeyPublic,
	}
	sshConfig := &TestingSSHServerConfig{
		ServerID:           serverID,
		HostKey:            TestSSHHostKeyPrivate,
		HostPort:           fmt.Sprintf("%s:%s", instanceState.Ephemeral.ConnInfo["host"], instanceState.Ephemeral.ConnInfo["port"]),
		AuthenticatedUsers: []*TestingSSHUser{authUser},
		Listeners:          5,
		Output:             output,
		LogPrintln:         true,
		LocalMode:          localMode,
	}
	sshServer := NewTestingSSHServer(t, sshConfig)
	go sshServer.Start()

	select {
	case <-sshServer.ReadyNotify():
	case <-time.After(ServerWaitTimeoutDuration):
		t.Fatal("Expected the TestingSSHServer to be running.")
	}

	// we need to update the instance info with the address the SSH server is bound on:
	h, p, err := sshServer.ListeningHostPort()
	if err != nil {
		t.Fatal("Expected the SSH server to return an address it is bound on but got an error", err)
	}

	// set connection details based on where the SSH server is bound:
	instanceState.Ephemeral.ConnInfo["host"] = h
	instanceState.Ephemeral.ConnInfo["port"] = p

	return sshServer
}

// GetCurrentUser returns current user.
func GetCurrentUser(t *testing.T) *user.User {
	user, err := user.Current()
	if err != nil {
		t.Fatal("Expectd to fetch the current user but received an error", err)
	}
	return user
}

// GetDefaultSettingsForUser returns default settings in the context of a given user.
func GetDefaultSettingsForUser(t *testing.T, user *user.User) *types.Defaults {
	defaultSettings := map[string]interface{}{
		"become_method": "sudo",
		"become_user":   user.Username,
	}
	return types.NewDefaultsFromMapInterface(defaultSettings, true)
}

// GetNewPlay returns *types.Play from a raw map using th default settings.
func GetNewPlay(t *testing.T, raw map[string]interface{}, defaultSettings *types.Defaults) *types.Play {
	return types.NewPlayFromMapInterface(raw, defaultSettings)
}

// GetNewRemoteSettings returns *types.RemoteSettings from a raw map.
func GetNewRemoteSettings(t *testing.T, raw map[string]interface{}) *types.RemoteSettings {
	return types.NewRemoteSettingsFromMapInterface(raw, true)
}

// GetNewSSHInstanceState returns a new instance of *teraform.InstanceState for a given SSH username.
func GetNewSSHInstanceState(t *testing.T, sshUsername string) *terraform.InstanceState {
	return &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":        "ssh",
				"user":        sshUsername,
				"host":        "127.0.0.1",
				"port":        "0", // will be set later
				"private_key": TestSSHUserKeyPrivate,
				"host_key":    TestSSHHostKeyPublic,
			},
		},
	}
}

// GetPlayModuleSchema returns a Terraform *schema.ResourceData for a given module name.
func GetPlayModuleSchema(t *testing.T, moduleName string) *schema.ResourceData {
	playModuleEntity := map[string]interface{}{
		"module": []map[string]interface{}{
			map[string]interface{}{
				"module": moduleName,
			},
		},
		"playbook": []map[string]interface{}{},
	}
	playEntitySchemas := map[string]*schema.Schema{
		"module":   types.NewModuleSchema(),
		"playbook": types.NewPlaybookSchema(),
	}
	return schema.TestResourceDataRaw(t, playEntitySchemas, playModuleEntity)
}

// GetPlayPlaybookSchema returns a Terraform *schema.ResourceData for a given playbook file path.
func GetPlayPlaybookSchema(t *testing.T, playbookFilePath string) *schema.ResourceData {
	playPlaybookEntity := map[string]interface{}{
		"playbook": []map[string]interface{}{
			map[string]interface{}{
				"file_path": playbookFilePath,
			},
		},
		"module": []map[string]interface{}{},
	}
	playEntitySchemas := map[string]*schema.Schema{
		"module":   types.NewModuleSchema(),
		"playbook": types.NewPlaybookSchema(),
	}
	return schema.TestResourceDataRaw(t, playEntitySchemas, playPlaybookEntity)
}

// WriteTempVaultIDFile creates and writes a temporrary Vault ID password file.
func WriteTempVaultIDFile(t *testing.T, password string) string {
	tempVaultIDFile, err := ioutil.TempFile("", ".temp-vault-id")
	if err != nil {
		t.Fatal("Expected a temp vault id file to be crated", err)
	}
	tempVaultIDFileToWrite, err := os.OpenFile(tempVaultIDFile.Name(), os.O_RDWR, 0644)
	if err != nil {
		t.Fatal("Expected a temp vault id file to be writable", err)
	}
	tempVaultIDFileToWrite.WriteString(password)
	tempVaultIDFileToWrite.Close()
	return tempVaultIDFile.Name()
}

// WriteTempPlaybook writes a temp playbook.
func WriteTempPlaybook(t *testing.T, dirpath string) string {

	playbooksDir := filepath.Join(dirpath, "playbooks")
	playbookFilePath := filepath.Join(playbooksDir, "playbook-integration-test.yml")

	roleTasksDir := filepath.Join(playbooksDir, "roles", "integration_test", "tasks")
	roleTasksMainFilePath := filepath.Join(roleTasksDir, "main.yml")

	playbookFileContents := `---
- hosts: all
  become: no
  roles:
    - integration_test
`

	roleTasksFileContents := `- name: test task
  command: echo test-task
`

	if err := os.MkdirAll(playbooksDir, os.ModePerm); err != nil {
		t.Fatal("Expected the playbooks directory to be created under temp directory", err)
	}

	playbookFile, err := os.Create(playbookFilePath)
	if err != nil {
		t.Fatal("Expected a temp playbook file to be created", err)
	}
	playbookFile.WriteString(playbookFileContents)
	playbookFile.Close()

	if err := os.MkdirAll(roleTasksDir, os.ModePerm); err != nil {
		t.Fatal("Expected the integration_test directory to be created under temp playbooks directory", err)
	}

	tasksFile, err := os.Create(roleTasksMainFilePath)
	if err != nil {
		t.Fatal("Expected a temp tasks file to be created", err)
	}
	tasksFile.WriteString(roleTasksFileContents)
	tasksFile.Close()

	return playbookFilePath
}
