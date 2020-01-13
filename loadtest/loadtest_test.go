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

	"github.com/stretchr/testify/assert"
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

	lt = New(&ltConfig, nil)
	require.Nil(t, lt)

	lt = New(&ltConfig, newController)
	require.NotNil(t, lt)
}

func TestAddUser(t *testing.T) {
	lt := New(&ltConfig, newController)
	require.NotNil(t, lt)

	err := lt.AddUser()
	require.Equal(t, ErrNotRunning, err)

	lt.status.State = StateRunning

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
	lt := New(&ltConfig, newController)
	require.NotNil(t, lt)

	err := lt.RemoveUser()
	require.Equal(t, ErrNotRunning, err)

	lt.status.State = StateRunning

	err = lt.RemoveUser()
	require.Equal(t, ErrNoUsersLeft, err)

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
	require.Equal(t, lt.status.State, StateRunning)
	require.Len(t, lt.controllers, ltConfig.UsersConfiguration.InitialActiveUsers)

	err = lt.Run()
	require.Equal(t, ErrNotStopped, err)

	err = lt.Stop()
	require.NoError(t, err)
}

func TestRerun(t *testing.T) {
	lt := New(&ltConfig, newController)
	err := lt.Run()
	require.NoError(t, err)

	err = lt.Stop()
	require.NoError(t, err)

	err = lt.Run()
	require.NoError(t, err)

	err = lt.Stop()
	require.NoError(t, err)
}

func TestStop(t *testing.T) {
	lt := New(&ltConfig, newController)
	err := lt.Stop()
	require.Equal(t, ErrNotRunning, err)

	err = lt.Run()
	require.NoError(t, err)
	lt.status.State = StateRunning

	numUsers := 8
	for i := 0; i < numUsers; i++ {
		err = lt.AddUser()
		require.NoError(t, err)
	}
	err = lt.Stop()
	require.NoError(t, err)
	require.Equal(t, lt.status.State, StateStopped)
	require.Empty(t, lt.controllers)
}

func TestStatus(t *testing.T) {
	lt := New(&ltConfig, newController)
	require.NotNil(t, lt)

	err := lt.Run()
	require.NoError(t, err)
	st := lt.Status()
	startTime := st.StartTime
	assert.Equal(t, StateRunning, st.State)
	assert.Equal(t, 0, st.NumUsers)
	assert.Equal(t, 0, st.NumUsersAdded)
	assert.Equal(t, 0, st.NumUsersRemoved)

	err = lt.AddUser()
	require.NoError(t, err)
	st = lt.Status()
	assert.Equal(t, StateRunning, st.State)
	assert.Equal(t, 1, st.NumUsers)
	assert.Equal(t, 1, st.NumUsersAdded)
	assert.Equal(t, 0, st.NumUsersRemoved)

	err = lt.RemoveUser()
	require.NoError(t, err)
	st = lt.Status()
	assert.Equal(t, StateRunning, st.State)
	assert.Equal(t, 0, st.NumUsers)
	assert.Equal(t, 1, st.NumUsersAdded)
	assert.Equal(t, 1, st.NumUsersRemoved)

	err = lt.Stop()
	require.NoError(t, err)
	st = lt.Status()
	assert.Equal(t, StateStopped, st.State)
	assert.Equal(t, 0, st.NumUsers)
	assert.Equal(t, 1, st.NumUsersAdded)
	assert.Equal(t, 1, st.NumUsersRemoved)

	// Start again, and verify that start time got reset.
	err = lt.Run()
	require.NoError(t, err)
	st = lt.Status()
	assert.True(t, startTime.Before(st.StartTime))
	assert.Equal(t, StateRunning, st.State)
}
