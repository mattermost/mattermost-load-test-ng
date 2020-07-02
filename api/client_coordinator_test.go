// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/stretchr/testify/require"
)

func TestCoordinatorId(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL, nil)
	require.NotNil(t, client)

	id := "coord0"
	coord := createCoordinator(t, client, id)
	require.Equal(t, id, coord.Id())
}

func TestCoordinatorStatus(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL, nil)
	require.NotNil(t, client)

	id := "coord0"
	coord := createCoordinator(t, client, id)

	status, err := coord.Status()
	require.NoError(t, err)
	require.Empty(t, status)
	require.Equal(t, coordinator.Stopped, status.State)
}

func TestCoordinatorStartStop(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL, nil)
	require.NotNil(t, client)

	id := "coord0"
	coord := createCoordinator(t, client, id)

	t.Run("stopping failure", func(t *testing.T) {
		status, err := coord.Status()
		require.NoError(t, err)
		require.Empty(t, status)
		require.Equal(t, coordinator.Stopped, status.State)

		status, err = coord.Stop()
		require.Error(t, err)
		require.Empty(t, status)
	})

	t.Run("successful run", func(t *testing.T) {
		status, err := coord.Status()
		require.NoError(t, err)
		require.Empty(t, status)
		require.Equal(t, coordinator.Stopped, status.State)

		now := time.Now()
		status, err = coord.Run()
		require.NoError(t, err)
		require.NotEmpty(t, status)
		require.Equal(t, coordinator.Running, status.State)
		require.True(t, time.Now().After(now))
	})

	t.Run("running twice", func(t *testing.T) {
		status, err := coord.Run()
		require.Error(t, err)
		require.Empty(t, status)

		status, err = coord.Status()
		require.NoError(t, err)
		require.NotEmpty(t, status)
		require.Equal(t, coordinator.Running, status.State)
	})

	t.Run("successful stop", func(t *testing.T) {
		status, err := coord.Status()
		require.NoError(t, err)
		require.NotEmpty(t, status)
		require.Equal(t, coordinator.Running, status.State)

		status, err = coord.Stop()
		require.NoError(t, err)
		require.NotEmpty(t, status)
		require.Equal(t, coordinator.Done, status.State)
	})

	t.Run("re-run failure", func(t *testing.T) {
		status, err := coord.Run()
		require.Error(t, err)
		require.Empty(t, status)

		status, err = coord.Status()
		require.NoError(t, err)
		require.NotEmpty(t, status)
		require.Equal(t, coordinator.Done, status.State)
	})
}

func TestCoordinatorDestroy(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL, nil)
	require.NotNil(t, client)

	id := "coord0"
	coord := createCoordinator(t, client, id)

	status, err := coord.Run()
	require.NoError(t, err)
	require.NotEmpty(t, status)
	require.Equal(t, coordinator.Running, status.State)

	t.Run("destroy success", func(t *testing.T) {
		status, err := coord.Destroy()
		require.NoError(t, err)
		require.NotEmpty(t, status)
		require.Equal(t, coordinator.Done, status.State)

		id := "coord0"
		coord := createCoordinator(t, client, id)
		status, err = coord.Destroy()
		require.NoError(t, err)
		require.Empty(t, status)
		require.Equal(t, coordinator.Stopped, status.State)
	})
}
