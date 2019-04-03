package test

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/hashicorp/terraform/terraform"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// TestingSSHUser represents a user who can authenticate againstg the SSH server.
type TestingSSHUser struct {
	Username  string
	PublicKey string
}

// TestingSSHServerConfig is an SSH server configuration.
type TestingSSHServerConfig struct {
	ServerID           string
	HostKey            string
	HostPort           string
	AuthenticatedUsers []*TestingSSHUser
	Listeners          int
	Output             terraform.UIOutput
	LogPrintln         bool
	LocalMode          bool
}

// TestingSSHServer represents an instancde of the testing SFTP server.
type TestingSSHServer struct {
	config            *TestingSSHServerConfig
	t                 *testing.T
	lock              *sync.Mutex
	running           bool
	listener          net.Listener
	chanNotifications chan interface{}
	chanReady         chan struct{}
	chanStop          chan os.Signal
}

type exitStatusMsg struct {
	Status uint32
}

// NewTestingSSHServer creates an instance of the TestingSSHServer.
func NewTestingSSHServer(t *testing.T, c *TestingSSHServerConfig) *TestingSSHServer {
	return &TestingSSHServer{
		config:            c,
		t:                 t,
		lock:              &sync.Mutex{},
		chanNotifications: make(chan interface{}, 50),
		chanReady:         make(chan struct{}),
		chanStop:          make(chan os.Signal, 1),
	}
}

// Start starts the TestingSSHServer instance.
func (s *TestingSSHServer) Start() {
	s.lock.Lock()
	if !s.running {

		sshConfig := &ssh.ServerConfig{
			PublicKeyCallback: func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
				givenKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
				for _, user := range s.config.AuthenticatedUsers {
					if c.User() == user.Username {
						pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(user.PublicKey))
						if err != nil {
							return nil, fmt.Errorf("Public key for user %s is invalid", c.User())
						}
						pkey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pubKey)))
						if pkey != givenKey {
							return nil, fmt.Errorf("Public key rejected for user %s", c.User())
						}
						return nil, nil
					}
				}
				return nil, fmt.Errorf("No authorization for user %s", c.User())
			},
		}

		hostKey, err := ssh.ParsePrivateKey([]byte(s.config.HostKey))
		if err != nil {
			s.t.Fatalf("[%s] Failed to parse host key: %+v", s.config.ServerID, err)
		}
		sshConfig.AddHostKey(hostKey)

		listener, err := net.Listen("tcp", s.config.HostPort)
		if err != nil {
			s.t.Fatalf("[%s] Failed to listen for connection: %v", s.config.ServerID, err)
		}

		s.listener = listener
		s.running = true
		s.lock.Unlock()
		close(s.chanReady)

		for i := 0; i < s.config.Listeners; i++ {
			s.createConnectionServer(listener, sshConfig)
		}

		<-s.chanStop

	} else {
		s.lock.Unlock()
	}
}

// Stop stops the TestingSSHServer instance.
func (s *TestingSSHServer) Stop() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.running {
		s.logInfo("[%s] Stopping...", s.config.ServerID)
		s.chanStop <- syscall.SIGTERM
		s.listener.Close()
		s.running = false
	}
}

// ReadyNotify returns a channel that will be closed when the server is ready to serve client requests.
func (s *TestingSSHServer) ReadyNotify() <-chan struct{} {
	return s.chanReady
}

// Notifications returns a channel used for delivering the file SFTP event notifications.
func (s *TestingSSHServer) Notifications() <-chan interface{} {
	return s.chanNotifications
}

// ListeningHostPort returns the host port of an address underlying listener is bound on, or error if server is not started.
func (s *TestingSSHServer) ListeningHostPort() (host, port string, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.running {
		return net.SplitHostPort(s.listener.Addr().String())
	}
	return "", "", fmt.Errorf("The server is not started")
}

func (s *TestingSSHServer) createConnectionServer(listener net.Listener, config *ssh.ServerConfig) {
	go s.serveConnection(listener, config)
}

func (s *TestingSSHServer) serveConnection(listener net.Listener, config *ssh.ServerConfig) {

	s.t.Logf("[%s] Awaiting connection...", s.config.ServerID)

	nConn, err := listener.Accept()
	if err != nil {
		s.logInfo("[%s] failed to accept incoming connection: %v", s.config.ServerID, err)
		return
	}

	s.createConnectionServer(listener, config)

	sconn, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		s.logInfo("[%s] Failed to handshake: %v", s.config.ServerID, err)
		return
	}

	s.logInfo("[%s] SSH server established for %s", s.config.ServerID, sconn.User())

	// The incoming Request channel must be serviced.
	go ssh.DiscardRequests(reqs)

	// Service the incoming Channel channel.
	for newChannel := range chans {
		// Channels have a type, depending on the application level
		// protocol intended. In the case of an SFTP session, this is "subsystem"
		// with a payload string of "<length=4>sftp"
		s.logInfo("[%s] Incoming channel: %s", s.config.ServerID, newChannel.ChannelType())
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			s.logInfo("[%s] Unknown channel type: %s", s.config.ServerID, newChannel.ChannelType())
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			s.logInfo("[%s] ould not accept channel: %v", s.config.ServerID, err)
		}
		s.logInfo("[%s] Channel accepted: %v", s.config.ServerID, newChannel.ChannelType())

		// Sessions have out-of-band requests such as "shell",
		// "pty-req" and "env".  Here we handle only the
		// "subsystem" request.
		go func(in <-chan *ssh.Request) {
			for req := range in {
				s.logInfo("[%s] Request: %v", s.config.ServerID, req.Type)
				ok := false
				replyAndClose := false
				switch req.Type {
				case "subsystem":
					s.logInfo("[%s] Subsystem: %s", s.config.ServerID, req.Payload[4:])
					if string(req.Payload[4:]) == "sftp" {
						ok = true
					}
				case "pty-req":
					termLen := req.Payload[3]
					termEnv := string(req.Payload[4 : termLen+4])
					dimsBytes := req.Payload[termLen+4:]
					w := binary.BigEndian.Uint32(dimsBytes)
					h := binary.BigEndian.Uint32(dimsBytes[4:])
					s.logInfo("[%s] pty-req: size: %d@%d, env=%s", s.config.ServerID, w, h, termEnv)
					ok = true
				case "exec":

					ok = true

					command := string(req.Payload[4:])
					s.logInfo("[%s] exec %s", s.config.ServerID, command)
					msg := exitStatusMsg{
						Status: 0,
					}

					localCommand := command
					if strings.HasPrefix(command, "/bin/sh -c '") {
						localCommand = localCommand[12 : len(localCommand)-1]
					}

					execCommand := "/bin/sh"
					execCommandArgs := []string{"-c", localCommand}

					isScpCommand := strings.HasPrefix(localCommand, "scp -")

					if s.config.LocalMode {

						cmd := exec.Command(execCommand, execCommandArgs...)
						cmd.Stdout = channel
						cmd.Stderr = channel
						cmd.Stdin = channel

						s.logInfo("[%s] Executing local command: %s %s...", s.config.ServerID, execCommand, strings.Join(execCommandArgs, " "))
						err := cmd.Start()
						if err != nil {
							s.logInfo("[%s] failed to start command: %v...", s.config.ServerID, err)
							continue
						}

						go func() {
							_, err := cmd.Process.Wait()
							if err != nil {
								s.logInfo("[%s] failed to execute command: %v...", s.config.ServerID, err)
							}
							channel.SendRequest("exit-status", false, ssh.Marshal(msg))
							channel.Close()
							s.logInfo("[%s] command channel closed", s.config.ServerID)
						}()

					}

					if !s.config.LocalMode {
						s.logInfo("[%s] Emulating remote command: %s %s...", s.config.ServerID, execCommand, strings.Join(execCommandArgs, " "))
						if isScpCommand {
							benchStart := time.Now()
							var wg sync.WaitGroup
							wg.Add(1)
							go func() {
								defer wg.Done()
								buf := make([]byte, 8192)
								read, err := channel.Read(buf)
								if err != nil {
									s.logInfo("[%s] Reading SCP data failed: %v", s.config.ServerID, err)
									return
								}
								scpData := buf[0:read]
								benchDiff := time.Now().Sub(benchStart)
								s.logInfo("[%s] SCP DATA (arrived after %s):\n==================================\n%s\n==================================\n", s.config.ServerID, benchDiff.String(), string(scpData))
							}()
							req.Reply(true, nil)
							channel.Write([]byte{0})
							wg.Wait()
							channel.Write([]byte{0})
							replyAndClose = true
						}
						channel.SendRequest("exit-status", false, ssh.Marshal(msg))
					}

					go func() {
						s.chanNotifications <- NotificationCommandExecuted{
							Command: command,
						}
					}()

					ok = true
				case "shell":
					// We don't accept any commands (Payload), default shell only.
					s.logInfo("[%s] shell requested...", s.config.ServerID)
					if len(req.Payload) == 0 {
						ok = true
					}
				default:
					s.logInfo("[%s] Unsupported request type '%s'.", s.config.ServerID, req.Type)
					ok = true
				}
				req.Reply(ok, nil)
				if replyAndClose {
					sconn.Close()
				}
			}
		}(requests)

		server, err := sftp.NewServer(
			channel,
		)
		if err != nil {
			s.logInfo("[%s] sftp server could not be started: %v.", s.config.ServerID, err)
			continue
		}
		//server := sftp.NewRequestServer(channel, NewTestingSFTPFS(s.t, s.config, s.chanNotifications))
		if err := server.Serve(); err == io.EOF {
			server.Close()
			s.logInfo("[%s] sftp client exited session.", s.config.ServerID)
		} else if err != nil {
			s.logInfo("[%s] sftp server completed with error: %v", s.config.ServerID, err)
		}
	}
}

func (s *TestingSSHServer) logInfo(pattern string, args ...interface{}) {
	s.t.Logf(pattern, args...)
	s.config.Output.Output(fmt.Sprintf(pattern, args...))
	if s.config.LogPrintln {
		fmt.Println(fmt.Sprintf(pattern, args...))
	}
}
