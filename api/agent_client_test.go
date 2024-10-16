// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	client "github.com/mattermost/mattermost-load-test-ng/api/client/agent"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func createFakeMMServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Version-Id", "dev")
		switch r.URL.Path {
		case "/api/v4/users/login":
			u := model.User{}
			u.Username = "sysadmin"
			json.NewEncoder(w).Encode(u)
		case "/api/v4/config":
			mmCfg := model.Config{}
			mmCfg.SetDefaults()
			maxUsers := 10000
			mmCfg.TeamSettings.MaxUsersPerTeam = &maxUsers
			json.NewEncoder(w).Encode(mmCfg)
		case "/api/v4/emoji":
			json.NewEncoder(w).Encode(&model.Emoji{})
		default:
			fmt.Fprintln(w, "Hello, client")
		}
	}))
}

func createAgent(t *testing.T, id, serverURL string) *client.Agent {
	t.Helper()

	mmServer := createFakeMMServer()
	t.Cleanup(mmServer.Close)

	var ltConfig loadtest.Config
	var ucConfig simulcontroller.Config
	defaults.Set(&ltConfig)
	ltConfig.ConnectionConfiguration.ServerURL = mmServer.URL
	ltConfig.UserControllerConfiguration.ServerVersion = control.MinSupportedVersion.String()
	defaults.Set(&ucConfig)
	agent, err := client.New(id, serverURL, nil)
	require.NoError(t, err)
	require.NotNil(t, agent)
	_, err = agent.Create(&ltConfig, &ucConfig)
	require.NoError(t, err)
	return agent
}

func TestCreateAgent(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	mmServer := createFakeMMServer()
	defer mmServer.Close()

	id := "agent0"
	agent, err := client.New(id, server.URL, nil)
	require.NoError(t, err)
	require.NotNil(t, agent)

	t.Run("missing ltConfig", func(t *testing.T) {
		status, err := agent.Create(nil, nil)
		require.EqualError(t, err, "client: ltConfig should not be nil")
		require.Empty(t, status)
	})

	t.Run("missing uc type", func(t *testing.T) {
		status, err := agent.Create(&loadtest.Config{}, &simulcontroller.Config{})
		require.EqualError(t, err, "client: UserController type is not set")
		require.Empty(t, status)
	})

	t.Run("missing ucConfig, simulcontroller", func(t *testing.T) {
		var ltConfig loadtest.Config
		ltConfig.UserControllerConfiguration.Type = "simulative"
		status, err := agent.Create(&ltConfig, nil)
		require.EqualError(t, err, "client: ucConfig should not be nil")
		require.Empty(t, status)
	})

	t.Run("missing ucConfig, simplecontroller", func(t *testing.T) {
		var ltConfig loadtest.Config
		ltConfig.UserControllerConfiguration.Type = "simple"
		status, err := agent.Create(&ltConfig, nil)
		require.EqualError(t, err, "client: ucConfig should not be nil")
		require.Empty(t, status)
	})

	t.Run("missing ucConfig, noopcontroller, invalid configs", func(t *testing.T) {
		var ltConfig loadtest.Config
		ltConfig.UserControllerConfiguration.Type = "noop"
		status, err := agent.Create(&ltConfig, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "could not validate config")
		require.Empty(t, status)
	})

	t.Run("invalid ucConfig type", func(t *testing.T) {
		var ltConfig loadtest.Config
		ltConfig.UserControllerConfiguration.Type = "simulative"
		status, err := agent.Create(&ltConfig, "invalid")
		require.EqualError(t, err, "client: ucConfig has the wrong type")
		require.Empty(t, status)
	})

	t.Run("invalid configs", func(t *testing.T) {
		var ltConfig loadtest.Config
		ltConfig.UserControllerConfiguration.Type = "simulative"
		status, err := agent.Create(&ltConfig, &simulcontroller.Config{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "could not validate config")
		require.Empty(t, status)
	})

	t.Run("successful creation", func(t *testing.T) {
		var ltConfig loadtest.Config
		var ucConfig simulcontroller.Config
		defaults.Set(&ltConfig)
		ltConfig.ConnectionConfiguration.ServerURL = mmServer.URL
		defaults.Set(&ucConfig)
		_, err := agent.Create(&ltConfig, &ucConfig)
		require.NoError(t, err)
	})

	t.Run("conflict failure", func(t *testing.T) {
		var ltConfig loadtest.Config
		var ucConfig simulcontroller.Config
		defaults.Set(&ltConfig)
		defaults.Set(&ucConfig)
		_, err := agent.Create(&ltConfig, &ucConfig)
		require.Error(t, err)
		require.Contains(t, err.Error(), "load-test agent with id agent0 already exists")
	})
}

func TestAgentId(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	id := "agent0"
	agent := createAgent(t, id, server.URL)
	require.Equal(t, id, agent.Id())
}

func TestAgentStatus(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	id := "agent0"
	agent := createAgent(t, id, server.URL)

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

	id := "agent0"
	agent := createAgent(t, id, server.URL)

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

	id := "agent0"
	agent := createAgent(t, id, server.URL)

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

	id := "agent0"
	agent := createAgent(t, id, server.URL)

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
		agent := createAgent(t, id, server.URL)
		status, err = agent.Destroy()
		require.NoError(t, err)
		require.Empty(t, status)
		require.Equal(t, loadtest.Stopped, status.State)
	})
}

func TestAgentInjectAction(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	id := "agent0"
	agent := createAgent(t, id, server.URL)

	status, err := agent.Run()
	require.NoError(t, err)
	require.NotEmpty(t, status)
	require.Equal(t, loadtest.Running, status.State)
	defer agent.Stop()

	t.Run("inject reload", func(t *testing.T) {
		status, err := agent.InjectAction("Reload")
		require.NoError(t, err)
		require.NotEmpty(t, status)
	})

	t.Run("invalid action", func(t *testing.T) {
		status, err := agent.InjectAction("Bogus")
		// not all actions are supported by all controllers, therefore an unsupported
		// action is not an error.
		require.NoError(t, err)
		require.NotEmpty(t, status)
	})
}
