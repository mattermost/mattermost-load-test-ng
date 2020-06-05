// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

const (
	defaultGrafanaUsernamePass = "admin:admin"
	defaultRequestTimeout      = 10 * time.Second
)

func (t *Terraform) setupMetrics(extAgent *ssh.ExtAgent, output *Output) error {
	// Updating Prometheus config
	sshc, err := extAgent.NewClient(output.MetricsServer.Value.PublicIP)
	if err != nil {
		return err
	}

	var hosts string
	var mmTargets, nodeTargets, esTargets []string
	for i, val := range output.Instances.Value {
		host := fmt.Sprintf("app-%d", i)
		mmTargets = append(mmTargets, fmt.Sprintf("'%s:8067'", host))
		nodeTargets = append(nodeTargets, fmt.Sprintf("'%s:9100'", host))
		hosts += fmt.Sprintf("%s %s\n", val.PrivateIP, host)
	}
	for i, val := range output.Agents.Value {
		host := fmt.Sprintf("agent-%d", i)
		nodeTargets = append(nodeTargets, fmt.Sprintf("'%s:9100'", host))
		hosts += fmt.Sprintf("%s %s\n", val.PrivateIP, host)
	}
	if output.HasProxy() {
		host := "proxy"
		nodeTargets = append(nodeTargets, fmt.Sprintf("'%s:9100'", host))
		hosts += fmt.Sprintf("%s %s\n", output.Proxy.Value[0].PrivateIP, host)
	}

	if output.HasElasticSearch() {
		esEndpoint := fmt.Sprintf("https://%s", output.ElasticServer.Value[0].Endpoint)
		esTargets = append(esTargets, "'metrics:9114'")

		mlog.Info("Enabling Elasticsearch exporter", mlog.String("host", output.MetricsServer.Value.PublicIP))
		esExporterService := fmt.Sprintf(esExporterServiceFile, esEndpoint)
		rdr := strings.NewReader(esExporterService)
		if out, err := sshc.Upload(rdr, "/lib/systemd/system/es-exporter.service", true); err != nil {
			return fmt.Errorf("error upload elasticsearch exporter service file: output: %s, error: %w", out, err)
		}
		cmd := "sudo systemctl enable es-exporter"
		if out, err := sshc.RunCommand(cmd); err != nil {
			return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
		}

		mlog.Info("Starting Elasticsearch exporter", mlog.String("host", output.MetricsServer.Value.PublicIP))
		cmd = "sudo service es-exporter restart"
		if out, err := sshc.RunCommand(cmd); err != nil {
			return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
		}
	}

	mlog.Info("Updating Prometheus config", mlog.String("host", output.MetricsServer.Value.PublicIP))
	prometheusConfigFile := fmt.Sprintf(prometheusConfig,
		strings.Join(nodeTargets, ","),
		strings.Join(mmTargets, ","),
		strings.Join(esTargets, ","),
	)
	rdr := strings.NewReader(prometheusConfigFile)
	if out, err := sshc.Upload(rdr, "/etc/prometheus/prometheus.yml", true); err != nil {
		return fmt.Errorf("error upload prometheus config: output: %s, error: %w", out, err)
	}
	metricsHostsFile := fmt.Sprintf(metricsHosts, hosts)
	rdr = strings.NewReader(metricsHostsFile)
	if out, err := sshc.Upload(rdr, "/etc/hosts", true); err != nil {
		return fmt.Errorf("error upload metrics hosts file: output: %s, error: %w", out, err)
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

	if output.HasElasticSearch() {
		buf, err = ioutil.ReadFile(path.Join(t.dir, "es_dashboard_data.json"))
		if err != nil {
			return err
		}
		if out, err := sshc.Upload(bytes.NewReader(buf), "/var/lib/grafana/dashboards/es_dashboard.json", true); err != nil {
			return fmt.Errorf("error while uploading es_dashboard_json: output: %s, error: %w", out, err)
		}
	}

	// Restart grafana
	cmd = "sudo service grafana-server restart"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
	}

	// Set preference to new dashboard.
	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()
	url := "http://" + defaultGrafanaUsernamePass + "@" + output.MetricsServer.Value.PublicIP + ":3000/api/user/preferences"
	payload := struct {
		Theme           string `json:"theme"`
		HomeDashboardID int    `json:"homeDashboardId"`
		Timezone        string `json:"timezone"`
	}{
		HomeDashboardID: 2,
	}
	buf, err = json.Marshal(&payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Dump body.
	buf, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad response: %s", string(buf))
	}
	mlog.Info("Response: " + string(buf))

	return nil
}
