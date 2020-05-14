// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package deployment

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/logger"
)

// Config contains the necessary data
// to deploy and provision a load test environment.
type Config struct {
	// ClusterName is the name of the cluster.
	ClusterName string `default:"loadtest" validate:"alpha"`
	// Number of application instances.
	AppInstanceCount int `default:"1" validate:"range:[1,)"`
	// Type of the EC2 instance for app.
	AppInstanceType string `default:"c5.xlarge" validate:"oneof:{c5.xlarge, t3.xlarge, m4.xlarge}"`
	// Number of agents, first agent and coordinator will share the same instance.
	AgentInstanceCount int `default:"2" validate:"range:[1,)"`
	// Type of the EC2 instance for agent.
	AgentInstanceType string `default:"t3.xlarge" validate:"oneof:{c5.xlarge, t3.xlarge, m4.xlarge}"`
	// Logs the command output (stdout & stderr) to home directory.
	EnableAgentFullLogs bool `default:"true"`
	// Type of the EC2 instance for proxy.
	ProxyInstanceType string `default:"m4.xlarge" validate:"oneof:{c5.xlarge, t3.xlarge, m4.xlarge}"`
	// Path to the SSH public key.
	SSHPublicKey string `default:"~/.ssh/id_rsa.pub" validate:"text"`
	// Number of DB instances.
	DBInstanceCount int `default:"1" validate:"range:[1,)"`
	// Type of the DB instance.
	DBInstanceType string `default:"db.r4.large" validate:"oneof:{db.r4.large}"`
	// Type of the DB instance - postgres or mysql.
	DBInstanceEngine string `default:"aurora-postgresql" validate:"oneof:{aurora-mysql, aurora-postgresql}"`
	// Username to connect to the DB.
	DBUserName string `default:"mmuser" validate:"text"`
	// Password to connect to the DB.
	DBPassword string `default:"mostest80098bigpass_" validate:"text"`
	// URL from where to download Mattermost release.
	// This can also point to a local binary path if the user wants to run loadtest
	// on a custom build. The path should be prefixed with "file://". In that case,
	// only the binary gets replaced, and the rest of the build comes from the latest
	// stable release.
	MattermostDownloadURL string `default:"https://latest.mattermost.com/mattermost-enterprise-linux" validate:"url"`
	// Path to the Mattermost EE license file.
	MattermostLicenseFile string `default:"" validate:"file"`
	// Mattermost instance sysadmin e-mail.
	AdminEmail string `default:"sysadmin@sample.mattermost.com" validate:"email"`
	// Mattermost instance sysadmin user name.
	AdminUsername string `default:"sysadmin" validate:"text"`
	// Mattermost instance sysadmin password.
	AdminPassword string `default:"Sys@dmin-sample1" validate:"text"`
	// URL from where to download load-test-ng binaries and configuration files.
	// The configuration files provided in the package will be overridden in
	// the deployment process.
	LoadTestDownloadURL string `default:"https://github.com/mattermost/mattermost-load-test-ng/releases/download/v0.5.0-alpha/mattermost-load-test-ng-v0.5.0-alpha-linux-amd64.tar.gz" validate:"url"`
	LogSettings         logger.Settings
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will return a config with default values.
func ReadConfig(configFilePath string) (*Config, error) {
	var cfg Config
	if configFilePath == "" {
		if err := defaults.Set(&cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	}

	file, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %w", err)
	}

	err = json.NewDecoder(file).Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("could not decode file: %w", err)
	}

	return &cfg, nil
}
