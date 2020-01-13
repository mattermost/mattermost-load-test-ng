package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/config"

	"github.com/gavv/httpexpect"
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

	sampleConfigBytes, _ := ioutil.ReadFile("../config/config.default.json")
	var sampleConfig config.LoadTestConfig
	_ = json.Unmarshal(sampleConfigBytes, &sampleConfig)
	sampleConfig.ConnectionConfiguration.ServerURL = "http://fakesitetotallydoesntexist.com"
	sampleConfig.UsersConfiguration.MaxActiveUsers = 100
	obj := e.POST("/create").WithJSON(sampleConfig).
		Expect().
		Status(http.StatusOK).JSON().Object()
	ltId := obj.Value("loadAgentId").String().Raw()

	e.POST(ltId + "/run").Expect().Status(http.StatusOK)
	e.POST(ltId+"/user/add").WithQuery("amount", 10).Expect().Status(http.StatusOK)
	e.POST(ltId+"/user/remove").WithQuery("amount", 3).Expect().Status(http.StatusOK)
	e.POST(ltId+"/user/add").WithQuery("amount", 0).Expect().
		Status(http.StatusBadRequest).
		JSON().Object().ContainsKey("error")

	e.POST(ltId+"/user/add").WithQuery("amount", -2).Expect().
		Status(http.StatusBadRequest).
		JSON().Object().ContainsKey("error")

	e.POST(ltId+"/user/add").WithQuery("amount", "bad").Expect().
		Status(http.StatusBadRequest).
		JSON().Object().ContainsKey("error")

	e.POST(ltId+"/user/remove").WithQuery("amount", 0).Expect().
		Status(http.StatusBadRequest).
		JSON().Object().ContainsKey("error")

	e.POST(ltId+"/user/remove").WithQuery("amount", -2).Expect().
		Status(http.StatusBadRequest).
		JSON().Object().ContainsKey("error")

	e.POST(ltId+"/user/remove").WithQuery("amount", "bad").Expect().
		Status(http.StatusBadRequest).
		JSON().Object().ContainsKey("error")

	e.POST(ltId + "/stop").Expect().Status(http.StatusOK)
	e.DELETE(ltId).Expect().Status(http.StatusOK)
}
