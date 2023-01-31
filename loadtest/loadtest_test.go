// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package loadtest

import (
	"testing"
	"time"

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
	},
	UsersConfiguration: UsersConfiguration{
		MaxActiveUsers:     8,
		InitialActiveUsers: 0,
		AvgSessionsPerUser: 1,
	},
	InstanceConfiguration: InstanceConfiguration{
		NumTeams:                    1,
		NumChannels:                 10,
		NumPosts:                    100,
		NumReactions:                50,
		NumAdmins:                   2,
		PercentReplies:              0.5,
		PercentRepliesInLongThreads: 0.05,
		PercentUrgentPosts:          0.001,
		PercentPublicChannels:       0.2,
		PercentPrivateChannels:      0.1,
		PercentDirectChannels:       0.6,
		PercentGroupChannels:        0.1,
	},
	LogSettings: logger.Settings{
		ConsoleLevel: "ERROR",
		FileLevel:    "ERROR",
	},
}

func newController(id int, status chan<- control.UserStatus) (control.UserController, error) {
	ueConfig := userentity.Config{
		ServerURL:    ltConfig.ConnectionConfiguration.ServerURL,
		WebSocketURL: ltConfig.ConnectionConfiguration.WebSocketURL,
	}
	store, err := memstore.New(nil)
	if err != nil {
		return nil, err
	}
	ue := userentity.New(userentity.Setup{Store: store}, ueConfig)
	cfg, err := simplecontroller.ReadConfig("")
	if err != nil {
		return nil, err
	}

	return simplecontroller.New(id, ue, cfg, status)
}

func TestNew(t *testing.T) {
	log := logger.New(&ltConfig.LogSettings)

	lt, err := New(nil, newController, log)
	require.NotNil(t, err)
	require.Nil(t, lt)

	lt, err = New(&ltConfig, nil, log)
	require.Error(t, err)
	require.Nil(t, lt)

	lt, err = New(&ltConfig, newController, nil)
	require.Error(t, err)
	require.Nil(t, lt)

	lt, err = New(&ltConfig, newController, log)
	require.NoError(t, err)
	require.NotNil(t, lt)
}

func TestAddUsers(t *testing.T) {
	log := logger.New(&ltConfig.LogSettings)

	lt, err := New(&ltConfig, newController, log)
	require.Nil(t, err)

	n, err := lt.AddUsers(0)
	require.Equal(t, ErrInvalidNumUsers, err)
	require.Zero(t, n)

	n, err = lt.AddUsers(1)
	require.Equal(t, ErrNotRunning, err)
	require.Zero(t, n)

	lt.status.State = Running

	ltConfig.UsersConfiguration.MaxActiveUsers = 4
	n, err = lt.AddUsers(8)
	require.Equal(t, ErrMaxUsersReached, err)
	require.Equal(t, 4, n)
	ltConfig.UsersConfiguration.MaxActiveUsers = 8

	n, err = lt.AddUsers(4)
	require.NoError(t, err)
	require.Equal(t, 4, n)

	require.Len(t, lt.activeControllers, 8)
	require.Empty(t, lt.idleControllers)
}

func TestRemoveUsers(t *testing.T) {
	log := logger.New(&ltConfig.LogSettings)
	lt, err := New(&ltConfig, newController, log)
	require.NoError(t, err)
	defer close(lt.statusChan)

	n, err := lt.RemoveUsers(0)
	require.Equal(t, ErrInvalidNumUsers, err)
	require.Zero(t, n)

	n, err = lt.RemoveUsers(1)
	require.Equal(t, ErrNotRunning, err)
	require.Zero(t, n)

	lt.status.State = Running
	// This is needed to avoid the controllers deadlocking
	// on sending data into the status channel.
	go func() {
		for range lt.statusChan {
		}
	}()

	n, err = lt.RemoveUsers(1)
	require.Equal(t, ErrNoUsersLeft, err)
	require.Zero(t, n)
	require.Empty(t, lt.idleControllers)
	require.Empty(t, lt.activeControllers)

	n, err = lt.AddUsers(1)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Empty(t, lt.idleControllers)
	require.Len(t, lt.activeControllers, 1)

	n, err = lt.RemoveUsers(1)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Empty(t, lt.activeControllers)
	require.Len(t, lt.idleControllers, 1)

	n, err = lt.AddUsers(2)
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Empty(t, lt.idleControllers)
	require.Len(t, lt.activeControllers, 2)

	n, err = lt.RemoveUsers(3)
	require.Equal(t, err, ErrNoUsersLeft)
	require.Equal(t, 2, n)
	require.Empty(t, lt.activeControllers)
	require.Len(t, lt.idleControllers, 2)
}

func TestRun(t *testing.T) {
	log := logger.New(&ltConfig.LogSettings)
	lt, err := New(&ltConfig, newController, log)
	require.Nil(t, err)
	err = lt.Run()
	require.NoError(t, err)
	require.Equal(t, lt.status.State, Running)
	require.Len(t, lt.activeControllers, ltConfig.UsersConfiguration.InitialActiveUsers)

	err = lt.Run()
	require.Equal(t, ErrNotStopped, err)

	err = lt.Stop()
	require.NoError(t, err)
}

func TestRerun(t *testing.T) {
	log := logger.New(&ltConfig.LogSettings)
	lt, err := New(&ltConfig, newController, log)
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
	log := logger.New(&ltConfig.LogSettings)
	lt, err := New(&ltConfig, newController, log)
	require.Nil(t, err)
	err = lt.Stop()
	require.Equal(t, ErrNotRunning, err)

	err = lt.Run()
	require.NoError(t, err)
	lt.status.State = Running

	numUsers := 8
	n, err := lt.AddUsers(numUsers)
	require.NoError(t, err)
	require.Equal(t, numUsers, n)

	err = lt.Stop()
	require.NoError(t, err)
	require.Equal(t, lt.status.State, Stopped)
	require.Empty(t, lt.activeControllers)
}

func TestStatus(t *testing.T) {
	log := logger.New(&ltConfig.LogSettings)
	lt, err := New(&ltConfig, newController, log)
	require.NotNil(t, lt)
	require.Nil(t, err)

	err = lt.Run()
	require.NoError(t, err)
	st := lt.Status()
	startTime := st.StartTime
	assert.Equal(t, Running, st.State)
	assert.Equal(t, int64(0), st.NumUsers)
	assert.Equal(t, int64(0), st.NumUsersAdded)
	assert.Equal(t, int64(0), st.NumUsersRemoved)

	n, err := lt.AddUsers(1)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	st = lt.Status()
	assert.Equal(t, Running, st.State)
	assert.Equal(t, int64(1), st.NumUsers)
	assert.Equal(t, int64(1), st.NumUsersAdded)
	assert.Equal(t, int64(0), st.NumUsersRemoved)

	n, err = lt.RemoveUsers(1)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	st = lt.Status()
	assert.Equal(t, Running, st.State)
	time.Sleep(1 * time.Second)
	assert.Equal(t, int64(0), st.NumUsers)
	assert.Equal(t, int64(1), st.NumUsersAdded)
	assert.Equal(t, int64(1), st.NumUsersRemoved)

	err = lt.Stop()
	require.NoError(t, err)
	st = lt.Status()
	assert.Equal(t, Stopped, st.State)
	assert.Equal(t, int64(0), st.NumUsers)
	assert.Equal(t, int64(1), st.NumUsersAdded)
	assert.Equal(t, int64(1), st.NumUsersRemoved)

	// Start again, and verify that start time got reset.
	err = lt.Run()
	require.NoError(t, err)
	st = lt.Status()
	assert.True(t, startTime.Before(st.StartTime))
	assert.Equal(t, Running, st.State)
}
