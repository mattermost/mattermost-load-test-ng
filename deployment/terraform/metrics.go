// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

const defaultGrafanaUsernamePass = "admin:admin"

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
	if err := sshc.Upload(rdr, "/etc/prometheus/prometheus.yml", true); err != nil {
		return err
	}

	mlog.Info("Starting Prometheus", mlog.String("host", output.MetricsServer.Value.PublicIP))
	if err := sshc.RunCommand("sudo service prometheus restart"); err != nil {
		return err
	}

	mlog.Info("Setting up Grafana", mlog.String("host", output.MetricsServer.Value.PublicIP))
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
	resp, err := http.Post(url, "application/json", bytes.NewReader(buf))
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
		return fmt.Errorf("non 200 response: %s", string(buf))
	}
	mlog.Info("Response: " + string(buf))
	return nil
}
