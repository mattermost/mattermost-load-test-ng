// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"fmt"
)

// Destroy destroys the created load-test environment.
func (t *Terraform) Destroy() error {
	err := t.PreFlightCheck()
	if err != nil {
		return err
	}

	return t.runCommand(nil, "destroy",
		"-var", fmt.Sprintf("cluster_name=%s", t.config.ClusterName),
		"-var", fmt.Sprintf("app_instance_count=%d", t.config.AppInstanceCount),
		"-var", fmt.Sprintf("app_instance_type=%s", t.config.AppInstanceType),
		"-var", fmt.Sprintf("agent_instance_count=%d", t.config.AgentInstanceCount),
		"-var", fmt.Sprintf("agent_instance_type=%s", t.config.AgentInstanceType),
		"-var", fmt.Sprintf("es_instance_count=%d", t.config.ElasticSearchSettings.InstanceCount),
		"-var", fmt.Sprintf("es_instance_type=%s", t.config.ElasticSearchSettings.InstanceType),
		"-var", fmt.Sprintf("es_version=%.1f", t.config.ElasticSearchSettings.Version),
		"-var", fmt.Sprintf("es_vpc=%s", t.config.ElasticSearchSettings.VpcID),
		"-var", fmt.Sprintf("es_create_role=%t", t.config.ElasticSearchSettings.CreateRole),
		"-var", fmt.Sprintf("proxy_instance_type=%s", t.config.ProxyInstanceType),
		"-var", fmt.Sprintf("ssh_public_key=%s", t.config.SSHPublicKey),
		"-var", fmt.Sprintf("db_instance_count=%d", t.config.TerraformDBSettings.InstanceCount),
		"-var", fmt.Sprintf("db_instance_engine=%s", t.config.TerraformDBSettings.InstanceEngine),
		"-var", fmt.Sprintf("db_instance_class=%s", t.config.TerraformDBSettings.InstanceType),
		"-var", fmt.Sprintf("db_username=%s", t.config.TerraformDBSettings.UserName),
		"-var", fmt.Sprintf("db_password=%s", t.config.TerraformDBSettings.Password),
		"-var", fmt.Sprintf("mattermost_download_url=%s", t.config.MattermostDownloadURL),
		"-var", fmt.Sprintf("mattermost_license_file=%s", t.config.MattermostLicenseFile),
		"-var", fmt.Sprintf("load_test_download_url=%s", t.config.LoadTestDownloadURL),
		"-auto-approve",
		"-state="+t.getStatePath(),
		t.dir,
	)
}
