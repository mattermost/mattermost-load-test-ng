// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

type uploadInfo struct {
	msg     string
	srcData string
	dstPath string
}

func uploadBatch(sshc *ssh.Client, batch []uploadInfo) error {
	if sshc == nil {
		return errors.New("sshc should not be nil")
	}
	if len(batch) == 0 {
		return errors.New("batch should not be empty")
	}

	for _, info := range batch {
		if info.msg != "" {
			mlog.Info(info.msg)
		}
		rdr := strings.NewReader(info.srcData)
		if out, err := sshc.Upload(rdr, info.dstPath, true); err != nil {
			return fmt.Errorf("error uploading file, dstPath: %s, output: %q: %w", info.dstPath, out, err)
		}
	}

	return nil
}

// OpenSSHFor starts a ssh connection to the resource
func (t *Terraform) OpenSSHFor(resource string) error {
	output, err := t.Output()
	if err != nil {
		return fmt.Errorf("could not parse output: %w", err)
	}
	var cmd *exec.Cmd
	for i, agent := range output.Agents {
		if resource == agent.Tags.Name || (i == 0 && resource == "coordinator") {
			cmd = exec.Command("ssh", fmt.Sprintf("ubuntu@%s", agent.PublicIP))
			break
		}
	}
	if cmd == nil {
		for _, instance := range output.Instances {
			if resource == instance.Tags.Name {
				cmd = exec.Command("ssh", fmt.Sprintf("ubuntu@%s", instance.PublicIP))
				break
			}
		}
	}
	if cmd == nil {
		return fmt.Errorf("could not find any resource with name %q", resource)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}

// OpenBrowserFor opens a web browser for the resource
func (t *Terraform) OpenBrowserFor(resource string) error {
	output, err := t.Output()
	if err != nil {
		return fmt.Errorf("could not parse output: %w", err)
	}
	url := "http://"
	switch resource {
	case "grafana":
		url += output.MetricsServer.PublicDNS + ":3000"
	case "mattermost":
		if output.Proxy.PublicDNS != "" {
			url += output.Proxy.PublicDNS
		} else {
			url += output.Instances[0].PublicDNS + ":8065"
		}
	case "prometheus":
		url += output.MetricsServer.PublicDNS + ":9090"
	default:
		return fmt.Errorf("undefined resource :%q", resource)
	}
	fmt.Printf("Opening %s...\n", url)
	return openBrowser(url)
}

func openBrowser(url string) (err error) {
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = errors.New("unsupported platform")
	}
	return
}
