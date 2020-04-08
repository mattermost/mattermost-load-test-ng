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
	"strings"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

const (
	defaultGrafanaUsernamePass = "admin:admin"
	defaultRequestTimeout      = 10 * time.Second
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()
	url := "http://" + defaultGrafanaUsernamePass + "@" + output.MetricsServer.Value.PublicIP + ":3000/api/datasources"
	payload := struct {
		Name     string `json:"name"`
		DataType string `json:"type"`
		URL      string `json:"url"`
		Access   string `json:"access"`
	}{
		Name:     "loadtest-source",
		DataType: "prometheus",
		URL:      "http://" + output.MetricsServer.Value.PublicIP + ":9090",
		Access:   "proxy",
	}
	buf, err := json.Marshal(&payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("bad response: %s", string(buf))
	}
	mlog.Info("Response: " + string(buf))
	return nil
}
