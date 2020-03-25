// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Package ssh is a simple wrapper around an ssh.Client
// which implements utilities to be performed with a remote server.
package ssh

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// ExtAgent is a wrapper type over agent.ExtendedAgent
// provding a method to return a Client.
type ExtAgent struct {
	agent agent.ExtendedAgent
}

// Client is a wrapper type over a ssh connection
// that takes care of creating a channel and running
// commands in a single method.
type Client struct {
	client *ssh.Client
}

// NewAgent connects to the local ssh agent and validates
// that it has atleast one key added. It returns the agent
// if everything looks good.
func NewAgent() (*ExtAgent, error) {
	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, err
	}

	extAgent := agent.NewClient(conn)
	// Check if keys are added.
	keys, err := extAgent.List()
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, errors.New("no identities have been added to the agent. Please run ssh-add")
	}

	return &ExtAgent{agent: extAgent}, nil
}

// NewClient returns a Client object by dialing
// the ssh agent.
func (ea *ExtAgent) NewClient(ip string) (*Client, error) {
	config := &ssh.ClientConfig{
		User: "ubuntu",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(ea.agent.Signers),
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

// StartCommand starts a given command in a new ssh session. Unlike RunCommand
// this command does not wait command to finish. This is needed for running
// commands in the background.
func (sshc *Client) StartCommand(cmd string) error {
	sess, err := sshc.client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	return sess.Start(cmd)
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
	return sshc.Upload(f, "/opt/mattermost/bin/mattermost", false)
}

// Close closes the underlying connection.
func (sshc *Client) Close() error {
	return sshc.client.Close()
}
