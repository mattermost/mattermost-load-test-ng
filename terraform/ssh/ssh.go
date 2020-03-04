// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package ssh

import (
	"io"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Conn is a wrapper type over a ssh connection
// that takes care of creating a channel and running
// commands in a single method.
type Conn struct {
	client *ssh.Client
}

// NewConn returns a Conn object by dialing
// to the local ssh agent.
func NewConn(ip string) (*Conn, error) {
	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: "ubuntu",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(agent.NewClient(conn).Signers),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshc, err := ssh.Dial("tcp", ip+":22", config)
	if err != nil {
		return nil, err
	}
	return &Conn{client: sshc}, nil
}

// RunCommand runs a given command in a new ssh session.
func (sshc *Conn) RunCommand(cmd string) error {
	sess, err := sshc.client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	return sess.Run(cmd)
}

// Upload uploads a given src object to a given destination file.
func (sshc *Conn) Upload(src io.Reader, dst string) error {
	sess, err := sshc.client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	sess.Stdin = src
	return sess.Run("cat > " + shellQuote(dst))
}

// Close closes the underlying connection.
func (sshc *Conn) Close() error {
	return sshc.client.Close()
}

func shellQuote(s string) string {
	if strings.ContainsAny(s, `'\`) {
		// TODO: copied from load-test repo. Need to be improved
		// by using an actual sftp library.
		panic("shell quoting not actually implemented. don't use weird paths")
	}
	return "'" + s + "'"
}
