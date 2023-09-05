// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package cluster

import (
	"errors"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
)

// LoadAgentConfig holds information about the load-test agent instance.
type LoadAgentConfig struct {
	// A sring that identifies the load-test agent instance.
	Id string `default:"lt0" validate:"notempty"`
	// The API URL used to control the specified load-test instance.
	ApiURL string `default:"http://localhost:4000" validate:"url"`
}

// LoadAgentClusterConfig holds information regarding the cluster of load-test
// agents.
type LoadAgentClusterConfig struct {
	// Agents is a list of the load-test agents API endpoints to be used during
	// the load-test. It's length defines the number of load-test instances
	// used during a load-test.
	Agents []LoadAgentConfig `default_size:"1"`
	// MaxActiveUsers defines the upper limit of concurrently active users to run across
	// the whole cluster.
	MaxActiveUsers int `default:"1000" validate:"range:(0,]"`
}

func (c *LoadAgentClusterConfig) IsValid(ltConfig loadtest.Config) error {
	if ltConfig.UsersConfiguration.MaxActiveUsers*len(c.Agents) < c.MaxActiveUsers {
		return errors.New("coordinator: total MaxActiveUsers in loadTest should not be less than clusterConfig.MaxActiveUsers")
	}

	ok, err := checkMaxUsersPerTeam(ltConfig, *c)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("coordinator: TeamSettings.MaxUsersPerTeam should not be less than clusterConfig.MaxActiveUsers")
	}

	return nil
}

func checkMaxUsersPerTeam(config loadtest.Config, cConfig LoadAgentClusterConfig) (bool, error) {
	adminStore, err := memstore.New(nil)
	if err != nil {
		return false, err
	}
	adminUeSetup := userentity.Setup{
		Store: adminStore,
	}
	adminUeConfig := userentity.Config{
		ServerURL:    config.ConnectionConfiguration.ServerURL,
		WebSocketURL: config.ConnectionConfiguration.WebSocketURL,
		Username:     "",
		Email:        config.ConnectionConfiguration.AdminEmail,
		Password:     config.ConnectionConfiguration.AdminPassword,
	}
	sysadmin := userentity.New(adminUeSetup, adminUeConfig)
	if err := sysadmin.Login(); err != nil {
		return false, err
	}

	// Load the config
	if err := sysadmin.GetConfig(); err != nil {
		return false, err
	}

	if cConfig.MaxActiveUsers > *sysadmin.Store().Config().TeamSettings.MaxUsersPerTeam {
		return false, nil
	}

	return true, nil
}
