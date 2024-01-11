// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package loadtest

import (
	"errors"
	"math"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/logger"
)

// ConnectionConfiguration holds information needed to connect to the instance.
type ConnectionConfiguration struct {
	// URL of the instance to connect to.
	ServerURL string `default:"http://localhost:8065" validate:"url"`
	// WebSocket URL of the instance to connect to.
	WebSocketURL string `default:"ws://localhost:8065" validate:"url"`
	// Email of the system admin.
	AdminEmail string `default:"sysadmin@sample.mattermost.com" validate:"email"`
	// Password of the system admin.
	AdminPassword string `default:"Sys@dmin-sample1" validate:"notempty"`
}

// userControllerType describes the type of a UserController.
type userControllerType string

// Available UserController implementations.
const (
	UserControllerSimple     userControllerType = "simple"
	UserControllerSimulative                    = "simulative"
	UserControllerNoop                          = "noop"
	UserControllerGenerative                    = "generative"
	UserControllerCluster                       = "cluster"
)

// RatesDistribution maps a rate to a percentage of controllers that should run
// at that rate.
type RatesDistribution struct {
	Rate       float64 `default:"1.0" validate:"range:[0,)"`
	Percentage float64 `default:"1.0" validate:"range:[0,1]"`
}

// UserControllerConfiguration holds information about the UserController to
// run during a load-test.
type UserControllerConfiguration struct {
	// The type of the UserController to run.
	// Possible values:
	//   UserControllerSimple - A simple version of a controller.
	//   UserControllerSimulative - A more realistic controller.
	//   UserControllerNoop
	//   UserControllerGenerative - A controller used to generate data.
	Type userControllerType `default:"simulative" validate:"oneof:{simple,simulative,noop,cluster,generative}"`
	// A distribution of rate multipliers that will affect the speed at which user actions are
	// executed by the UserController.
	// A Rate of < 1.0 will run actions at a faster pace.
	// A Rate of 1.0 will run actions at the default pace.
	// A Rate > 1.0 will run actions at a slower pace.
	RatesDistribution []RatesDistribution `default_len:"1"`
	// An optional MM server version to use when running actions (e.g. `5.30.0`).
	// This value overrides the actual server version. If left empty,
	// the one returned by the server is used instead.
	ServerVersion string
}

// IsValid reports whether a given UserControllerConfiguration is valid or not.
// Returns an error if the validation fails.
func (ucc *UserControllerConfiguration) IsValid() error {
	var sum float64
	for _, el := range ucc.RatesDistribution {
		sum += el.Percentage
	}
	if len(ucc.RatesDistribution) > 0 && sum != 1 {
		return errors.New("Percentages in RatesDistribution should sum to 1")
	}
	return nil
}

// InstanceConfiguration holds information about the data to be populated
// during the init process.
type InstanceConfiguration struct {
	// The target number of teams to be created.
	NumTeams int64 `default:"2" validate:"range:[0,]"`
	// The target number of channels to be created.
	NumChannels int64 `default:"10" validate:"range:[0,]"`
	// The target number of posts to be created.
	NumPosts int64 `default:"0" validate:"range:[0,]"`
	// The target number of reactions to be created.
	NumReactions int64 `default:"0" validate:"range:[0,]"`
	// The target number of admin users to be created.
	NumAdmins int64 `default:"0" validate:"range:[0,]"`

	// The percentage of replies to be created.
	PercentReplies float64 `default:"0.5" validate:"range:[0,1]"`
	// The percentage of replies that should be in long threads
	PercentRepliesInLongThreads float64 `default:"0.05" validate:"range:[0,1]"`
	// The percentage of post that are marked as urgent
	PercentUrgentPosts float64 `default:"0.001" validate:"range:[0,1]"`

	// Percentages of channels to be created, grouped by type.
	// The total sum of these values must be equal to 1.

	// The percentage of public channels to be created.
	PercentPublicChannels float64 `default:"0.2" validate:"range:[0,1]"`
	// The percentage of private channels to be created.
	PercentPrivateChannels float64 `default:"0.1" validate:"range:[0,1]"`
	// The percentage of direct channels to be created.
	PercentDirectChannels float64 `default:"0.6" validate:"range:[0,1]"`
	// The percentage of group channels to be created.
	PercentGroupChannels float64 `default:"0.1" validate:"range:[0,1]"`
}

// IsValid reports whether a given InstanceConfiguration is valid or not.
// Returns an error if the validation fails.
func (c *InstanceConfiguration) IsValid() error {
	percentChannels := c.PercentPublicChannels + c.PercentPrivateChannels + c.PercentDirectChannels + c.PercentGroupChannels
	if (math.Round(percentChannels*100) / 100) != 1 {
		return errors.New("sum of percentages for channels should be equal to 1")
	}

	return nil
}

// UsersConfiguration holds information about the users of the load-test.
type UsersConfiguration struct {
	// The file which contains the user emails and passwords in case the operator
	// wants to login using a different set of credentials. This is helpful during
	// LDAP logins.
	UsersFilePath string
	// The number of initial users the load-test should start with.
	InitialActiveUsers int `default:"0" validate:"range:[0,$MaxActiveUsers]"`
	// The maximum number of users that can be simulated by a single load-test
	// agent.
	MaxActiveUsers int `default:"2000" validate:"range:(0,]"`
	// The average number of sessions per user.
	AvgSessionsPerUser int `default:"1" validate:"range:[1,]"`
	// The percentage of users generated that will be system admins
	PercentOfUsersAreAdmin float64 `default:"0.02" validate:"range:[0,1]"`
}

// Config holds information needed to create and initialize a new load-test
// agent.
type Config struct {
	ConnectionConfiguration     ConnectionConfiguration
	UserControllerConfiguration UserControllerConfiguration
	InstanceConfiguration       InstanceConfiguration
	UsersConfiguration          UsersConfiguration
	LogSettings                 logger.Settings
}

// IsValid reports whether a given Config is valid or not.
// Returns an error if the validation fails.
func (c *Config) IsValid() error {
	if err := c.UserControllerConfiguration.IsValid(); err != nil {
		return err
	}
	if err := c.InstanceConfiguration.IsValid(); err != nil {
		return err
	}
	return nil
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will return a config with default values.
func ReadConfig(configFilePath string) (*Config, error) {
	var cfg Config

	if err := defaults.ReadFromJSON(configFilePath, "./config/config.json", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
