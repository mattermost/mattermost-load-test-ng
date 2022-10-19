// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"text/template"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/shared/mlog"
	"github.com/mattermost/mattermost-server/v6/utils"
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

// RunSSHCommand runs a command on a given machine.
func (t *Terraform) RunSSHCommand(resource string, args []string) error {
	cmd, err := t.makeCmdForResource(resource)
	if err != nil {
		return fmt.Errorf("failed to make cmd for resource: %w", err)
	}
	cmd.Args = append(cmd.Args, args...)

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

	validator := &utils.LicenseValidatorImpl{}
	ok, licenseStr := validator.ValidateLicense(data)
	if !ok {
		return errors.New("failed to validate license")
	}

	var license model.License
	if err := json.Unmarshal([]byte(licenseStr), &license); err != nil {
		return fmt.Errorf("failed to parse license: %w", err)
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
	return path.Join(t.stateDir, statePath)
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

func (t *Terraform) getParams() []string {
	return []string{
		"-var", fmt.Sprintf("cluster_name=%s", t.config.ClusterName),
		"-var", fmt.Sprintf("cluster_vpc_id=%s", t.config.ClusterVpcID),
		"-var", fmt.Sprintf("cluster_subnet_id=%s", t.config.ClusterSubnetID),
		"-var", fmt.Sprintf("app_instance_count=%d", t.config.AppInstanceCount),
		"-var", fmt.Sprintf("app_instance_type=%s", t.config.AppInstanceType),
		"-var", fmt.Sprintf("agent_instance_count=%d", t.config.AgentInstanceCount),
		"-var", fmt.Sprintf("agent_instance_type=%s", t.config.AgentInstanceType),
		"-var", fmt.Sprintf("es_instance_count=%d", t.config.ElasticSearchSettings.InstanceCount),
		"-var", fmt.Sprintf("es_instance_type=%s", t.config.ElasticSearchSettings.InstanceType),
		"-var", fmt.Sprintf("es_version=%.1f", t.config.ElasticSearchSettings.Version),
		"-var", fmt.Sprintf("es_vpc=%s", t.config.ElasticSearchSettings.VpcID),
		"-var", fmt.Sprintf("es_create_role=%t", t.config.ElasticSearchSettings.CreateRole),
		"-var", fmt.Sprintf("proxy_instance_type=%s", t.config.ProxyInstanceType),
		"-var", fmt.Sprintf("ssh_public_key=%s", t.config.SSHPublicKey),
		"-var", fmt.Sprintf("db_instance_count=%d", t.config.TerraformDBSettings.InstanceCount),
		"-var", fmt.Sprintf("db_instance_engine=%s", t.config.TerraformDBSettings.InstanceEngine),
		"-var", fmt.Sprintf("db_instance_class=%s", t.config.TerraformDBSettings.InstanceType),
		"-var", fmt.Sprintf("db_username=%s", t.config.TerraformDBSettings.UserName),
		"-var", fmt.Sprintf("db_password=%s", t.config.TerraformDBSettings.Password),
		"-var", fmt.Sprintf("db_enable_performance_insights=%t", t.config.TerraformDBSettings.EnablePerformanceInsights),
		"-var", fmt.Sprintf("mattermost_license_file=%s", t.config.MattermostLicenseFile),
		"-var", fmt.Sprintf("job_server_instance_count=%d", t.config.JobServerSettings.InstanceCount),
		"-var", fmt.Sprintf("job_server_instance_type=%s", t.config.JobServerSettings.InstanceType),
	}
}

func (t *Terraform) getClusterDSN() (string, error) {
	switch t.config.TerraformDBSettings.InstanceEngine {
	case "aurora-postgresql":
		return "postgres://" + t.config.TerraformDBSettings.UserName + ":" + t.config.TerraformDBSettings.Password + "@" + t.output.DBWriter() + "/" + t.config.DBName() + "?sslmode=disable", nil

	case "aurora-mysql":
		return t.config.TerraformDBSettings.UserName + ":" + t.config.TerraformDBSettings.Password + "@tcp(" + t.output.DBWriter() + ")/" + t.config.DBName() + "?charset=utf8mb4,utf8\u0026readTimeout=30s\u0026writeTimeout=30s", nil
	default:
		return "", errors.New("unsupported database engine")
	}
}
