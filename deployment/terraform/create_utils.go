package terraform

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// setupPrometheusNodeExporter sets up prometheus-node-exporter on the given host.
// For now, this only set ups the service file, enables and starts the service.
func (t *Terraform) setupPrometheusNodeExporter(sshClient *ssh.Client) error {
	mlog.Info("Setting up prometheus-node-exporter")

	serviceFile, err := fillConfigTemplate(prometheusNodeExporterServiceFile, nil)
	if err != nil {
		return fmt.Errorf("failed to fill prometheus-node-exporter service file template: %w", err)
	}

	if out, err := sshClient.Upload(strings.NewReader(serviceFile), "/etc/systemd/system/prometheus-node-exporter.service", true); err != nil {
		mlog.Error(string(out))
		return fmt.Errorf("failed to upload prometheus-node-exporter service file: %w", err)
	}

	if out, err := sshClient.RunCommand("sudo systemctl daemon-reload"); err != nil {
		mlog.Error(string(out))
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	if out, err := sshClient.RunCommand("sudo systemctl enable --now prometheus-node-exporter"); err != nil {
		mlog.Error(string(out))
		return fmt.Errorf("failed to enable prometheus-node-exporter service: %w", err)
	}

	return nil
}
