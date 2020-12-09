// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package comparison

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
)

type LoadTestType string
type DatabaseEngine string

const (
	DBEngineMySQL DatabaseEngine = "mysql"
	DBEnginePgSQL DatabaseEngine = "postgresql"
)

const (
	LoadTestTypeBounded   LoadTestType = "bounded"
	LoadTestTypeUnbounded LoadTestType = "unbounded"
)

// LoadTestConfig holds information about a load-test
// to be automated.
type LoadTestConfig struct {
	// The type of load-test to run.
	Type LoadTestType `validate:"oneof:{bounded,unbounded}"`
	// The database engine for the app server.
	DBEngine DatabaseEngine `validate:"oneof:{mysql,postgresql}"`

	// The number of users to run.
	// This is only considered if Type is "bounded"
	NumUsers int `default:"0" validate:"range:[0,]"`
	// The duration of the load-test.
	// This is only considered if Type is "bounded"
	Duration string
}

// IsValid reports whether a given LoadTestConfig is valid or not.
// Returns an error if the validation fails.
func (c *LoadTestConfig) IsValid() error {
	if c.Type == LoadTestTypeBounded {
		if _, err := time.ParseDuration(c.Duration); err != nil {
			return fmt.Errorf("failed to parse Duration: %w", err)
		}
	}

	return nil
}

// BuildConfig holds information about a build.
type BuildConfig struct {
	// A label identifying the build
	Label string `validate:"notempty"`

	// URL from where to download a build release.
	// This can also point to a local file if prefixed with "file://".
	// In such case, the build file will be uploaded to the app servers.
	URL string `validate:"url"`
}

// OutputConfig defines settings for the output of the comparison.
type OutputConfig struct {
	// A boolean indicating whether a comparative Grafana dashboard should
	// be generated and uploaded.
	UploadDashboard bool `default:"true"`
	// A boolean indicating whether to generate a markdown report
	// at the end of the comparison.
	GenerateReport bool `default:"true"`
	// A boolean indicating whether to generate gnuplot graphs
	// at the end of the comparison.
	GenerateGraphs bool `default:"false"`
	// An optional path indicating where to write the graphs.
	GraphsPath string
}

// Config holds information needed perform automated load-test comparisons.
type Config struct {
	BaseBuild BuildConfig
	NewBuild  BuildConfig
	LoadTests []LoadTestConfig `validate:"notempty"`
	Output    OutputConfig
}

func (c *Config) IsValid() error {
	for _, ltConfig := range c.LoadTests {
		if err := defaults.Validate(ltConfig); err != nil {
			return err
		}
	}
	return nil
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will return a config with default values.
func ReadConfig(configFilePath string) (*Config, error) {
	var cfg Config

	if err := defaults.ReadFromJSON(configFilePath, "./config/comparison.json", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
