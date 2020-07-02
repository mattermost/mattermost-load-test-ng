// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/stretchr/testify/require"
)

func TestAgentId(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL, nil)
	require.NotNil(t, client)

	id := "agent0"
	agent := createAgent(t, client, id)
	require.Equal(t, id, agent.Id())
}

func TestAgentStatus(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL, nil)
	require.NotNil(t, client)

	id := "agent0"
	agent := createAgent(t, client, id)

	status, err := agent.Status()
	require.NoError(t, err)
	require.Empty(t, status)
	require.Equal(t, loadtest.Stopped, status.State)
}

func TestAgentRunStop(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL, nil)
	require.NotNil(t, client)

	id := "agent0"
	agent := createAgent(t, client, id)

	t.Run("stopping failure", func(t *testing.T) {
		status, err := agent.Status()
		require.NoError(t, err)
		require.Empty(t, status)
		require.Equal(t, loadtest.Stopped, status.State)

		status, err = agent.Stop()
		require.Error(t, err)
		require.Empty(t, status)
	})

	t.Run("successful run", func(t *testing.T) {
		status, err := agent.Status()
		require.NoError(t, err)
		require.Empty(t, status)
		require.Equal(t, loadtest.Stopped, status.State)

		now := time.Now()
		status, err = agent.Run()
		require.NoError(t, err)
		require.NotEmpty(t, status)
		require.Equal(t, loadtest.Running, status.State)
		require.True(t, time.Now().After(now))
	})

	t.Run("running twice", func(t *testing.T) {
		status, err := agent.Run()
		require.Error(t, err)
		require.Empty(t, status)

		status, err = agent.Status()
		require.NoError(t, err)
		require.NotEmpty(t, status)
		require.Equal(t, loadtest.Running, status.State)
	})

	t.Run("successful stop", func(t *testing.T) {
		status, err := agent.Status()
		require.NoError(t, err)
		require.NotEmpty(t, status)
		require.Equal(t, loadtest.Running, status.State)

		status, err = agent.Stop()
		require.NoError(t, err)
		require.NotEmpty(t, status)
		require.Equal(t, loadtest.Stopped, status.State)
	})
}

func TestAgentAddRemoveUsers(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL, nil)
	require.NotNil(t, client)

	id := "agent0"
	agent := createAgent(t, client, id)

	status, err := agent.Run()
	require.NoError(t, err)
	require.NotEmpty(t, status)
	require.Equal(t, loadtest.Running, status.State)
	defer agent.Stop()

	t.Run("invalid amount", func(t *testing.T) {
		status, err := agent.AddUsers(-10)
		require.Error(t, err)
		require.Empty(t, status)
		require.Contains(t, err.Error(), "invalid amount")

		status, err = agent.RemoveUsers(-10)
		require.Error(t, err)
		require.Empty(t, status)
		require.Contains(t, err.Error(), "invalid amount")
	})

	t.Run("no users left", func(t *testing.T) {
		status, err = agent.RemoveUsers(10)
		require.Error(t, err)
		require.Empty(t, status)
		require.Contains(t, err.Error(), "no active users left")
	})

	t.Run("add success", func(t *testing.T) {
		status, err = agent.AddUsers(10)
		require.NoError(t, err)
		require.NotEmpty(t, status)
		require.Equal(t, int64(10), status.NumUsers)
		require.Equal(t, int64(10), status.NumUsersAdded)
		require.Equal(t, int64(0), status.NumUsersStopped)
		require.Equal(t, int64(0), status.NumUsersRemoved)
	})

	t.Run("remove success", func(t *testing.T) {
		status, err = agent.RemoveUsers(10)
		require.NoError(t, err)
		require.NotEmpty(t, status)
		require.Equal(t, int64(0), status.NumUsers)
		require.Equal(t, int64(10), status.NumUsersAdded)
		require.Equal(t, int64(10), status.NumUsersRemoved)
	})
}

func TestAgentDestroy(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL, nil)
	require.NotNil(t, client)

	id := "agent0"
	agent := createAgent(t, client, id)

	status, err := agent.Run()
	require.NoError(t, err)
	require.NotEmpty(t, status)
	require.Equal(t, loadtest.Running, status.State)

	t.Run("destroy success", func(t *testing.T) {
		status, err := agent.Destroy()
		require.NoError(t, err)
		require.NotEmpty(t, status)
		require.Equal(t, loadtest.Stopped, status.State)

		id := "agent0"
		agent := createAgent(t, client, id)
		status, err = agent.Destroy()
		require.NoError(t, err)
		require.Empty(t, status)
		require.Equal(t, loadtest.Stopped, status.State)
	})
}
