// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"

	"github.com/stretchr/testify/require"
)

var ltConfig = config.LoadTestConfig{
	ConnectionConfiguration: config.ConnectionConfiguration{
		ServerURL:    "http://localhost:8065",
		WebSocketURL: "ws://localhost:8065",
	},
	UsersConfiguration: config.UsersConfiguration{
		MaxActiveUsers:     8,
		InitialActiveUsers: 0,
	},
	LogSettings: config.LoggerSettings{},
}

func newController(id int, status chan<- control.UserStatus) control.UserController {
	ueConfig := userentity.Config{
		ServerURL:    ltConfig.ConnectionConfiguration.ServerURL,
		WebSocketURL: ltConfig.ConnectionConfiguration.WebSocketURL,
	}
	ue := userentity.New(memstore.New(), ueConfig)
	return simplecontroller.New(id, ue, status)
}

func TestNew(t *testing.T) {
	lt := New(nil, newController)
	require.Nil(t, lt)

	lt = New(&ltConfig, newController)
	require.NotNil(t, lt)
}

func TestAddUser(t *testing.T) {
	lt := New(&ltConfig, newController)
	require.NotNil(t, lt)

	err := lt.AddUser()
	require.EqualError(t, err, "LoadTester is not running")

	lt.started = true

	ltConfig.UsersConfiguration.MaxActiveUsers = 0
	err = lt.AddUser()
	require.EqualError(t, err, "Max active users limit reached")
	ltConfig.UsersConfiguration.MaxActiveUsers = 8

	numUsers := 8
	for i := 0; i < numUsers; i++ {
		err = lt.AddUser()
		require.NoError(t, err)
	}

	require.Len(t, lt.controllers, numUsers)
}

func TestRemoveUser(t *testing.T) {
	lt := New(&ltConfig, newController)
	require.NotNil(t, lt)

	err := lt.RemoveUser()
	require.EqualError(t, err, "LoadTester is not running")

	lt.started = true

	err = lt.RemoveUser()
	require.EqualError(t, err, "No active users left")

	err = lt.AddUser()
	require.NoError(t, err)

	err = lt.RemoveUser()
	require.NoError(t, err)
	require.Empty(t, lt.controllers)
}

func TestRun(t *testing.T) {
	lt := New(&ltConfig, newController)
	err := lt.Run()
	require.NoError(t, err)
	require.True(t, lt.started)
	require.Len(t, lt.controllers, ltConfig.UsersConfiguration.InitialActiveUsers)
	err = lt.Run()
	require.EqualError(t, err, "LoadTester is already running")
}

func TestStop(t *testing.T) {
	lt := New(&ltConfig, newController)
	err := lt.Stop()
	require.EqualError(t, err, "LoadTester is not running")

	err = lt.Run()
	require.NoError(t, err)
	require.True(t, lt.started)

	err = lt.Stop()
	require.NoError(t, err)
	require.False(t, lt.started)
	require.Empty(t, lt.controllers)
}
