// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Package ssh is a simple wrapper around an ssh.Client
// which implements utilities to be performed with a remote server.
package ssh

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Client is a wrapper type over a ssh connection
// that takes care of creating a channel and running
// commands in a single method.
type Client struct {
	client *ssh.Client
}

// NewClient returns a Client object by dialing
// to the local ssh agent.
func NewClient(ip string) (*Client, error) {
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
	return &Client{client: sshc}, nil
}

// RunCommand runs a given command in a new ssh session.
func (sshc *Client) RunCommand(cmd string) error {
	sess, err := sshc.client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	return sess.Run(cmd)
}

// Upload uploads a given src object to a given destination file.
func (sshc *Client) Upload(src io.Reader, dst string, sudo bool) error {
	if strings.ContainsAny(dst, `'\`) {
		// TODO: copied from load-test repo. Need to be improved
		// by using an actual sftp library.
		return fmt.Errorf("shell quoting not actually implemented. don't use weird paths")
	}

	sess, err := sshc.client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	sess.Stdin = src
	cmd := "cat > " + "'" + dst + "'"
	if sudo {
		cmd = fmt.Sprintf("sudo su -c %q", cmd)
	}
	return sess.Run(cmd)
}

// UploadFile uploads a given file path to a given destination file.
func (sshc *Client) UploadFile(src, dst string, sudo bool) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := sshc.Upload(f, false, "/opt/mattermost/bin/mattermost"); err != nil {
		return err
	}
	return nil
}

// Close closes the underlying connection.
func (sshc *Client) Close() error {
	return sshc.client.Close()
}
