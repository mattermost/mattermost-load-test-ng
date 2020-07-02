// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package deployment

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/report"
	"github.com/mattermost/mattermost-load-test-ng/logger"
)

var esDomainNameRe = regexp.MustCompile(`^[a-z][a-z0-9\-]{2,27}$`)

// Config contains the necessary data
// to deploy and provision a load test environment.
type Config struct {
	// ClusterName is the name of the cluster.
	ClusterName string `default:"loadtest" validate:"alpha"`
	// Number of application instances.
	AppInstanceCount int `default:"1" validate:"range:[0,)"`
	// Type of the EC2 instance for app.
	AppInstanceType string `default:"c5.xlarge" validate:"notempty"`
	// Number of agents, first agent and coordinator will share the same instance.
	AgentInstanceCount int `default:"2" validate:"range:[1,)"`
	// Type of the EC2 instance for agent.
	AgentInstanceType string `default:"t3.xlarge" validate:"notempty"`
	// Logs the command output (stdout & stderr) to home directory.
	EnableAgentFullLogs bool `default:"true"`
	// Type of the EC2 instance for proxy.
	ProxyInstanceType string `default:"m4.xlarge" validate:"notempty"`
	// Path to the SSH public key.
	SSHPublicKey string `default:"~/.ssh/id_rsa.pub" validate:"notempty"`
	// Number of DB instances.
	DBInstanceCount int `default:"1" validate:"range:[1,)"`
	// Type of the DB instance.
	DBInstanceType string `default:"db.r4.large" validate:"oneof:{db.r4.large}"`
	// Type of the DB instance - postgres or mysql.
	DBInstanceEngine string `default:"aurora-postgresql" validate:"oneof:{aurora-mysql, aurora-postgresql}"`
	// Username to connect to the DB.
	DBUserName string `default:"mmuser" validate:"notempty"`
	// Password to connect to the DB.
	DBPassword string `default:"mostest80098bigpass_" validate:"notempty"`
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
	AdminUsername string `default:"sysadmin" validate:"notempty"`
	// Mattermost instance sysadmin password.
	AdminPassword string `default:"Sys@dmin-sample1" validate:"notempty"`
	// URL from where to download load-test-ng binaries and configuration files.
	// The configuration files provided in the package will be overridden in
	// the deployment process.
	LoadTestDownloadURL   string `default:"https://github.com/mattermost/mattermost-load-test-ng/releases/download/v0.5.0-alpha/mattermost-load-test-ng-v0.5.0-alpha-linux-amd64.tar.gz" validate:"url"`
	ElasticSearchSettings ElasticSearchSettings
	LogSettings           logger.Settings
	Report                report.Config
}

// ElasticSearchSettings contains the necessary data
// to configure an ElasticSearch instance to be deployed
// and provisioned.
type ElasticSearchSettings struct {
	// Elasticsearch instances number.
	InstanceCount int
	// Elasticsearch instance type to be created.
	InstanceType string
	// Elasticsearch version to be deployed.
	Version float64
	// Id of the VPC associated with the instance to be created.
	VpcID string
	// Set to true if the AWSServiceRoleForAmazonElasticsearchService role should be created.
	CreateRole bool
}

// IsValid reports whether a given deployment config is valid or not.
func (c *Config) IsValid() error {
	if !checkPrefix(c.MattermostDownloadURL) {
		return fmt.Errorf("mattermost download url is not in correct format: %q", c.MattermostDownloadURL)
	}

	if !checkPrefix(c.LoadTestDownloadURL) {
		return fmt.Errorf("load-test download url is not in correct format: %q", c.LoadTestDownloadURL)
	}

	if err := c.validateElasticSearchConfig(); err != nil {
		return err
	}
	return nil
}

func checkPrefix(str string) bool {
	return strings.HasPrefix(str, "https://") ||
		strings.HasPrefix(str, "http://") ||
		strings.HasPrefix(str, "file://")
}

func (c *Config) validateElasticSearchConfig() error {
	if (c.ElasticSearchSettings != ElasticSearchSettings{}) {
		if c.ElasticSearchSettings.InstanceCount > 1 {
			return errors.New("it is not possible to create more than 1 instance of Elasticsearch")
		}

		if c.ElasticSearchSettings.InstanceCount > 0 && c.ElasticSearchSettings.VpcID == "" {
			return errors.New("VpcID must be set in order to create an Elasticsearch instance")
		}

		domainName := c.ClusterName + "-es"
		if !esDomainNameRe.Match([]byte(domainName)) {
			return fmt.Errorf("Elasticsearch domain name must start with a lowercase alphabet and be at least " +
				"3 and no more than 28 characters long. Valid characters are a-z (lowercase letters), 0-9, and - " +
				"(hyphen). Current value is \"" + domainName + "\"")
		}

	}

	return nil
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will return a config with default values.
func ReadConfig(configFilePath string) (*Config, error) {
	var cfg Config

	if err := defaults.ReadFromJSON(configFilePath, "./config/deployer.json", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
