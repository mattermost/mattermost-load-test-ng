// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/gavv/httpexpect"
	"github.com/stretchr/testify/require"
)

type requestData struct {
	LoadTestConfig         loadtest.Config
	SimpleControllerConfig *simplecontroller.Config `json:",omitempty"`
	SimulControllerConfig  *simulcontroller.Config  `json:",omitempty"`
}

func TestAgentAPI(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	mmServer := createFakeMMServer()
	defer mmServer.Close()

	// create httpexpect instance
	e := httpexpect.New(t, server.URL+"/loadagent")

	// is it working?
	e.GET("/123/status").
		Expect().
		Status(http.StatusNotFound)

	ltConfig := loadtest.Config{}
	err := defaults.Set(&ltConfig)
	require.NoError(t, err)

	ltConfig.UserControllerConfiguration.ServerVersion = control.MinSupportedVersion.String()
	ltConfig.ConnectionConfiguration.ServerURL = mmServer.URL
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
			Expect().Status(http.StatusConflict).
			JSON().Object().ContainsKey("error")
		rawMsg = obj.Value("error").String().Raw()
		require.Equal(t, fmt.Sprintf("load-test agent with id %s already exists", ltId), rawMsg)

		eCoord := httpexpect.New(t, server.URL+"/coordinator")
		obj = eCoord.GET(ltId).Expect().Status(http.StatusBadRequest).
			JSON().Object().ContainsKey("error")
		rawMsg = obj.Value("error").String().Raw()
		require.Equal(t, fmt.Sprintf("resource with id %s is not a load-test coordinator", ltId), rawMsg)

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

func TestAgentAPIConcurrency(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	mmServer := createFakeMMServer()
	defer mmServer.Close()

	// create httpexpect instance
	e := httpexpect.New(t, server.URL+"/loadagent")

	ltConfig := loadtest.Config{}
	err := defaults.Set(&ltConfig)
	require.NoError(t, err)

	ltConfig.UserControllerConfiguration.ServerVersion = control.MinSupportedVersion.String()
	ltConfig.ConnectionConfiguration.ServerURL = mmServer.URL
	ltConfig.UsersConfiguration.MaxActiveUsers = 100

	ucConfig, err := simulcontroller.ReadConfig("../config/simulcontroller.sample.json")
	require.NoError(t, err)

	n := 4
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(id int) {
			defer wg.Done()
			rd := requestData{
				LoadTestConfig:        ltConfig,
				SimulControllerConfig: ucConfig,
			}
			ltId := fmt.Sprintf("lt%d", id)
			obj := e.POST("/create").WithQuery("id", ltId).WithJSON(rd).
				Expect().Status(http.StatusCreated).
				JSON().Object().ValueEqual("id", ltId)
			rawMsg := obj.Value("message").String().Raw()
			require.Equal(t, rawMsg, "load-test agent created")

			e.GET(ltId + "/status").
				Expect().
				Status(http.StatusOK).
				JSON().Object().NotContainsKey("error")
		}(i)
	}

	wg.Wait()
}

func TestGetUserCredentials(t *testing.T) {
	for _, tc := range []struct {
		name          string
		fileContents  []string
		expectedCreds []user
		expectErr     bool
	}{
		{
			name: "get simple credentials",
			expectedCreds: []user{{
				email:       "email1@sample.mattermost.com",
				password:    "password",
				username:    "email1",
				authService: userentity.AuthenticationTypeMattermost,
			}, {
				email:       "email2@sample.mattermost.com",
				password:    "password",
				username:    "email2",
				authService: userentity.AuthenticationTypeMattermost,
			}},
			fileContents: []string{
				"email1@sample.mattermost.com password",
				"email2@sample.mattermost.com password",
			},
		}, {
			name: "get credentials with custom auth provider",
			expectedCreds: []user{{
				email:       "email1@sample.mattermost.com",
				password:    "password",
				username:    "email1",
				authService: userentity.AuthenticationTypeOpenID,
			}},
			fileContents: []string{
				"openid:email1@sample.mattermost.com password",
			},
		}, {
			name:      "incorrect auth provider",
			expectErr: true,
			fileContents: []string{
				"incorrect:email@sample.mattermost.com password",
			},
		}, {
			name:      "incorrect number of fields",
			expectErr: true,
			fileContents: []string{
				"bogus",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "credentials")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.Write([]byte(strings.Join(tc.fileContents, "\n")))
			require.NoError(t, err)

			creds, err := getUserCredentials(tmpFile.Name(), nil)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedCreds, creds)
			}
		})
	}
}

func TestIsBrowserAgentInstance(t *testing.T) {
	// Get original home directory to restore later
	originalHome := os.Getenv("HOME")

	t.Run("returns true when agent_type.txt contains browser_agent", func(t *testing.T) {
		// Create temporary directory to use as home
		tempDir, err := os.MkdirTemp("", "test_home_browser")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Set temporary home directory
		os.Setenv("HOME", tempDir)
		defer os.Setenv("HOME", originalHome)

		agentTypeFile := filepath.Join(tempDir, deployment.AgentTypeFileName)
		err = os.WriteFile(agentTypeFile, []byte("  "+deployment.AgentTypeBrowser+"  \n"), 0644)
		require.NoError(t, err)

		result, err := isBrowserAgentInstance()
		require.NoError(t, err)
		require.True(t, result)
	})

	t.Run("returns false when agent_type.txt contains server_agent", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test_home_server")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		os.Setenv("HOME", tempDir)
		defer os.Setenv("HOME", originalHome)

		agentTypeFile := filepath.Join(tempDir, deployment.AgentTypeFileName)
		err = os.WriteFile(agentTypeFile, []byte(deployment.AgentTypeServer), 0644)
		require.NoError(t, err)

		result, err := isBrowserAgentInstance()
		require.NoError(t, err)
		require.False(t, result)
	})

	t.Run("returns false when agent_type.txt file does not exist", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test_home_missing")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		os.Setenv("HOME", tempDir)
		defer os.Setenv("HOME", originalHome)

		result, err := isBrowserAgentInstance()
		require.Error(t, err)
		require.False(t, result)
	})

	t.Run("returns false when agent_type.txt contains unknown content", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test_home_unknown")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		os.Setenv("HOME", tempDir)
		defer os.Setenv("HOME", originalHome)

		agentTypeFile := filepath.Join(tempDir, deployment.AgentTypeFileName)
		err = os.WriteFile(agentTypeFile, []byte("unknown_agent_type"), 0644)
		require.NoError(t, err)

		result, err := isBrowserAgentInstance()
		require.Error(t, err)
		require.False(t, result)
	})
}
