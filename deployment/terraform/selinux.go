package terraform

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func (t *Terraform) disableSELinux(sshc *ssh.Client) error {
	mlog.Info("Checking SELinux status", mlog.String("host", sshc.IP))

	// Check if SELinux is enabled
	cmd := "sudo getenforce"
	out, err := sshc.RunCommand(cmd)
	if err != nil {
		mlog.Warn("SELinux check failed, assuming not available", mlog.String("host", sshc.IP), mlog.Err(err))
		return nil
	}

	status := strings.TrimSpace(string(out))
	mlog.Info("Current SELinux status", mlog.String("status", status), mlog.String("host", sshc.IP))

	// Only disable if SELinux is enabled (Enforcing or Permissive)
	if status == "Enforcing" || status == "Permissive" {
		mlog.Info("Disabling SELinux", mlog.String("host", sshc.IP))
		cmd = "sudo setenforce 0"
		if out, err := sshc.RunCommand(cmd); err != nil {
			mlog.Error(string(out))
			return fmt.Errorf("error running ssh command %q, output: %q: %w", cmd, string(out), err)
		}
	} else {
		mlog.Info("SELinux not enabled, no need to disable", mlog.String("host", sshc.IP))
	}

	return nil
}
