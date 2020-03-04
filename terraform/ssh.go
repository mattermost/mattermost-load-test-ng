// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// TODO: move this to its separate package
func sshConn(ip string) (*ssh.Client, error) {
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
	return sshc, nil
}
