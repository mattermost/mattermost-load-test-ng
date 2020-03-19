// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"fmt"
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
	if err := sshc.Upload(rdr, "/etc/prometheus/prometheus.yml", true); err != nil {
		return err
	}

	mlog.Info("Starting Prometheus", mlog.String("host", output.MetricsServer.Value.PublicIP))
	cmd := fmt.Sprintf("sudo service prometheus restart && sudo systemctl enable prometheus")
	return sshc.RunCommand(cmd)
}
