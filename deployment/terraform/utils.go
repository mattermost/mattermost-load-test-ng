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
	"time"

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
	for i, agent := range output.Agents.Value {
		if resource == agent.Tags.Name || (i == 0 && resource == "coordinator") {
			cmd = exec.Command("ssh", fmt.Sprintf("ubuntu@%s", agent.PublicIP))
			break
		}
	}
	if cmd == nil {
		for _, instance := range output.Instances.Value {
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
		url += output.MetricsServer.Value.PublicDNS + ":3000"
	case "mattermost":
		if output.Proxy.Value[0].PublicDNS != "" {
			url += output.Proxy.Value[0].PublicDNS
		} else {
			url += output.Instances.Value[0].PublicDNS + ":8065"
		}
	case "prometheus":
		url += output.MetricsServer.Value.PublicDNS + ":9090"
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
		err = fmt.Errorf("unsupported platform")
	}
	return
}

// LogAgentsOutput listens the strace output of currently running agents and
// saves to a timestamped file.
func (t *Terraform) LogAgentsOutput() error {
	output, err := t.Output()
	if err != nil {
		return err
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}

	for _, val := range output.Agents.Value {
		sshc, err := extAgent.NewClient(val.PublicIP)
		if err != nil {
			return err
		}

		script := fmt.Sprintf("sudo strace -e trace=write -s1000 -fp $(pidof ltagent) 2>&1 | grep --line-buffered -o '\".\\+[^\"]\"' | grep --line-buffered -o '[^\"]\\+[^\"]' | while read line; do  echo >> %s_ltagent.log $line; done", time.Now().Format("20060102T1504"))
		if err := sshc.StartCommand(script); err != nil {
			return fmt.Errorf("error running command: %w", err)
		}
	}

	return nil
}
