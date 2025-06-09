// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/spf13/cobra"
)

type collectFileInfo struct {
	src      string
	instance string
	compress bool
	// modifier, if not nil, is a function that lets the caller modify the downloaded
	// content before adding it to the tarball
	modifier func([]byte) ([]byte, error)
}

type collectCmdInfo struct {
	cmd        string
	outputName string
	instance   string
	compress   bool
	// modifier, if not nil, is a function that lets the caller modify the downloaded
	// content before adding it to the tarball
	modifier func([]byte) ([]byte, error)
}

type file struct {
	name string
	data []byte
}

func saveCollection(namePrefix string, files []file) error {
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

	name := fmt.Sprintf("%scollection_%d", namePrefix, time.Now().Unix())
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
		for i, inst := range output.Proxies {
			sshc, err := extAgent.NewClient(output.AMIUser, inst.GetConnectionIP())
			if err != nil {
				return nil, fmt.Errorf("error in getting ssh connection %w", err)
			}
			clients[fmt.Sprintf("proxy%d", i)] = sshc
		}
	}

	for i, instance := range output.Instances {
		sshc, err := extAgent.NewClient(output.AMIUser, instance.GetConnectionIP())
		if err != nil {
			return nil, fmt.Errorf("error in getting ssh connection %w", err)
		}
		clients[fmt.Sprintf("app%d", i)] = sshc
	}

	for i, agent := range output.Agents {
		sshc, err := extAgent.NewClient(output.AMIUser, agent.GetConnectionIP())
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

	return collect(config, "", "")
}

func collect(config deployment.Config, deploymentId string, outputName string) error {
	t, err := terraform.New(deploymentId, config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}

	output, err := t.Output()
	if err != nil {
		return err
	}

	clients, err := createClients(output)
	if err != nil {
		return err
	}

	if len(clients) == 0 {
		return errors.New("no active deployment found")
	}

	var collectFiles []collectFileInfo
	var collectCmds []collectCmdInfo

	addFile := func(instance, src string, compress bool, modifier func([]byte) ([]byte, error)) {
		collectFiles = append(collectFiles, collectFileInfo{
			src:      src,
			instance: instance,
			compress: compress,
			modifier: modifier,
		})
	}

	addCmd := func(instance, cmd string, outputName string, compress bool, modifier func([]byte) ([]byte, error)) {
		collectCmds = append(collectCmds, collectCmdInfo{
			cmd:        cmd,
			outputName: outputName,
			instance:   instance,
			compress:   compress,
			modifier:   modifier,
		})
	}

	for instance := range clients {
		switch {
		case strings.HasPrefix(instance, "proxy"):
			addFile(instance, "/var/log/nginx/error.log", true, nil)
			addFile(instance, "/etc/nginx/nginx.conf", false, nil)
			addFile(instance, "/etc/nginx/sites-enabled/mattermost", false, nil)
		case strings.HasPrefix(instance, "app"):
			addFile(instance, "/opt/mattermost/logs/mattermost.log", true, nil)
			addFile(instance, "/opt/mattermost/config/config.json", false, func(input []byte) ([]byte, error) {
				var cfg model.Config
				if err := json.Unmarshal(input, &cfg); err != nil {
					return nil, fmt.Errorf("failed to unmarshal MM configuration: %w", err)
				}
				cfg.Sanitize(nil)
				sanitizedCfg, err := json.MarshalIndent(cfg, "", "  ")
				if err != nil {
					return nil, fmt.Errorf("failed to sanitize MM configuration: %w", err)
				}
				return sanitizedCfg, nil
			})
		case strings.HasPrefix(instance, "agent"):
			addFile(instance, t.ExpandWithUser("/home/{{.Username}}/mattermost-load-test-ng/ltagent.log"), true, nil)
		case instance == "coordinator":
			addFile(instance, t.ExpandWithUser("/home/{{.Username}}/mattermost-load-test-ng/ltcoordinator.log"), true, nil)
			addFile(instance, t.ExpandWithUser("/home/{{.Username}}/mattermost-load-test-ng/config/config.json"), false, nil)
			addFile(instance, t.ExpandWithUser("/home/{{.Username}}/mattermost-load-test-ng/config/coordinator.json"), false, nil)
			addFile(instance, t.ExpandWithUser("/home/{{.Username}}/mattermost-load-test-ng/config/simplecontroller.json"), false, nil)
			addFile(instance, t.ExpandWithUser("/home/{{.Username}}/mattermost-load-test-ng/config/simulcontroller.json"), false, nil)
			continue
		}
		addCmd(instance, "sudo dmesg", "dmesg.out", false, nil)
	}

	var wg sync.WaitGroup
	collectChan := make(chan file, len(collectFiles)+len(collectCmds))
	wg.Add(len(collectFiles))
	for _, fileInfo := range collectFiles {
		go func(fileInfo collectFileInfo) {
			defer wg.Done()
			sshc := clients[fileInfo.instance]
			collectFile(sshc, collectChan, fileInfo)
		}(fileInfo)
	}
	wg.Add(len(collectCmds))
	for _, cmdInfo := range collectCmds {
		go func(cmdInfo collectCmdInfo) {
			defer wg.Done()
			sshc := clients[cmdInfo.instance]
			collectCmd(sshc, collectChan, cmdInfo)
		}(cmdInfo)
	}

	wg.Wait()

	numFiles := len(collectChan)
	if numFiles == 0 {
		return errors.New("failed to collect any file")
	}

	files := make([]file, numFiles)
	for i := 0; i < numFiles; i++ {
		files[i] = <-collectChan
	}

	if err := saveCollection(outputName, files); err != nil {
		return fmt.Errorf("failed to save collection: %w", err)
	}

	return nil
}

func collectFile(sshc *ssh.Client, collectChan chan file, fileInfo collectFileInfo) {
	downloadPath := fileInfo.src

	if fileInfo.compress {
		downloadPath = fmt.Sprintf("/tmp/%s.xz", filepath.Base(fileInfo.src))
		cmd := fmt.Sprintf("cat %s | xz -2 -T4 > %s", fileInfo.src, downloadPath)
		if _, err := sshc.RunCommand(cmd); err != nil {
			fmt.Printf("failed to run cmd %q: %s\n", cmd, err)
			return
		}
	}

	var b bytes.Buffer
	if err := sshc.Download(downloadPath, &b, false); err != nil {
		fmt.Printf("failed to download file %q: %s\n", downloadPath, err)
		return
	}

	// Apply modifiers to the data if any
	var output []byte
	var err error
	if fileInfo.modifier != nil {
		output, err = fileInfo.modifier(b.Bytes())
		if err != nil {
			fmt.Printf("failed to modify file %q: %s\n", downloadPath, err)
			return
		}
	} else {
		output = b.Bytes()
	}

	fmt.Printf("collected %s from %s instance\n", filepath.Base(downloadPath), fileInfo.instance)

	file := file{
		name: fmt.Sprintf("%s_%s", fileInfo.instance, filepath.Base(downloadPath)),
		data: output,
	}

	collectChan <- file
}

func collectCmd(sshc *ssh.Client, collectChan chan file, cmdInfo collectCmdInfo) {
	outPath := fmt.Sprintf("/tmp/%s", cmdInfo.outputName)

	cmd := fmt.Sprintf("%s > %s", cmdInfo.cmd, outPath)
	if _, err := sshc.RunCommand(cmd); err != nil {
		fmt.Printf("failed to run cmd %q: %s\n", cmd, err)
		return
	}

	fileInfo := collectFileInfo{
		src:      outPath,
		instance: cmdInfo.instance,
		compress: cmdInfo.compress,
		modifier: cmdInfo.modifier,
	}

	collectFile(sshc, collectChan, fileInfo)
}
