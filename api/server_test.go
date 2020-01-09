package api

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gavv/httpexpect"
)

func TestAPI(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter()

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	// create httpexpect instance
	e := httpexpect.New(t, server.URL)

	// is it working?
	e.GET("/loadtest/status/123").
		Expect().
		Status(http.StatusNotFound)

	sampleConfig, _ := ioutil.ReadFile("../config/config.default.json")
	obj := e.POST("/loadtest/create").WithHeader("Content-Type", "application/json").WithBytes(sampleConfig).
		Expect().
		Status(http.StatusOK).JSON().Object()
	ltId := obj.Value("loadTestId").String().Raw()

	e.POST("/loadtest/run/" + ltId).Expect().Status(http.StatusOK)
	e.PUT("/loadtest/user/"+ltId).WithQuery("amount", 10).Expect().Status(http.StatusOK)
	e.DELETE("/loadtest/user/"+ltId).WithQuery("amount", 3).Expect().Status(http.StatusOK)
	e.POST("/loadtest/stop/" + ltId).Expect().Status(http.StatusOK)
	e.POST("/loadtest/destroy/" + ltId).Expect().Status(http.StatusOK)
}
