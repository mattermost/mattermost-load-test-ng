// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Package ssh is a simple wrapper around an ssh.Client
// which implements utilities to be performed with a remote server.
package ssh

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
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
	IP     string
}

// NewAgent connects to the local ssh agent and validates
// that it has at least one key added. It returns the agent
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

// NewClientWithPort returns a Client object by dialing
// the ssh agent on the provided port
func (ea *ExtAgent) NewClientWithPort(ip, port, user string) (*Client, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(ea.agent.Signers),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshc, err := ssh.Dial("tcp", ip+port, config)
	if err != nil {
		return nil, err
	}
	return &Client{client: sshc, IP: ip}, nil
}

// NewClient returns a Client object by dialing
// the ssh agent on port 22
func (ea *ExtAgent) NewClient(user, ip string) (*Client, error) {
	return ea.NewClientWithPort(ip, ":22", user)
}

// RunCommand runs a given command in a new ssh session.
func (sshc *Client) RunCommand(cmd string) ([]byte, error) {
	sess, err := sshc.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	return sess.CombinedOutput(cmd)
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
func (sshc *Client) Upload(src io.Reader, dst string, sudo bool) ([]byte, error) {
	sftpClient, err := sftp.NewClient(sshc.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// For sudo operations and to avoid permission errors we need to upload the file to the temp folder and then
	// move it to the destination with sudo using a command.
	if sudo {
		randomBytes := make([]byte, 8)
		if _, err := rand.Read(randomBytes); err != nil {
			return nil, fmt.Errorf("failed to generate random filename: %w", err)
		}
		tempDst := filepath.Join("/tmp", "upload_"+hex.EncodeToString(randomBytes)+".tmp")
		dstFile, err := sftpClient.Create(tempDst)
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file '%s': %w", tempDst, err)
		}

		_, err = io.Copy(dstFile, src)
		dstFile.Close()
		if err != nil {
			sftpClient.Remove(tempDst)
			return nil, fmt.Errorf("failed to upload file: %w", err)
		}

		// Move temp file to final destination with sudo
		cmd := fmt.Sprintf("sudo mv %q %q", tempDst, dst)
		return sshc.RunCommand(cmd)
	}

	// Direct upload without sudo
	dstFile, err := sftpClient.Create(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, src)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	return nil, nil
}

// UploadFile uploads a given file path to a given destination file.
func (sshc *Client) UploadFile(src, dst string, sudo bool) ([]byte, error) {
	f, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return sshc.Upload(f, dst, sudo)
}

// Download downloads a given src remote filepath to a given dst writer.
func (sshc *Client) Download(src string, dst io.Writer, sudo bool) error {
	if strings.ContainsAny(src, `'\`) {
		// TODO: copied from load-test repo. Need to be improved
		// by using an actual sftp library.
		return errors.New("shell quoting not actually implemented. don't use weird paths")
	}

	sess, err := sshc.client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	sess.Stdout = dst

	cmd := fmt.Sprintf("cat '%s'", src)
	if sudo {
		cmd = fmt.Sprintf("sudo su -c %q", cmd)
	}

	if err := sess.Run(cmd); err != nil {
		return err
	}

	return nil
}

// Close closes the underlying connection.
func (sshc *Client) Close() error {
	return sshc.client.Close()
}

// DialContextF returns the underlying client's DialContext function
func (sshc *Client) DialContextF() func(context.Context, string, string) (net.Conn, error) {
	return sshc.client.DialContext
}
