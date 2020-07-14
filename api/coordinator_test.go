// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/gavv/httpexpect"
	"github.com/stretchr/testify/require"
)

func TestCoordinatorAPI(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	// create httpexpect instance
	e := httpexpect.New(t, server.URL+"/coordinator")

	var ltConfig loadtest.Config
	err := defaults.Set(&ltConfig)
	require.NoError(t, err)
	ltConfig.ConnectionConfiguration.ServerURL = "http://fakesitetotallydoesntexist.com"
	ltConfig.UsersConfiguration.MaxActiveUsers = 100

	var config coordinator.Config
	err = defaults.Set(&config)
	require.NoError(t, err)
	config.MonitorConfig.Queries[0].Description = "Query"
	config.MonitorConfig.Queries[0].Query = "query"
	config.ClusterConfig.Agents[0].ApiURL = server.URL

	t.Run("create/destroy", func(t *testing.T) {
		data := struct {
			CoordinatorConfig coordinator.Config
			LoadTestConfig    loadtest.Config
		}{}

		id := "ltc0"
		obj := e.POST("/create").WithQuery("id", id).WithJSON(data).
			Expect().Status(http.StatusBadRequest).
			JSON().Object()
		rawMsg := obj.Value("error").String().Raw()
		require.Contains(t, rawMsg, "could not validate load-test config")

		data.LoadTestConfig = ltConfig
		data.CoordinatorConfig = config

		obj = e.POST("/create").WithQuery("id", id).WithJSON(data).
			Expect().Status(http.StatusCreated).
			JSON().Object().ContainsKey("message")
		rawMsg = obj.Value("message").String().Raw()
		require.Equal(t, "load-test coordinator created", rawMsg)

		obj = e.POST("/create").WithQuery("id", id).WithJSON(data).
			Expect().Status(http.StatusConflict).
			JSON().Object().ContainsKey("error")
		rawMsg = obj.Value("error").String().Raw()
		require.Equal(t, "load-test coordinator with id ltc0 already exists", rawMsg)

		eAgent := httpexpect.New(t, server.URL+"/loadagent")
		obj = eAgent.GET(id).Expect().Status(http.StatusBadRequest).
			JSON().Object().ContainsKey("error")
		rawMsg = obj.Value("error").String().Raw()
		require.Equal(t, "resource with id ltc0 is not a load-test agent", rawMsg)

		obj = e.DELETE(id).
			Expect().Status(http.StatusOK).
			JSON().Object().ContainsKey("message")
		rawMsg = obj.Value("message").String().Raw()
		require.Equal(t, "load-test coordinator destroyed", rawMsg)

		e.DELETE(id).Expect().Status(http.StatusNotFound)
	})

	t.Run("run/stop", func(t *testing.T) {
		id := "ltc0"
		e.GET(id + "/status").Expect().Status(http.StatusNotFound)

		data := struct {
			CoordinatorConfig coordinator.Config
			LoadTestConfig    loadtest.Config
		}{
			config,
			ltConfig,
		}

		data.CoordinatorConfig.ClusterConfig.Agents[0].ApiURL = server.URL

		obj := e.POST("/create").WithQuery("id", id).WithJSON(data).
			Expect().Status(http.StatusCreated).
			JSON().Object().ContainsKey("message")
		rawMsg := obj.Value("message").String().Raw()
		require.Equal(t, "load-test coordinator created", rawMsg)

		e.POST(id + "/stop").Expect().Status(http.StatusBadRequest)

		eAgent := httpexpect.New(t, server.URL+"/loadagent")
		eAgent.GET("/" + config.ClusterConfig.Agents[0].Id).Expect().Status(http.StatusOK)
		e.POST(id + "/run").Expect().Status(http.StatusOK)
		e.POST(id + "/run").Expect().Status(http.StatusBadRequest)
		e.POST(id + "/stop").Expect().Status(http.StatusOK)
		eAgent.GET("/" + config.ClusterConfig.Agents[0].Id).Expect().Status(http.StatusNotFound)

		e.DELETE(id).Expect().Status(http.StatusOK)
	})

	t.Run("status", func(t *testing.T) {
		id := "ltc0"
		e.GET(id + "/status").Expect().Status(http.StatusNotFound)

		data := struct {
			CoordinatorConfig coordinator.Config
			LoadTestConfig    loadtest.Config
		}{
			config,
			ltConfig,
		}

		obj := e.POST("/create").WithQuery("id", id).WithJSON(data).
			Expect().Status(http.StatusCreated).
			JSON().Object().ContainsKey("message")
		rawMsg := obj.Value("message").String().Raw()
		require.Equal(t, "load-test coordinator created", rawMsg)

		e.GET(id + "/status").Expect().Status(http.StatusOK).
			JSON().Object().ContainsKey("status")

		e.DELETE(id).Expect().Status(http.StatusOK)
	})
}
