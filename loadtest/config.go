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

type ConnectionConfiguration struct {
	ServerURL                   string
	WebSocketURL                string
	AdminEmail                  string
	AdminPassword               string
	MaxIdleConns                int
	MaxIdleConnsPerHost         int
	IdleConnTimeoutMilliseconds int
}

// IsValid reports whether a given ConnectionConfiguration is valid or not.
func (cc *ConnectionConfiguration) IsValid() (bool, error) {
	if cc.ServerURL == "" {
		return false, fmt.Errorf("ServerURL is not present in config")
	}
	if cc.WebSocketURL == "" {
		return false, fmt.Errorf("WebSocketURL is not present in config")
	}
	if cc.AdminEmail == "" {
		return false, fmt.Errorf("AdminEmail is not present in config")
	}
	if cc.AdminPassword == "" {
		return false, fmt.Errorf("AdminPassword is not present in config")
	}
	return true, nil
}

// userControllerType describes the type of a UserController.
type userControllerType string

// Available UserController implementations.
const (
	UserControllerSimple     userControllerType = "simple"
	UserControllerSimulative                    = "simulative"
)

// IsValid reports whether a given UserControllerType is valid or not.
func (t userControllerType) IsValid() (bool, error) {
	switch t {
	case UserControllerSimple:
		return true, nil
	case UserControllerSimulative:
		return true, nil
	case "":
		return false, fmt.Errorf("UserControllerType cannot be empty")
	default:
		return false, fmt.Errorf("UserControllerType %s is not valid", t)
	}
}

// UserControllerConfiguration holds information about the UserController to
// run during a load-test.
type UserControllerConfiguration struct {
	// The type of the UserController to run.
	// Possible values:
	//   UserControllerSimple - A simple version of a controller.
	//   UserControllerSimulative - A more realistic controller.
	Type userControllerType
	// A rate multiplier that will affect the speed at which user actions are
	// executed by the UserController.
	// A Rate of < 1.0 will run actions at a faster pace.
	// A Rate of 1.0 will run actions at the default pace.
	// A Rate > 1.0 will run actions at a slower pace.
	Rate float64
}

// IsValid reports whether a given UserControllerConfiguration is valid or not.
func (ucc *UserControllerConfiguration) IsValid() (bool, error) {
	if ok, err := ucc.Type.IsValid(); !ok {
		return false, err
	}
	if ucc.Rate < 0 {
		return false, errors.New("Rate cannot be < 0")
	}
	return true, nil
}

type InstanceConfiguration struct {
	NumTeams int
}

// IsValid reports whether a given InstanceConfiguration is valid or not.
func (ic *InstanceConfiguration) IsValid() (bool, error) {
	if ic.NumTeams <= 0 {
		return false, fmt.Errorf("NumTeams cannot be <= 0")
	}
	return true, nil
}

type UsersConfiguration struct {
	InitialActiveUsers int
	MaxActiveUsers     int
}

// IsValid reports whether a given UsersConfiguration is valid or not.
func (uc *UsersConfiguration) IsValid() (bool, error) {
	if uc.InitialActiveUsers < 0 {
		return false, fmt.Errorf("InitialActiveUsers cannot be < 0")
	}
	if uc.MaxActiveUsers <= 0 {
		return false, fmt.Errorf("MaxActiveUsers cannot be <= 0")
	}
	if uc.InitialActiveUsers > uc.MaxActiveUsers {
		return false, fmt.Errorf("InitialActiveUsers cannot be greater than MaxActiveUsers")
	}
	return true, nil
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

type Config struct {
	ConnectionConfiguration     ConnectionConfiguration
	UserControllerConfiguration UserControllerConfiguration
	InstanceConfiguration       InstanceConfiguration
	UsersConfiguration          UsersConfiguration
	DeploymentConfiguration     DeploymentConfiguration
	LogSettings                 logger.Settings
}

// IsValid reports whether a config is valid or not.
func (c *Config) IsValid() (bool, error) {
	if valid, err := c.ConnectionConfiguration.IsValid(); !valid {
		return false, fmt.Errorf("invalid connection configuration: %w", err)
	}

	if valid, err := c.InstanceConfiguration.IsValid(); !valid {
		return false, fmt.Errorf("invalid instance configuration: %w", err)
	}

	if valid, err := c.UserControllerConfiguration.IsValid(); !valid {
		return false, fmt.Errorf("invalid user controller configuration: %w", err)
	}

	if valid, err := c.UsersConfiguration.IsValid(); !valid {
		return false, fmt.Errorf("invalid users configuration: %w", err)
	}

	// TODO: to be moved to its own config file.
	if c.DeploymentConfiguration.DBInstanceEngine != "" {
		switch c.DeploymentConfiguration.DBInstanceEngine {
		case "aurora", "aurora-postgresql", "mysql", "postgres":
		default:
			return false, fmt.Errorf("Invalid value %s for DBInstanceEngine", c.DeploymentConfiguration.DBInstanceEngine)
		}
	}

	return true, nil
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
