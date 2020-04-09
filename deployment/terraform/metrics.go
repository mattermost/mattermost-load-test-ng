// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

func (t *Terraform) setupMetrics(extAgent *ssh.ExtAgent, output *terraformOutput) error {
	// Updating Prometheus config
	sshc, err := extAgent.NewClient(output.MetricsServer.Value.PublicIP)
	if err != nil {
		return err
	}

	var mmEndpoint, nodeExporterEndpoint []string
	for _, val := range output.Instances.Value {
		mmEndpoint = append(mmEndpoint, "'"+val.PrivateIP+":8067'")
		nodeExporterEndpoint = append(nodeExporterEndpoint, "'"+val.PrivateIP+":9100'")
	}
	mmConfig := strings.Join(mmEndpoint, ",")
	nodeExporterConfig := strings.Join(nodeExporterEndpoint, ",")

	prometheusConfigFile := fmt.Sprintf(prometheusConfig, mmConfig, nodeExporterConfig)
	rdr := strings.NewReader(prometheusConfigFile)
	mlog.Info("Updating Prometheus config", mlog.String("host", output.MetricsServer.Value.PublicIP))
	if out, err := sshc.Upload(rdr, "/etc/prometheus/prometheus.yml", true); err != nil {
		return fmt.Errorf("error upload prometheus config: output: %s, error: %w", out, err)
	}

	mlog.Info("Starting Prometheus", mlog.String("host", output.MetricsServer.Value.PublicIP))
	cmd := "sudo service prometheus restart"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
	}

	mlog.Info("Setting up Grafana", mlog.String("host", output.MetricsServer.Value.PublicIP))

	// Upload datasource file
	buf, err := ioutil.ReadFile(path.Join(t.dir, "datasource.yaml"))
	if err != nil {
		return err
	}
	dataSource := fmt.Sprintf(string(buf), "http://"+output.MetricsServer.Value.PublicIP+":9090")
	if out, err := sshc.Upload(strings.NewReader(dataSource), "/etc/grafana/provisioning/datasources/datasource.yaml", true); err != nil {
		return fmt.Errorf("error while uploading datasource: output: %s, error: %w", out, err)
	}

	// Upload dashboard file
	buf, err = ioutil.ReadFile(path.Join(t.dir, "dashboard.yaml"))
	if err != nil {
		return err
	}
	if out, err := sshc.Upload(bytes.NewReader(buf), "/etc/grafana/provisioning/dashboards/dashboard.yaml", true); err != nil {
		return fmt.Errorf("error while uploading dashboard: output: %s, error: %w", out, err)
	}

	// Upload dashboard json
	buf, err = ioutil.ReadFile(path.Join(t.dir, "dashboard_data.json"))
	if err != nil {
		return err
	}
	cmd = "sudo mkdir -p /var/lib/grafana/dashboards"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
	}
	if out, err := sshc.Upload(bytes.NewReader(buf), "/var/lib/grafana/dashboards/dashboard.json", true); err != nil {
		return fmt.Errorf("error while uploading dashboard_json: output: %s, error: %w", out, err)
	}

	// Restart grafana
	cmd = "sudo service grafana-server restart"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
	}

	return nil
}
