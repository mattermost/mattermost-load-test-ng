// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"fmt"
)

// Destroy destroys the created load-test environment.
func (t *Terraform) Destroy() error {
	err := t.preFlightCheck()
	if err != nil {
		return err
	}

	return t.runCommand(nil, "destroy",
		"-var", fmt.Sprintf("cluster_name=%s", t.config.ClusterName),
		"-var", fmt.Sprintf("app_instance_count=%d", t.config.AppInstanceCount),
		"-var", fmt.Sprintf("loadtest_agent_count=%d", t.config.AgentCount),
		"-var", fmt.Sprintf("ssh_public_key=%s", t.config.SSHPublicKey),
		"-var", fmt.Sprintf("db_instance_count=%d", t.config.DBInstanceCount),
		"-var", fmt.Sprintf("db_instance_engine=%s", t.config.DBInstanceEngine),
		"-var", fmt.Sprintf("db_instance_class=%s", t.config.DBInstanceClass),
		"-var", fmt.Sprintf("db_username=%s", t.config.DBUserName),
		"-var", fmt.Sprintf("db_password=%s", t.config.DBPassword),
		"-var", fmt.Sprintf("mattermost_download_url=%s", t.config.MattermostDownloadURL),
		"-var", fmt.Sprintf("mattermost_license_file=%s", t.config.MattermostLicenseFile),
		"-var", fmt.Sprintf("go_version=%s", t.config.GoVersion),
		"-var", fmt.Sprintf("loadtest_source_code_ref=%s", t.config.SourceCodeRef),
		"-auto-approve",
		t.dir,
	)
}
