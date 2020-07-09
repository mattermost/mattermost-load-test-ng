// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/spf13/cobra"
)

type collectInfo struct {
	instance string
	src      string
	compress bool
}

type file struct {
	name string
	data []byte
}

func saveCollection(files []file) error {
	u, err := user.Current()
	if err != nil {
		return err
	}

	uid, err := strconv.ParseInt(u.Uid, 10, 32)
	if err != nil {
		return err
	}

	gid, err := strconv.ParseInt(u.Gid, 10, 32)
	if err != nil {
		return err
	}

	name := fmt.Sprintf("collection_%d", time.Now().Unix())
	filename := fmt.Sprintf("./%s.tar", name)
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %w", err)
	}
	defer f.Close()

	tw := tar.NewWriter(f)
	hdr := &tar.Header{
		Name:    name + "/",
		Mode:    0755,
		Uid:     int(uid),
		Gid:     int(gid),
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}

	for _, file := range files {
		hdr = &tar.Header{
			Name:    name + "/" + file.name,
			Mode:    0600,
			Size:    int64(len(file.data)),
			Uid:     int(uid),
			Gid:     int(gid),
			ModTime: time.Now(),
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if _, err := tw.Write(file.data); err != nil {
			return err
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}

	fmt.Printf("%s generated.\n", filename)

	return nil
}

func createClients(output *terraform.Output) (map[string]*ssh.Client, error) {
	extAgent, err := ssh.NewAgent()
	if err != nil {
		return nil, err
	}

	clients := make(map[string]*ssh.Client)
	if output.HasProxy() {
		sshc, err := extAgent.NewClient(output.Proxy.Value[0].PublicIP)
		if err != nil {
			return nil, fmt.Errorf("error in getting ssh connection %w", err)
		}
		clients["proxy"] = sshc
	}

	for i, instance := range output.Instances.Value {
		sshc, err := extAgent.NewClient(instance.PublicIP)
		if err != nil {
			return nil, fmt.Errorf("error in getting ssh connection %w", err)
		}
		clients[fmt.Sprintf("app%d", i)] = sshc
	}

	for i, agent := range output.Agents.Value {
		sshc, err := extAgent.NewClient(agent.PublicIP)
		if err != nil {
			return nil, fmt.Errorf("error in getting ssh connection %w", err)
		}
		clients[fmt.Sprintf("agent%d", i)] = sshc
		if i == 0 {
			clients["coordinator"] = sshc
		}
	}

	return clients, nil
}

func RunCollectCmdF(cmd *cobra.Command, args []string) error {
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		return errors.New("ssh agent not running. Please run eval \"$(ssh-agent -s)\" and then ssh-add")
	}

	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t := terraform.New(config)

	output, err := t.Output()
	if err != nil {
		return err
	}

	if !output.HasAppServers() {
		return errors.New("no active deployment found")
	}

	clients, err := createClients(output)
	if err != nil {
		return err
	}

	var collection []collectInfo
	addInfo := func(instance, src string, compress bool) {
		collection = append(collection, collectInfo{
			instance,
			src,
			compress,
		})
	}
	for name := range clients {
		switch {
		case name == "proxy":
			addInfo(name, "/var/log/nginx/error.log", true)
			addInfo(name, "/etc/nginx/nginx.conf", false)
			addInfo(name, "/etc/nginx/sites-enabled/mattermost", false)
		case strings.HasPrefix(name, "app"):
			addInfo(name, "/opt/mattermost/logs/mattermost.log", true)
			addInfo(name, "/opt/mattermost/config/config.json", false)
		case strings.HasPrefix(name, "agent"):
			addInfo(name, "/home/ubuntu/agent.log", true)
		case name == "coordinator":
			addInfo(name, "/home/ubuntu/mattermost-load-test-ng/ltcoordinator.log", true)
			addInfo(name, "/home/ubuntu/mattermost-load-test-ng/config/config.json", false)
			addInfo(name, "/home/ubuntu/mattermost-load-test-ng/config/coordinator.json", false)
			addInfo(name, "/home/ubuntu/mattermost-load-test-ng/config/simplecontroller.json", false)
			addInfo(name, "/home/ubuntu/mattermost-load-test-ng/config/simulcontroller.json", false)
			continue
		}
		addInfo(name, "dmesg", false)
	}

	var wg sync.WaitGroup
	filesChan := make(chan file, len(collection))
	wg.Add(len(collection))
	for _, info := range collection {
		go func(info collectInfo) {
			defer wg.Done()

			sshc := clients[info.instance]

			var downloadPath string

			if !filepath.IsAbs(info.src) {
				cmd := info.src
				info.src = fmt.Sprintf("/tmp/%s.log", info.src)
				cmd = fmt.Sprintf("%s > %s", cmd, info.src)
				if _, err := sshc.RunCommand(cmd); err != nil {
					fmt.Printf("failed to run cmd %q: %s\n", cmd, err)
					return
				}
			}

			if info.compress {
				downloadPath = fmt.Sprintf("/tmp/%s.xz", filepath.Base(info.src))
				cmd := fmt.Sprintf("cat %s | xz -2 -T4 > %s", info.src, downloadPath)
				if _, err := sshc.RunCommand(cmd); err != nil {
					fmt.Printf("failed to run cmd %q: %s\n", cmd, err)
					return
				}
			}

			if downloadPath == "" {
				downloadPath = info.src
			}

			var b bytes.Buffer
			if err := sshc.Download(downloadPath, &b, false); err != nil {
				fmt.Printf("failed to download file %q: %s\n", downloadPath, err)
				return
			}

			fmt.Printf("collected %s from %s instance\n", filepath.Base(downloadPath), info.instance)

			file := file{
				name: fmt.Sprintf("%s_%s", info.instance, filepath.Base(downloadPath)),
				data: b.Bytes(),
			}

			filesChan <- file
		}(info)
	}

	wg.Wait()

	numFiles := len(filesChan)
	if numFiles == 0 {
		return errors.New("failed to collect any file")
	}

	files := make([]file, numFiles)
	for i := 0; i < numFiles; i++ {
		files[i] = <-filesChan
	}

	if err := saveCollection(files); err != nil {
		return fmt.Errorf("failed to save collection: %w", err)
	}

	return nil
}
