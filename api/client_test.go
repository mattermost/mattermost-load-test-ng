// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/stretchr/testify/require"
)

func createAgent(t *testing.T, client *Client, id string) *Agent {
	t.Helper()
	var ltConfig loadtest.Config
	var ucConfig simulcontroller.Config
	defaults.Set(&ltConfig)
	defaults.Set(&ucConfig)
	agent, err := client.CreateAgent(id, &ltConfig, &ucConfig)
	require.NoError(t, err)
	require.NotNil(t, agent)
	return agent
}

func createCoordinator(t *testing.T, client *Client, id string) *Coordinator {
	t.Helper()
	var coordConfig coordinator.Config
	var ltConfig loadtest.Config
	defaults.Set(&coordConfig)
	defaults.Set(&ltConfig)
	coordConfig.ClusterConfig.Agents[0].ApiURL = client.serverURL
	coordConfig.MonitorConfig.Queries[0].Description = "Query"
	coordConfig.MonitorConfig.Queries[0].Query = "query"
	coord, err := client.CreateCoordinator(id, &coordConfig, &ltConfig)
	require.NoError(t, err)
	require.NotNil(t, coord)
	return coord
}

func TestNewClient(t *testing.T) {
	serverURL := "http://apiserver"
	client := NewClient(serverURL, nil)
	require.NotNil(t, client)
	require.Equal(t, http.DefaultClient, client.httpClient)
	require.Equal(t, serverURL, client.serverURL)

	httpClient := &http.Client{}
	client = NewClient(serverURL, httpClient)
	require.NotNil(t, client)
	require.Equal(t, httpClient, client.httpClient)
	require.Equal(t, serverURL, client.serverURL)
}

func TestCreateAgent(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL, nil)
	require.NotNil(t, client)

	t.Run("missing ltConfig", func(t *testing.T) {
		agent, err := client.CreateAgent("someid", nil, nil)
		require.EqualError(t, err, "agent: ltConfig should not be nil")
		require.Nil(t, agent)
	})

	t.Run("missing ucConfig", func(t *testing.T) {
		agent, err := client.CreateAgent("someid", &loadtest.Config{}, nil)
		require.EqualError(t, err, "agent: ucConfig should not be nil")
		require.Nil(t, agent)
	})

	t.Run("missing uc type", func(t *testing.T) {
		agent, err := client.CreateAgent("someid", &loadtest.Config{}, &simulcontroller.Config{})
		require.EqualError(t, err, "agent: UserController type is not set")
		require.Nil(t, agent)
	})

	t.Run("invalid ucConfig type", func(t *testing.T) {
		var ltConfig loadtest.Config
		ltConfig.UserControllerConfiguration.Type = "simulative"
		agent, err := client.CreateAgent("someid", &ltConfig, "invalid")
		require.EqualError(t, err, "agent: ucConfig has the wrong type")
		require.Nil(t, agent)
	})

	t.Run("invalid configs", func(t *testing.T) {
		var ltConfig loadtest.Config
		ltConfig.UserControllerConfiguration.Type = "simulative"
		agent, err := client.CreateAgent("agent0", &ltConfig, &simulcontroller.Config{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "could not validate config")
		require.Nil(t, agent)
	})

	t.Run("successful creation", func(t *testing.T) {
		var ltConfig loadtest.Config
		var ucConfig simulcontroller.Config
		defaults.Set(&ltConfig)
		defaults.Set(&ucConfig)
		agent, err := client.CreateAgent("agent0", &ltConfig, &ucConfig)
		require.NoError(t, err)
		require.NotNil(t, agent)
	})

	t.Run("conflict failure", func(t *testing.T) {
		var ltConfig loadtest.Config
		var ucConfig simulcontroller.Config
		defaults.Set(&ltConfig)
		defaults.Set(&ucConfig)
		agent, err := client.CreateAgent("agent0", &ltConfig, &ucConfig)
		require.Error(t, err)
		require.Nil(t, agent)
		require.Contains(t, err.Error(), "load-test agent with id agent0 already exists")
	})
}

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

func TestCreateCoordinator(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient(server.URL, nil)
	require.NotNil(t, client)

	t.Run("missing coordConfig", func(t *testing.T) {
		coord, err := client.CreateCoordinator("someid", nil, nil)
		require.EqualError(t, err, "coordinator: coordConfig should not be nil")
		require.Nil(t, coord)
	})

	t.Run("missing ltConfig", func(t *testing.T) {
		coord, err := client.CreateCoordinator("someid", &coordinator.Config{}, nil)
		require.EqualError(t, err, "coordinator: ltConfig should not be nil")
		require.Nil(t, coord)
	})

	t.Run("successful creation", func(t *testing.T) {
		var coordConfig coordinator.Config
		var ltConfig loadtest.Config
		defaults.Set(&coordConfig)
		defaults.Set(&ltConfig)
		coordConfig.MonitorConfig.Queries[0].Description = "Query"
		coordConfig.MonitorConfig.Queries[0].Query = "query"
		coord, err := client.CreateCoordinator("coord0", &coordConfig, &ltConfig)
		require.NoError(t, err)
		require.NotNil(t, coord)
	})

	t.Run("conflict failure", func(t *testing.T) {
		var coordConfig coordinator.Config
		var ltConfig loadtest.Config
		defaults.Set(&coordConfig)
		defaults.Set(&ltConfig)
		coordConfig.MonitorConfig.Queries[0].Description = "Query"
		coordConfig.MonitorConfig.Queries[0].Query = "query"
		coord, err := client.CreateCoordinator("coord0", &coordConfig, &ltConfig)
		require.Error(t, err)
		require.Nil(t, coord)
		require.Contains(t, err.Error(), "load-test coordinator with id coord0 already exists")
	})
}

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
