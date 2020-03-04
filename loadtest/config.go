// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package loadtest

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type Config struct {
	ConnectionConfiguration ConnectionConfiguration
	InstanceConfiguration   InstanceConfiguration
	UsersConfiguration      UsersConfiguration
	DeploymentConfiguration DeploymentConfiguration
	LogSettings             logger.Settings
}

type ConnectionConfiguration struct {
	ServerURL                   string
	WebSocketURL                string
	DriverName                  string
	DataSource                  string
	AdminEmail                  string
	AdminPassword               string
	MaxIdleConns                int
	MaxIdleConnsPerHost         int
	IdleConnTimeoutMilliseconds int
}

type InstanceConfiguration struct {
	NumTeams int
}

type UsersConfiguration struct {
	InitialActiveUsers int
	MaxActiveUsers     int
}

// DeploymentConfiguration contains the necessary data
// to deploy and provision a load test environment through terraform.
type DeploymentConfiguration struct {
	ClusterName           string // Name of the cluster.
	AppInstanceCount      int    // Number of application instances.
	SSHPublicKey          string // Path to the SSH public key.
	DBInstanceCount       int    // Number of DB instances.
	DBInstanceClass       string // Type of the DB instance.
	DBInstanceEngine      string // Type of the DB instance - postgres or mysql.
	DBUserName            string // Username to connect to the DB.
	DBPassword            string // Password to connect to the DB.
	MattermostDownloadURL string // URL from where to download Mattermost distribution.
	MattermostLicenseFile string // Path to the Mattermost EE license file.
}

func ReadConfig(configFilePath string) error {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config/")
	viper.SetEnvPrefix("mmloadtest")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("LogSettings.EnableConsole", true)
	viper.SetDefault("LogSettings.ConsoleLevel", "INFO")
	viper.SetDefault("LogSettings.ConsoleJson", true)
	viper.SetDefault("LogSettings.EnableFile", true)
	viper.SetDefault("LogSettings.FileLevel", "INFO")
	viper.SetDefault("LogSettings.FileJson", true)
	viper.SetDefault("LogSettings.FileLocation", "loadtest.log")
	viper.SetDefault("ConnectionConfiguration.MaxIdleConns", 100)
	viper.SetDefault("ConnectionConfiguration.MaxIdleConnsPerHost", 128)
	viper.SetDefault("ConnectionConfiguration.IdleConnTimeoutMilliseconds", 90000)

	if configFilePath != "" {
		viper.SetConfigFile(configFilePath)
	}

	if err := viper.ReadInConfig(); err != nil {
		return errors.Wrap(err, "unable to read configuration file")
	}

	return nil
}

func GetConfig() (*Config, error) {
	var cfg *Config

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// IsValid checks whether a config is valid or not.
func (c *Config) IsValid() (bool, error) {
	if c.ConnectionConfiguration.ServerURL == "" {
		return false, fmt.Errorf("ServerURL is not present in config")
	}
	if c.ConnectionConfiguration.WebSocketURL == "" {
		return false, fmt.Errorf("WebSocketURL is not present in config")
	}

	if c.DeploymentConfiguration.DBInstanceEngine != "" {
		switch c.DeploymentConfiguration.DBInstanceEngine {
		case "aurora", "aurora-postgresql", "mysql", "postgres":
		default:
			return false, fmt.Errorf("Invalid value %s for DBInstanceEngine", c.DeploymentConfiguration.DBInstanceEngine)
		}
	}

	return true, nil
}
