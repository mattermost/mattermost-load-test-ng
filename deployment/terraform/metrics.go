// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

const (
	defaultGrafanaUsernamePass = "admin:admin"
	defaultRequestTimeout      = 10 * time.Second
)

func doAPIRequest(url, method string, payload io.Reader) (string, error) {
	// Set preference to new dashboard.
	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, payload)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Dump body.
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad response: %s", string(data))
	}

	return string(data), nil
}

// UploadDashboard uploads the given dashboard to Grafana and returns its URL.
// Returns an error in case of failure.
func (t *Terraform) UploadDashboard(dashboard string) (string, error) {
	output, err := t.Output()
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("http://%s@%s:3000/api/dashboards/db", defaultGrafanaUsernamePass, output.MetricsServer.PrivateIP)
	data := fmt.Sprintf(`{"dashboard":%s,"folderId":0,"overwrite":true}`, dashboard)
	data, err = doAPIRequest(url, http.MethodPost, strings.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("Grafana API request failed: %w", err)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	url, ok := resp["url"].(string)
	if !ok {
		return "", fmt.Errorf("bad response, missing url")
	}

	return url, nil
}

type PanelData struct {
	Id        int
	Title     string
	Legend    string
	Height    int
	Width     int
	PosX      int
	PosY      int
	Query     string
	Threshold float64
}

// Panel dimensions for the Grafana dashboard containing the coordinator metrics.
const (
	// According to the Grafana docs, "the width of the dashboard is divided into
	// 24 columns", so we set it to half that to fit two panels in each row.
	panelWidth = 12
	// According to the Grafana docs, each height unit "represents 30 pixels".
	// Setting the height to 9 (270px) is a good default.
	panelHeight = 9
)

type DashboardData struct {
	Panels []PanelData
}

func (t *Terraform) setupMetrics(extAgent *ssh.ExtAgent) error {
	// Updating Prometheus config
	sshc, err := extAgent.NewClient(t.output.MetricsServer.PrivateIP, t.Config().AWSAMIUser)
	if err != nil {
		return err
	}

	var hosts string
	var mmTargets, nodeTargets, esTargets, ltTargets, keycloakTargets, redisTargets, cloudwatchTargets, netpeekTargets []string
	for i, val := range t.output.Instances {
		host := fmt.Sprintf("app-%d", i)
		mmTargets = append(mmTargets, fmt.Sprintf("%s:8067", host))
		nodeTargets = append(nodeTargets, fmt.Sprintf("%s:9100", host))
		netpeekTargets = append(netpeekTargets, fmt.Sprintf("%s:9045", host))
		hosts += fmt.Sprintf("%s %s\n", val.PrivateIP, host)
	}
	for i, val := range t.output.Agents {
		host := fmt.Sprintf("agent-%d", i)
		nodeTargets = append(nodeTargets, fmt.Sprintf("%s:9100", host))
		ltTargets = append(ltTargets, fmt.Sprintf("%s:4000", host))
		hosts += fmt.Sprintf("%s %s\n", val.PrivateIP, host)
	}
	if t.output.HasProxy() {
		for i, val := range t.output.Proxies {
			host := fmt.Sprintf("proxy-%d", i)
			nodeTargets = append(nodeTargets, fmt.Sprintf("%s:9100", host))
			hosts += fmt.Sprintf("%s %s\n", val.PrivateIP, host)
		}
	}

	if t.output.HasElasticSearch() {
		esEndpoint := fmt.Sprintf("https://%s", t.output.ElasticSearchServer.Endpoint)
		esTargets = append(esTargets, "metrics:9114")

		serviceFileTmpl, err := template.New("es-exporter-service").Parse(esExporterServiceFile)
		if err != nil {
			return fmt.Errorf("error parsing elasticsearch exporter service file: %w", err)
		}

		var serviceFileOutput bytes.Buffer
		if err := serviceFileTmpl.Execute(&serviceFileOutput, map[string]string{
			"ESEndpoint": esEndpoint,
			"User":       t.Config().AWSAMIUser,
		}); err != nil {
			return fmt.Errorf("error executing elasticsearch exporter service file: %w", err)
		}

		mlog.Info("Enabling Elasticsearch exporter", mlog.String("host", t.output.MetricsServer.PrivateIP))
		rdr := strings.NewReader(serviceFileOutput.String())
		if out, err := sshc.Upload(rdr, "/lib/systemd/system/es-exporter.service", true); err != nil {
			return fmt.Errorf("error upload elasticsearch exporter service file: output: %s, error: %w", out, err)
		}
		cmd := "sudo systemctl enable es-exporter"
		if out, err := sshc.RunCommand(cmd); err != nil {
			return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
		}

		mlog.Info("Starting Elasticsearch exporter", mlog.String("host", t.output.MetricsServer.PrivateIP))
		cmd = "sudo systemctl restart es-exporter"
		if out, err := sshc.RunCommand(cmd); err != nil {
			return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
		}
	}

	if t.output.HasKeycloak() {
		host := "keycloak"
		keycloakTargets = append(keycloakTargets, fmt.Sprintf("%s:8080", host))
		hosts += fmt.Sprintf("%s %s\n", t.output.KeycloakServer.PrivateIP, host)
	}

	if t.output.HasRedis() {
		redisEndpoint := fmt.Sprintf("redis://%s", net.JoinHostPort(t.output.RedisServer.Address, strconv.Itoa(t.output.RedisServer.Port)))
		redisTargets = append(redisTargets, "metrics:9121")

		mlog.Info("Enabling Redis exporter", mlog.String("host", t.output.MetricsServer.PrivateIP))

		serviceFileTmpl, err := template.New("es-exporter-service").Parse(esExporterServiceFile)
		if err != nil {
			return fmt.Errorf("error parsing elasticsearch exporter service file: %w", err)
		}

		var serviceFileOutput bytes.Buffer
		if err := serviceFileTmpl.Execute(&serviceFileOutput, map[string]string{
			"RedisAddr": redisEndpoint,
			"User":      t.Config().AWSAMIUser,
		}); err != nil {
			return fmt.Errorf("error executing elasticsearch exporter service file: %w", err)
		}

		// TODO: Pass username/pass later if we ever start using them internally.
		// It's possible to configure them on the server, but there is no need to set them up for internal load tests.
		rdr := strings.NewReader(serviceFileOutput.String())
		if out, err := sshc.Upload(rdr, "/lib/systemd/system/redis-exporter.service", true); err != nil {
			return fmt.Errorf("error uploading redis exporter service file: output: %s, error: %w", out, err)
		}
		cmd := "sudo systemctl enable redis-exporter"
		if out, err := sshc.RunCommand(cmd); err != nil {
			return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
		}

		mlog.Info("Starting Redis exporter", mlog.String("host", t.output.MetricsServer.PrivateIP))
		cmd = "sudo systemctl restart redis-exporter"
		if out, err := sshc.RunCommand(cmd); err != nil {
			return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
		}
	}

	yacePort := "9106"
	yaceDurationSeconds := "300" // Used for period, length, delay and scraping interval

	cloudwatchTargets = append(cloudwatchTargets, "metrics:"+yacePort)

	mlog.Info("Updating YACE config", mlog.String("host", t.output.MetricsServer.PublicIP))
	yaceConfig, err := fillConfigTemplate(yaceConfigFile, map[string]any{
		"ClusterName": t.output.ClusterName,
		"Period":      yaceDurationSeconds,
		"Length":      yaceDurationSeconds,
		"Delay":       yaceDurationSeconds,
	})
	if err != nil {
		return fmt.Errorf("error rendering YACE configuration template: %w", err)
	}
	yace := strings.NewReader(yaceConfig)
	if out, err := sshc.Upload(yace, "/opt/yace/conf.yml", true); err != nil {
		return fmt.Errorf("error upload yace config: output: %s, error: %w", out, err)
	}

	yaceService, err := fillConfigTemplate(yaceServiceFile, map[string]any{
		"ScrapingInterval": yaceDurationSeconds,
		"Port":             yacePort,
		"User":             t.Config().AWSAMIUser,
	})
	if err != nil {
		return fmt.Errorf("error rendering YACE service template: %w", err)
	}
	yaceServiceReader := strings.NewReader(yaceService)
	if out, err := sshc.Upload(yaceServiceReader, "/lib/systemd/system/yace.service", true); err != nil {
		return fmt.Errorf("error uploading yace service file: output: %s, error: %w", out, err)
	}
	cmd := "sudo systemctl enable yace"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
	}

	mlog.Info("Starting Cloudwatch exporter: YACE", mlog.String("host", t.output.MetricsServer.PublicIP))
	cmd = "sudo systemctl restart yace"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
	}

	quoteAll := func(elems []string) []string {
		quoted := make([]string, 0, len(elems))
		for _, elem := range elems {
			quoted = append(quoted, "'"+elem+"'")
		}
		return quoted
	}

	mlog.Info("Updating Prometheus config", mlog.String("host", t.output.MetricsServer.PrivateIP))
	prometheusConfigFile := fmt.Sprintf(prometheusConfig,
		strings.Join(quoteAll(nodeTargets), ","),
		strings.Join(quoteAll(mmTargets), ","),
		strings.Join(quoteAll(esTargets), ","),
		strings.Join(quoteAll(ltTargets), ","),
		strings.Join(quoteAll(keycloakTargets), ""),
		strings.Join(quoteAll(redisTargets), ","),
		strings.Join(quoteAll(cloudwatchTargets), ","),
		strings.Join(quoteAll(netpeekTargets), ","),
	)
	rdr := strings.NewReader(prometheusConfigFile)
	if out, err := sshc.Upload(rdr, "/etc/prometheus/prometheus.yml", true); err != nil {
		return fmt.Errorf("error upload prometheus config: output: %s, error: %w", out, err)
	}

	mlog.Info("Updating Pyroscope config", mlog.String("host", t.output.MetricsServer.PrivateIP))
	pyroscopeMMTargets := []string{}
	if t.config.PyroscopeSettings.EnableAppProfiling {
		pyroscopeMMTargets = mmTargets
	}
	pyroscopeLTTargets := []string{}
	if t.config.PyroscopeSettings.EnableAgentProfiling {
		pyroscopeLTTargets = ltTargets
	}
	alloyConfig, err := NewAlloyConfig(pyroscopeMMTargets, pyroscopeLTTargets).marshal()
	if err != nil {
		return fmt.Errorf("error marshaling Alloy config: %w", err)
	}
	pyroscopeConfig, err := NewPyroscopeConfig().marshal()
	if err != nil {
		return fmt.Errorf("error marshaling Pyroscope config: %w", err)
	}
	alloyReader := bytes.NewReader(alloyConfig)
	if out, err := sshc.Upload(alloyReader, "/etc/alloy/config.alloy", true); err != nil {
		return fmt.Errorf("error upload alloy config: output: %s, error: %w", out, err)
	}
	pyroscopeReader := bytes.NewReader(pyroscopeConfig)
	if out, err := sshc.Upload(pyroscopeReader, "/etc/pyroscope/config.yml", true); err != nil {
		return fmt.Errorf("error upload pyroscope config: output: %s, error: %w", out, err)
	}

	metricsHostsFile := fmt.Sprintf(metricsHosts, hosts)
	rdr = strings.NewReader(metricsHostsFile)
	if out, err := sshc.Upload(rdr, "/etc/hosts", true); err != nil {
		return fmt.Errorf("error upload metrics hosts file: output: %s, error: %w", out, err)
	}

	mlog.Info("Starting Prometheus", mlog.String("host", t.output.MetricsServer.PrivateIP))
	cmd = "sudo systemctl restart prometheus"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
	}

	mlog.Info("Starting Alloy", mlog.String("host", t.output.MetricsServer.PrivateIP))
	cmd = "sudo systemctl restart alloy"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
	}

	mlog.Info("Starting Pyroscope", mlog.String("host", t.output.MetricsServer.PrivateIP))
	cmd = "sudo systemctl restart pyroscope"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
	}

	mlog.Info("Setting up Grafana", mlog.String("host", t.output.MetricsServer.PrivateIP))

	// Upload config file
	rdr = strings.NewReader(grafanaConfigFile)
	if out, err := sshc.Upload(rdr, "/etc/grafana/grafana.ini", true); err != nil {
		return fmt.Errorf("error upload grafana config: output: %s, error: %w", out, err)
	}

	// Upload datasource file
	buf, err := os.ReadFile(t.getAsset("datasource.yaml"))
	if err != nil {
		return err
	}
	dataSource := fmt.Sprintf(string(buf), "http://"+t.output.MetricsServer.PrivateIP+":9090")
	if out, err := sshc.Upload(strings.NewReader(dataSource), "/etc/grafana/provisioning/datasources/datasource.yaml", true); err != nil {
		return fmt.Errorf("error while uploading datasource: output: %s, error: %w", out, err)
	}

	// Upload dashboard file
	buf, err = os.ReadFile(t.getAsset("dashboard.yaml"))
	if err != nil {
		return err
	}
	if out, err := sshc.Upload(bytes.NewReader(buf), "/etc/grafana/provisioning/dashboards/dashboard.yaml", true); err != nil {
		return fmt.Errorf("error while uploading dashboard: output: %s, error: %w", out, err)
	}

	// Upload dashboard json
	buf, err = os.ReadFile(t.getAsset("default_dashboard_tmpl.json"))
	if err != nil {
		return err
	}
	bufStr, err := fillConfigTemplate(string(buf), map[string]any{"ClusterName": t.output.ClusterName})
	if err != nil {
		return err
	}
	cmd = "sudo mkdir -p /var/lib/grafana/dashboards"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
	}
	if out, err := sshc.Upload(strings.NewReader(bufStr), "/var/lib/grafana/dashboards/dashboard.json", true); err != nil {
		return fmt.Errorf("error while uploading dashboard_json: output: %s, error: %w", out, err)
	}

	// Download dashboard v2 from and upload it
	dashboardv2Resp, err := http.Get("https://grafana.com/api/dashboards/15582/revisions/latest/download")
	if err != nil {
		return fmt.Errorf("error downloading latest grafana v2 dashboard: %w", err)
	}
	defer dashboardv2Resp.Body.Close()

	var dashboardV2Contents bytes.Buffer
	_, err = io.Copy(&dashboardV2Contents, dashboardv2Resp.Body)
	if err != nil {
		return fmt.Errorf("error while reading dashboard v2: %w", err)
	}

	// Removes the DS_PROMETHEUS variable requirement to allow grafana to use the only prometheus
	// datasource available
	re := regexp.MustCompile(`,\r?\n\s+\"uid\":\s?\"\$\{DS_PROMETHEUS\}\"`)
	result := re.ReplaceAll(dashboardV2Contents.Bytes(), []byte(``))

	if out, err := sshc.Upload(bytes.NewReader(result), "/var/lib/grafana/dashboards/dashboard_v2.json", true); err != nil {
		return fmt.Errorf("error while uploading dashboard v2: output: %s, error: %w", out, err)
	}

	// Upload coordinator metrics dashboard
	coordConfig, err := coordinator.ReadConfig("")
	if err != nil {
		return fmt.Errorf("error while reading coordinator's config: %w", err)
	}
	panels := []PanelData{}
	i := 0
	for _, query := range coordConfig.MonitorConfig.Queries {
		if query.Alert {
			panels = append(panels, PanelData{
				Id:        i,
				Title:     query.Description,
				Legend:    query.Legend,
				Height:    panelHeight,
				Width:     panelWidth,
				PosX:      panelWidth * (i % 2),
				PosY:      panelHeight * (i / 2),
				Query:     strings.ReplaceAll(query.Query, "\"", "\\\""),
				Threshold: query.Threshold,
			})
			i++
		}
	}
	// Create the coordinator dashboard only if there is at least one panel
	if len(panels) > 0 {
		dashboard := DashboardData{panels}
		var b bytes.Buffer
		tmpl, err := template.ParseFiles(t.getAsset("coordinator_dashboard_tmpl.json"))
		if err != nil {
			return err
		}
		tmpl.Execute(&b, dashboard)
		if out, err := sshc.Upload(&b, "/var/lib/grafana/dashboards/coordinator_dashboard.json", true); err != nil {
			return fmt.Errorf("error while uploading coordinator_dashboard.json: output: %s, error: %w", out, err)
		}
	}

	if t.output.HasElasticSearch() {
		buf, err = os.ReadFile(t.getAsset("es_dashboard_data.json"))
		if err != nil {
			return err
		}
		if out, err := sshc.Upload(bytes.NewReader(buf), "/var/lib/grafana/dashboards/es_dashboard.json", true); err != nil {
			return fmt.Errorf("error while uploading es_dashboard_json: output: %s, error: %w", out, err)
		}
	}

	if t.output.HasRedis() {
		buf, err = os.ReadFile(t.getAsset("redis_dashboard_data.json"))
		if err != nil {
			return err
		}
		if out, err := sshc.Upload(bytes.NewReader(buf), "/var/lib/grafana/dashboards/redis_dashboard.json", true); err != nil {
			return fmt.Errorf("error while uploading redis_dashboard_json: output: %s, error: %w", out, err)
		}
	}

	// Restart grafana
	cmd = "sudo systemctl restart grafana-server"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: cmd: %s, output: %s, err: %v", cmd, out, err)
	}

	// Waiting for Grafana to be back up.
	url := fmt.Sprintf("http://%s@%s:3000/api/user/preferences", defaultGrafanaUsernamePass, t.output.MetricsServer.PrivateIP)
	timeout := time.After(10 * time.Second)
	for {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		mlog.Info("Server not up yet, waiting...")
		select {
		case <-timeout:
			return errors.New("timeout: server is not responding")
		case <-time.After(1 * time.Second):
		}
	}

	payload := struct {
		Theme           string `json:"theme"`
		HomeDashboardID int    `json:"homeDashboardId"`
		Timezone        string `json:"timezone"`
	}{
		HomeDashboardID: 4,
	}
	data, err := json.Marshal(&payload)
	if err != nil {
		return err
	}

	resp, err := doAPIRequest(url, http.MethodPut, bytes.NewReader(data))
	if err != nil {
		return err
	}
	mlog.Info("Response: " + resp)

	return nil
}
