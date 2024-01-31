// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"net/http/httptest"
	"testing"
	"time"

	client "github.com/mattermost/mattermost-load-test-ng/api/client/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/stretchr/testify/require"
)

func createCoordinator(t *testing.T, id, serverURL string) *client.Coordinator {
	t.Helper()
	mmServer := createFakeMMServer()
	t.Cleanup(mmServer.Close)
	var coordConfig coordinator.Config
	var ltConfig loadtest.Config
	defaults.Set(&coordConfig)
	defaults.Set(&ltConfig)
	ltConfig.ConnectionConfiguration.ServerURL = mmServer.URL
	coordConfig.ClusterConfig.Agents[0].ApiURL = serverURL
	coord, err := client.New(id, serverURL, nil)
	require.NoError(t, err)
	require.NotNil(t, coord)
	_, err = coord.Create(&coordConfig, &ltConfig)
	require.NoError(t, err)
	return coord
}

func TestCreateCoordinator(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	mmServer := createFakeMMServer()
	defer mmServer.Close()

	id := "coord0"
	coord, err := client.New(id, server.URL, nil)
	require.NoError(t, err)
	require.NotNil(t, coord)

	t.Run("missing coordConfig", func(t *testing.T) {
		status, err := coord.Create(nil, nil)
		require.Error(t, err)
		require.EqualError(t, err, "client: coordConfig should not be nil")
		require.Empty(t, status)
	})

	t.Run("missing ltConfig", func(t *testing.T) {
		status, err := coord.Create(&coordinator.Config{}, nil)
		require.Error(t, err)
		require.EqualError(t, err, "client: ltConfig should not be nil")
		require.Empty(t, status)
	})

	t.Run("successful creation", func(t *testing.T) {
		var coordConfig coordinator.Config
		var ltConfig loadtest.Config
		defaults.Set(&coordConfig)
		defaults.Set(&ltConfig)
		ltConfig.ConnectionConfiguration.ServerURL = mmServer.URL
		coordConfig.ClusterConfig.Agents[0].ApiURL = server.URL
		_, err := coord.Create(&coordConfig, &ltConfig)
		require.NoError(t, err)
	})

	t.Run("conflict failure", func(t *testing.T) {
		var coordConfig coordinator.Config
		var ltConfig loadtest.Config
		defaults.Set(&coordConfig)
		defaults.Set(&ltConfig)
		ltConfig.ConnectionConfiguration.ServerURL = mmServer.URL
		coordConfig.ClusterConfig.Agents[0].ApiURL = server.URL
		status, err := coord.Create(&coordConfig, &ltConfig)
		require.Error(t, err)
		require.Contains(t, err.Error(), "load-test coordinator with id coord0 already exists")
		require.Empty(t, status)
	})
}

func TestCoordinatorId(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	id := "coord0"
	coord := createCoordinator(t, id, server.URL)
	require.Equal(t, id, coord.Id())
}

func TestCoordinatorStatus(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	id := "coord0"
	coord := createCoordinator(t, id, server.URL)

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

	id := "coord0"
	coord := createCoordinator(t, id, server.URL)

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

	id := "coord0"
	coord := createCoordinator(t, id, server.URL)

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
		coord := createCoordinator(t, id, server.URL)
		status, err = coord.Destroy()
		require.NoError(t, err)
		require.Equal(t, coordinator.Stopped, status.State)
	})
}

func TestCoordinatorInjectAction(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	id := "coord0"
	coord := createCoordinator(t, id, server.URL)

	status, err := coord.Run()
	require.NoError(t, err)
	require.NotEmpty(t, status)
	require.Equal(t, coordinator.Running, status.State)

	t.Run("Inject reload", func(t *testing.T) {
		status, err := coord.InjectAction("Reload")
		require.NoError(t, err)
		require.NotEmpty(t, status)
	})

	t.Run("Inject invalid action", func(t *testing.T) {
		status, err := coord.InjectAction("Bogus")
		// not all actions are supported by all controllers, therefore an unsupported
		// action is not an error.
		require.NoError(t, err)
		require.NotEmpty(t, status)
	})
}
