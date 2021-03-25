// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/template"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/shared/mlog"
	"github.com/mattermost/mattermost-server/v5/utils"
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
	cmd, err := t.makeCmdForResource(resource)
	if err != nil {
		return fmt.Errorf("failed to make cmd for resource: %w", err)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}

func (t *Terraform) makeCmdForResource(resource string) (*exec.Cmd, error) {
	output, err := t.Output()
	if err != nil {
		return nil, fmt.Errorf("could not parse output: %w", err)
	}

	// Match against the agent names, or the reserved "coordinator" keyword referring to the
	// first agent.
	for i, agent := range output.Agents {
		if resource == agent.Tags.Name || (i == 0 && resource == "coordinator") {
			return exec.Command("ssh", fmt.Sprintf("ubuntu@%s", agent.PublicIP)), nil
		}
	}

	// Match against the instance names.
	for _, instance := range output.Instances {
		if resource == instance.Tags.Name {
			return exec.Command("ssh", fmt.Sprintf("ubuntu@%s", instance.PublicIP)), nil
		}
	}

	// Match against the job server names.
	for _, instance := range output.JobServers {
		if resource == instance.Tags.Name {
			return exec.Command("ssh", fmt.Sprintf("ubuntu@%s", instance.PublicIP)), nil
		}
	}

	// Match against the proxy or metrics servers, as well as convenient aliases.
	switch resource {
	case "proxy", output.Proxy.Tags.Name:
		return exec.Command("ssh", fmt.Sprintf("ubuntu@%s", output.Proxy.PublicIP)), nil
	case "metrics", "prometheus", "grafana", output.MetricsServer.Tags.Name:
		return exec.Command("ssh", fmt.Sprintf("ubuntu@%s", output.MetricsServer.PublicIP)), nil
	}

	return nil, fmt.Errorf("could not find any resource with name %q", resource)
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

func validateLicense(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read license file: %w", err)
	}

	ok, licenseStr := utils.ValidateLicense(data)
	if !ok {
		return errors.New("failed to validate license")
	}

	license := model.LicenseFromJson(strings.NewReader(licenseStr))
	if license == nil {
		return errors.New("failed to parse license")
	}

	if !license.IsStarted() {
		return errors.New("license has not started")
	}

	if license.IsExpired() {
		return errors.New("license has expired")
	}

	return nil
}

func (t *Terraform) getStatePath() string {
	statePath := "terraform.tfstate"
	if t.id != "" {
		statePath = t.id + ".tfstate"
	}
	return statePath
}

func fillConfigTemplate(configTmpl string, data map[string]string) (string, error) {
	var buf bytes.Buffer
	tmpl := template.New("template")
	tmpl, err := tmpl.Parse(configTmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.String(), nil
}
