// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"

	"github.com/gavv/httpexpect"
	"github.com/stretchr/testify/require"
)

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

	ltConfig, err := loadtest.ReadConfig("../config/config.default.json")
	require.NoError(t, err)
	ucConfig, err := simplecontroller.ReadConfig("../config/simplecontroller.default.json")
	require.NoError(t, err)

	ltConfig.ConnectionConfiguration.ServerURL = "http://fakesitetotallydoesntexist.com"
	ltConfig.UsersConfiguration.MaxActiveUsers = 100

	var data = struct {
		LoadTestConfig   loadtest.Config
		ControllerConfig control.Config
	}{
		LoadTestConfig:   *ltConfig,
		ControllerConfig: ucConfig,
	}
	ltId := "lt0"
	obj := e.POST("/create").WithQuery("id", ltId).WithJSON(data).
		Expect().Status(http.StatusCreated).
		JSON().Object().ValueEqual("id", ltId)
	rawMsg := obj.Value("message").String().Raw()
	require.Equal(t, rawMsg, "load-test agent created")

	obj = e.POST("/create").WithQuery("id", ltId).WithJSON(data).
		Expect().Status(http.StatusBadRequest).
		JSON().Object().ContainsKey("error")
	rawMsg = obj.Value("error").String().Raw()
	require.Equal(t, fmt.Sprintf("load-test agent with id %s already exists", ltId), rawMsg)

	e.GET(ltId + "/status").
		Expect().
		Status(http.StatusOK).
		JSON().Object().NotContainsKey("error")

	e.GET(ltId).
		Expect().
		Status(http.StatusOK).
		JSON().Object().NotContainsKey("error")

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
}
