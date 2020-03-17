// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package loadtest

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ltConfig = Config{
	ConnectionConfiguration: ConnectionConfiguration{
		ServerURL:     "http://localhost:8065",
		WebSocketURL:  "ws://localhost:8065",
		AdminEmail:    "user@example.com",
		AdminPassword: "str0ngPassword##",
	},
	UserControllerConfiguration: UserControllerConfiguration{
		Type: "simple",
		Rate: 1.0,
	},
	UsersConfiguration: UsersConfiguration{
		MaxActiveUsers:     8,
		InitialActiveUsers: 0,
	},
	InstanceConfiguration: InstanceConfiguration{
		NumTeams: 1,
	},
	LogSettings: logger.Settings{},
}

func newController(id int, status chan<- control.UserStatus) (control.UserController, error) {
	ueConfig := userentity.Config{
		ServerURL:    ltConfig.ConnectionConfiguration.ServerURL,
		WebSocketURL: ltConfig.ConnectionConfiguration.WebSocketURL,
	}
	ue := userentity.New(memstore.New(), ueConfig)
	cfg, err := simplecontroller.ReadConfig("")
	if err != nil {
		return nil, err
	}

	return simplecontroller.New(id, ue, cfg, status)

}

func TestNew(t *testing.T) {
	// ignore lt structs if there is an error.
	_, err := New(nil, newController)
	require.NotNil(t, err)

	_, err = New(&ltConfig, nil)
	require.NotNil(t, err)

	lt, err := New(&ltConfig, newController)
	require.Nil(t, err)
	require.NotNil(t, lt)
}

func TestAddUser(t *testing.T) {
	lt, err := New(&ltConfig, newController)
	require.Nil(t, err)

	err = lt.AddUser()
	require.Equal(t, ErrNotRunning, err)

	lt.status.State = Running

	ltConfig.UsersConfiguration.MaxActiveUsers = 0
	err = lt.AddUser()
	require.Equal(t, ErrMaxUsersReached, err)
	ltConfig.UsersConfiguration.MaxActiveUsers = 8

	numUsers := 8
	for i := 0; i < numUsers; i++ {
		err = lt.AddUser()
		require.NoError(t, err)
	}

	require.Len(t, lt.controllers, numUsers)
}

func TestRemoveUser(t *testing.T) {
	lt, err := New(&ltConfig, newController)
	require.Nil(t, err)

	err = lt.RemoveUser()
	require.Equal(t, ErrNotRunning, err)

	lt.status.State = Running

	err = lt.RemoveUser()
	require.Equal(t, ErrNoUsersLeft, err)

	err = lt.AddUser()
	require.NoError(t, err)

	err = lt.RemoveUser()
	require.NoError(t, err)
	require.Empty(t, lt.controllers)
}

func TestRun(t *testing.T) {
	lt, err := New(&ltConfig, newController)
	require.Nil(t, err)
	err = lt.Run()
	require.NoError(t, err)
	require.Equal(t, lt.status.State, Running)
	require.Len(t, lt.controllers, ltConfig.UsersConfiguration.InitialActiveUsers)

	err = lt.Run()
	require.Equal(t, ErrNotStopped, err)

	err = lt.Stop()
	require.NoError(t, err)
}

func TestRerun(t *testing.T) {
	lt, err := New(&ltConfig, newController)
	require.Nil(t, err)
	err = lt.Run()
	require.NoError(t, err)

	err = lt.Stop()
	require.NoError(t, err)

	err = lt.Run()
	require.NoError(t, err)

	err = lt.Stop()
	require.NoError(t, err)
}

func TestStop(t *testing.T) {
	lt, err := New(&ltConfig, newController)
	require.Nil(t, err)
	err = lt.Stop()
	require.Equal(t, ErrNotRunning, err)

	err = lt.Run()
	require.NoError(t, err)
	lt.status.State = Running

	numUsers := 8
	for i := 0; i < numUsers; i++ {
		err = lt.AddUser()
		require.NoError(t, err)
	}
	err = lt.Stop()
	require.NoError(t, err)
	require.Equal(t, lt.status.State, Stopped)
	require.Empty(t, lt.controllers)
}

func TestStatus(t *testing.T) {
	lt, err := New(&ltConfig, newController)
	require.NotNil(t, lt)
	require.Nil(t, err)

	err = lt.Run()
	require.NoError(t, err)
	st := lt.Status()
	startTime := st.StartTime
	assert.Equal(t, Running, st.State)
	assert.Equal(t, 0, st.NumUsers)
	assert.Equal(t, 0, st.NumUsersAdded)
	assert.Equal(t, 0, st.NumUsersRemoved)

	err = lt.AddUser()
	require.NoError(t, err)
	st = lt.Status()
	assert.Equal(t, Running, st.State)
	assert.Equal(t, 1, st.NumUsers)
	assert.Equal(t, 1, st.NumUsersAdded)
	assert.Equal(t, 0, st.NumUsersRemoved)

	err = lt.RemoveUser()
	require.NoError(t, err)
	st = lt.Status()
	assert.Equal(t, Running, st.State)
	assert.Equal(t, 0, st.NumUsers)
	assert.Equal(t, 1, st.NumUsersAdded)
	assert.Equal(t, 1, st.NumUsersRemoved)

	err = lt.Stop()
	require.NoError(t, err)
	st = lt.Status()
	assert.Equal(t, Stopped, st.State)
	assert.Equal(t, 0, st.NumUsers)
	assert.Equal(t, 1, st.NumUsersAdded)
	assert.Equal(t, 1, st.NumUsersRemoved)

	// Start again, and verify that start time got reset.
	err = lt.Run()
	require.NoError(t, err)
	st = lt.Status()
	assert.True(t, startTime.Before(st.StartTime))
	assert.Equal(t, Running, st.State)
}
