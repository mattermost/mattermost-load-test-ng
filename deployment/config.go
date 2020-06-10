// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package deployment

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/spf13/viper"
)

var esDomainNameRe = regexp.MustCompile(`^[a-z][a-z0-9\-]{2,27}$`)

// Config contains the necessary data
// to deploy and provision a load test environment.
type Config struct {
	ClusterName           string // Name of the cluster.
	AppInstanceCount      int    // Number of application instances.
	AppInstanceType       string // Type of the EC2 instance for app.
	AgentInstanceCount    int    // Number of agents, first agent and coordinator will share the same instance.
	AgentInstanceType     string // Type of the EC2 instance for agent.
	ElasticSearchSettings ElasticSearchSettings
	EnableAgentFullLogs   bool   // Logs the command output (stdout & stderr) to home directory.
	ProxyInstanceType     string // Type of the EC2 instance for proxy.
	SSHPublicKey          string // Path to the SSH public key.
	DBInstanceCount       int    // Number of DB instances.
	DBInstanceType        string // Type of the DB instance.
	DBInstanceEngine      string // Type of the DB instance - postgres or mysql.
	DBUserName            string // Username to connect to the DB.
	DBPassword            string // Password to connect to the DB.
	// URL from where to download Mattermost release.
	// This can also point to a local binary path if the user wants to run loadtest
	// on a custom build. The path should be prefixed with "file://". In that case,
	// only the binary gets replaced, and the rest of the build comes from the latest
	// stable release.
	MattermostDownloadURL string
	MattermostLicenseFile string // Path to the Mattermost EE license file.
	AdminEmail            string // Mattermost instance sysadmin e-mail.
	AdminUsername         string // Mattermost instance sysadmin user name.
	AdminPassword         string // Mattermost instance sysadmin password.
	// URL from where to download load-test-ng binaries and configuration files.
	// The configuration files provided in the package will be overridden in
	// the deployment process.
	LoadTestDownloadURL string
	LogSettings         logger.Settings
}

// ElasticSearchSettings contains the necessary data
// to configure an ElasticSearch instance to be deployed
// and provisioned.
type ElasticSearchSettings struct {
	InstanceCount int    // Elasticsearch instances number.
	InstanceType  string // Elasticsearch instance type to be created.
	Version       string // Elasticsearch version to be deployed.
	VpcID         string // Id of the VPC associated with the instance to be created.
	CreateRole    bool   // Set to true if the AWSServiceRoleForAmazonElasticsearchService role should be created.
}

// IsValid reports whether a given deployment config is valid or not.
func (c *Config) IsValid() error {
	if _, err := os.Stat(c.MattermostLicenseFile); os.IsNotExist(err) {
		return fmt.Errorf("license file %q doesn't exist", c.MattermostLicenseFile)
	}

	if c.DBInstanceEngine != "" {
		switch c.DBInstanceEngine {
		case "aurora-mysql", "aurora-postgresql":
		default:
			return fmt.Errorf("invalid value %s for DBInstanceEngine", c.DBInstanceEngine)
		}
	}

	if len(c.DBPassword) < 8 {
		return fmt.Errorf("db password needs to be at least 8 characters")
	}
	clusterName := c.ClusterName
	firstRune, _ := utf8.DecodeRuneInString(clusterName)
	if len(clusterName) == 0 || !unicode.IsLetter(firstRune) || !isAlphanumeric(clusterName) {
		return fmt.Errorf("db cluster name must begin with a letter and contain only alphanumeric characters")
	}
	if c.AgentInstanceCount < 1 {
		return fmt.Errorf("at least 1 agent is required to run load tests")
	}
	if !strings.HasSuffix(c.LoadTestDownloadURL, ".tar.gz") {
		return fmt.Errorf("load-test package file must be a tar.gz file")
	}

	if err := c.validateElasticSearchConfig(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateElasticSearchConfig() error {
	if (c.ElasticSearchSettings != ElasticSearchSettings{}) {
		if c.ElasticSearchSettings.InstanceCount > 1 {
			return fmt.Errorf("it is not possible to create more than 1 instance of Elasticsearch")
		}

		if c.ElasticSearchSettings.InstanceCount > 0 && c.ElasticSearchSettings.VpcID == "" {
			return fmt.Errorf("VpcID must be set in order to create an Elasticsearch instance")
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
// is empty, it will search a config file in predefined folders.
func ReadConfig(filePath string) (*Config, error) {
	v := viper.New()

	v.SetConfigName("deployer")
	v.AddConfigPath(".")
	v.AddConfigPath("./config/")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("LogSettings.EnableConsole", true)
	v.SetDefault("LogSettings.ConsoleLevel", "ERROR")
	v.SetDefault("LogSettings.ConsoleJson", false)
	v.SetDefault("LogSettings.EnableFile", true)
	v.SetDefault("LogSettings.FileLevel", "INFO")
	v.SetDefault("LogSettings.FileJson", true)
	v.SetDefault("LogSettings.FileLocation", "loadtest.log")

	if filePath != "" {
		v.SetConfigFile(filePath)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("unable to read configuration file: %w", err)
	}

	var cfg *Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
