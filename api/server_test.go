// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"

	"github.com/gavv/httpexpect"
	"github.com/stretchr/testify/require"
)

type requestData struct {
	LoadTestConfig         loadtest.Config
	SimpleControllerConfig *simplecontroller.Config `json:",omitempty"`
	SimulControllerConfig  *simulcontroller.Config  `json:",omitempty"`
}

func TestAPI(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter()

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	// create httpexpect instance
	e := httpexpect.New(t, server.URL+"/loadagent")

	// is it working?
	e.GET("/123/status").
		Expect().
		Status(http.StatusNotFound)

	ltConfig := loadtest.Config{}
	err := defaults.Set(&ltConfig)
	require.NoError(t, err)

	ltConfig.ConnectionConfiguration.ServerURL = "http://fakesitetotallydoesntexist.com"
	ltConfig.UsersConfiguration.MaxActiveUsers = 100

	ucConfig1, err := simplecontroller.ReadConfig("../config/simplecontroller.sample.json")
	require.NoError(t, err)

	ucConfig2, err := simulcontroller.ReadConfig("../config/simulcontroller.sample.json")
	require.NoError(t, err)

	t.Run("test with loadtest.Config only", func(t *testing.T) {
		rd := requestData{
			LoadTestConfig: ltConfig,
		}
		ltId := "lt0"
		obj := e.POST("/create").WithQuery("id", ltId).WithJSON(rd).
			Expect().Status(http.StatusCreated).
			JSON().Object().ValueEqual("id", ltId)
		rawMsg := obj.Value("message").String().Raw()
		require.Equal(t, rawMsg, "load-test agent created")

		e.GET(ltId + "/status").
			Expect().
			Status(http.StatusOK).
			JSON().Object().NotContainsKey("error")

		e.GET(ltId).
			Expect().
			Status(http.StatusOK).
			JSON().Object().NotContainsKey("error")

		obj = e.POST("/create").WithQuery("id", ltId).WithJSON(rd).
			Expect().Status(http.StatusBadRequest).
			JSON().Object().ContainsKey("error")
		rawMsg = obj.Value("error").String().Raw()
		require.Equal(t, fmt.Sprintf("load-test agent with id %s already exists", ltId), rawMsg)

		e.POST(ltId + "/run").Expect().Status(http.StatusOK)
		e.POST(ltId+"/addusers").WithQuery("amount", 10).Expect().Status(http.StatusOK)
		e.POST(ltId+"/removeusers").WithQuery("amount", 3).Expect().Status(http.StatusOK)
		e.POST(ltId+"/addusers").WithQuery("amount", 0).Expect().
			Status(http.StatusBadRequest).
			JSON().Object().ContainsKey("error")

		e.POST(ltId+"/addusers").WithQuery("amount", -2).Expect().
			Status(http.StatusBadRequest).
			JSON().Object().ContainsKey("error")

		e.POST(ltId+"/addusers").WithQuery("amount", "bad").Expect().
			Status(http.StatusBadRequest).
			JSON().Object().ContainsKey("error")

		e.POST(ltId+"/removeusers").WithQuery("amount", 0).Expect().
			Status(http.StatusBadRequest).
			JSON().Object().ContainsKey("error")

		e.POST(ltId+"/removeusers").WithQuery("amount", -2).Expect().
			Status(http.StatusBadRequest).
			JSON().Object().ContainsKey("error")

		e.POST(ltId+"/removeusers").WithQuery("amount", "bad").Expect().
			Status(http.StatusBadRequest).
			JSON().Object().ContainsKey("error")

		e.POST(ltId + "/stop").Expect().Status(http.StatusOK)
		e.DELETE(ltId).Expect().Status(http.StatusOK)
	})

	t.Run("start agent with a simplecontroller.Config", func(t *testing.T) {
		ltConfig.UserControllerConfiguration.Type = loadtest.UserControllerSimple
		rd := requestData{
			LoadTestConfig:         ltConfig,
			SimpleControllerConfig: ucConfig1,
		}
		ltId := "lt0"
		obj := e.POST("/create").WithQuery("id", ltId).WithJSON(rd).
			Expect().Status(http.StatusCreated).
			JSON().Object().ValueEqual("id", ltId)
		rawMsg := obj.Value("message").String().Raw()
		require.Equal(t, rawMsg, "load-test agent created")
		e.POST(ltId + "/run").Expect().Status(http.StatusOK)
		e.POST(ltId + "/stop").Expect().Status(http.StatusOK)
		e.DELETE(ltId).Expect().Status(http.StatusOK)
	})

	t.Run("start agent with simulcontroller.Config", func(t *testing.T) {
		ltConfig.UserControllerConfiguration.Type = loadtest.UserControllerSimulative
		rd := requestData{
			LoadTestConfig:        ltConfig,
			SimulControllerConfig: ucConfig2,
		}
		ltId := "lt0"
		obj := e.POST("/create").WithQuery("id", ltId).WithJSON(rd).
			Expect().Status(http.StatusCreated).
			JSON().Object().ValueEqual("id", ltId)
		rawMsg := obj.Value("message").String().Raw()
		require.Equal(t, rawMsg, "load-test agent created")
		e.POST(ltId + "/run").Expect().Status(http.StatusOK)
		e.POST(ltId + "/stop").Expect().Status(http.StatusOK)
		e.DELETE(ltId).Expect().Status(http.StatusOK)
	})

	t.Run("start agent with no controller config", func(t *testing.T) {
		ltConfig.UserControllerConfiguration.Type = loadtest.UserControllerSimulative
		rd := requestData{
			LoadTestConfig: ltConfig,
		}
		ltId := "lt0"
		obj := e.POST("/create").WithQuery("id", ltId).WithJSON(rd).
			Expect().Status(http.StatusCreated).
			JSON().Object().ValueEqual("id", ltId)
		rawMsg := obj.Value("message").String().Raw()
		require.Equal(t, rawMsg, "load-test agent created")
		e.POST(ltId + "/run").Expect().Status(http.StatusOK)
		e.POST(ltId + "/stop").Expect().Status(http.StatusOK)
		e.DELETE(ltId).Expect().Status(http.StatusOK)
	})
}
