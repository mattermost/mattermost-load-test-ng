// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import "fmt"

// Destroy destroys the created load-test environment.
func (t *Terraform) Destroy() error {
	err := t.preFlightCheck()
	if err != nil {
		return err
	}

	err = t.runCommand(nil, "destroy",
		"-var", fmt.Sprintf("cluster_name=%s", t.config.DeploymentConfiguration.ClusterName),
		"-var", fmt.Sprintf("app_instance_count=%d", t.config.DeploymentConfiguration.AppInstanceCount),
		"-var", fmt.Sprintf("ssh_public_key=%s", t.config.DeploymentConfiguration.SSHPublicKey),
		"-var", fmt.Sprintf("db_instance_count=%d", t.config.DeploymentConfiguration.DBInstanceCount),
		"-var", fmt.Sprintf("db_instance_engine=%s", t.config.DeploymentConfiguration.DBInstanceEngine),
		"-var", fmt.Sprintf("db_instance_class=%s", t.config.DeploymentConfiguration.DBInstanceClass),
		"-var", fmt.Sprintf("db_username=%s", t.config.DeploymentConfiguration.DBUserName),
		"-var", fmt.Sprintf("db_password=%s", t.config.DeploymentConfiguration.DBPassword),
		"-var", fmt.Sprintf("mattermost_download_url=%s", t.config.DeploymentConfiguration.MattermostDownloadURL),
		"-var", fmt.Sprintf("mattermost_license_file=%s", t.config.DeploymentConfiguration.MattermostLicenseFile),
		"-auto-approve",
		"./terraform",
	)
	if err != nil {
		return err
	}
	return nil
}
