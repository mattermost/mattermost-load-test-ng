package terraform

import (
	"fmt"
	"strconv"
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

// modifyRXSize modifies the size of the receiving buffer of the network card
// to the maximum permitted by the hardware.
func modifyRXSize(sshc *ssh.Client) error {
	rxSizes, err := getEthtoolRxSizes(sshc)
	if err != nil {
		return fmt.Errorf("error getting the RX sizes from the ethtool: %w", err)
	}

	oldRxSize := rxSizes.actualRX

	// Modify RX queue size to the maximum permitted
	incRxSizeCmd := fmt.Sprintf("sudo ethtool -G $(ip route show to default | awk '{print $5}') rx %d", rxSizes.maxRX)
	cmd := fmt.Sprintf("%s && sudo sysctl -p && sudo systemctl restart nginx", incRxSizeCmd)
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command; output: %q; cmd: %q; err: %w", string(out), cmd, err)
	}

	// Log the actual RX queue size after the modification
	newRxSizes, err := getEthtoolRxSizes(sshc)
	if err != nil {
		return fmt.Errorf("error getting the RX sizes from the ethtool: %w", err)
	}

	newRxSize := newRxSizes.actualRX
	if err != nil {
		return fmt.Errorf("error retrieving actual RX queue size: %w", err)
	}

	mlog.Info("size of the receiving ring buffer in the proxy has been modified", mlog.Int("old_size", oldRxSize), mlog.Int("new_size", newRxSize), mlog.Int("max_size", newRxSizes.maxRX))
	return nil
}

type ethtoolOutput struct {
	maxRX    int
	actualRX int
}

// getEthtoolRxSizes runs the ethtool command and parses its output.
func getEthtoolRxSizes(sshc *ssh.Client) (ethtoolOutput, error) {
	getRXCmd := "sudo ethtool -g ens3"
	out, err := sshc.RunCommand(getRXCmd)
	if err != nil {
		return ethtoolOutput{}, fmt.Errorf("error running ssh command; output: %q; cmd: %q, err: %w", string(out), getRXCmd, err)
	}

	return parseEthtoolOutputRXSizes(string(out))
}

// parseEthtoolOutputRXSizes parses the RX ring buffer sizes (maximum permitted
// and actual) from the output of ethtool -g, which looks like:
//
//	Ring parameters for ens3:
//	Pre-set maximums:
//	RX:             4096
//	RX Mini:        n/a
//	RX Jumbo:       n/a
//	TX:             4096
//	Current hardware settings:
//	RX:             1024
//	RX Mini:        n/a
//	RX Jumbo:       n/a
//	TX:             1024
func parseEthtoolOutputRXSizes(out string) (ethtoolOutput, error) {
	lines := strings.Split(out, "\n")
	rxLines := []string{}
	prefix := "RX:"
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			noPrefix := strings.TrimPrefix(line, prefix)
			trimmed := strings.ReplaceAll(noPrefix, "\t", "")
			rxLines = append(rxLines, trimmed)
		}
	}

	if len(rxLines) != 2 {
		return ethtoolOutput{}, fmt.Errorf("unexpected number of matching RX lines: %d", len(rxLines))
	}

	rxValues := make([]int, 2)
	for i, line := range rxLines {
		val, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil {
			return ethtoolOutput{}, fmt.Errorf("unable to parse integer in line %d: %q; err: %w", i, line, err)
		}

		rxValues[i] = val
	}

	return ethtoolOutput{
		maxRX:    rxValues[0],
		actualRX: rxValues[1],
	}, nil
}
