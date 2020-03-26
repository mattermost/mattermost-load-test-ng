// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package deployment

import (
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/spf13/viper"
)

// Config contains the necessary data
// to deploy and provision a load test environment.
type Config struct {
	ClusterName      string // Name of the cluster.
	AppInstanceCount int    // Number of application instances.
	SSHPublicKey     string // Path to the SSH public key.
	DBInstanceCount  int    // Number of DB instances.
	DBInstanceClass  string // Type of the DB instance.
	DBInstanceEngine string // Type of the DB instance - postgres or mysql.
	DBUserName       string // Username to connect to the DB.
	DBPassword       string // Password to connect to the DB.
	// URL from where to download Mattermost release.
	// This can also point to a local binary path if the user wants to run loadtest
	// on a custom build. The path should be prefixed with "file://". In that case,
	// only the binary gets replaced, and the rest of the build comes from the latest
	// stable release.
	MattermostDownloadURL string
	MattermostLicenseFile string // Path to the Mattermost EE license file.
	LogSettings           logger.Settings
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

	return nil
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will search a config file in predefined folders.
func ReadConfig(filePath string) (*Config, error) {
	v := viper.New()

	v.SetConfigName("deployer")
	v.AddConfigPath(".")
	v.AddConfigPath("./config/")
	v.AddConfigPath("./../../../config/")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("LogSettings.EnableConsole", true)
	v.SetDefault("LogSettings.ConsoleLevel", "INFO")
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
