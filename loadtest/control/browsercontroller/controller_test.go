// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package browsercontroller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	store, err := memstore.New(nil)
	require.NoError(t, err)
	user := userentity.New(userentity.Setup{Store: store}, userentity.Config{
		ServerURL:    "http://localhost:8065",
		WebSocketURL: "ws://localhost:8065",
		Username:     "testuser",
		Email:        "test@example.com",
		Password:     "testpass",
	})

	statusChan := make(chan control.UserStatus, 10)
	defer close(statusChan)

	t.Run("controller is created successfully", func(t *testing.T) {
		controller, err := New(1, user, "http://localhost:8065", statusChan)
		require.NoError(t, err)
		require.NotNil(t, controller)
		require.Equal(t, 1, controller.id)
		require.Equal(t, user, controller.user)
		require.Equal(t, LTBrowserApi, controller.ltBrowserApiUrl)
		require.NotNil(t, controller.httpClient)
	})

	t.Run("controller is not created if user is nil", func(t *testing.T) {
		controller, err := New(1, nil, "http://localhost:8065", statusChan)
		require.Error(t, err)
		require.Nil(t, controller)
		require.Contains(t, err.Error(), "user cannot be nil")
	})

	t.Run("controller is not created if status channel is nil", func(t *testing.T) {
		controller, err := New(1, user, "http://localhost:8065", nil)
		require.Error(t, err)
		require.Nil(t, controller)
		require.Contains(t, err.Error(), "status channel cannot be nil")
	})
}

func newController(t *testing.T) (*BrowserController, chan control.UserStatus) {
	t.Helper()

	store, err := memstore.New(nil)
	require.NoError(t, err)

	user := userentity.New(userentity.Setup{Store: store}, userentity.Config{
		ServerURL:    "http://localhost:8065",
		WebSocketURL: "ws://localhost:8065",
		Username:     "testuser",
		Email:        "test@example.com",
		Password:     "testpass",
	})

	statusChan := make(chan control.UserStatus, 10)

	controller, err := New(1, user, "http://localhost:8065", statusChan)
	require.NoError(t, err)
	require.NotNil(t, controller)

	return controller, statusChan
}

func TestRun(t *testing.T) {
	t.Run("controller run fails if user is nil", func(t *testing.T) {
		statusChanWithoutUser := make(chan control.UserStatus, 10)
		defer close(statusChanWithoutUser)

		controllerWithEmptyUser := &BrowserController{
			id:              1,
			user:            nil,
			status:          statusChanWithoutUser,
			stopChan:        make(chan struct{}),
			stoppedChan:     make(chan struct{}),
			ltBrowserApiUrl: LTBrowserApi,
		}

		go controllerWithEmptyUser.Run()
		statusF := <-statusChanWithoutUser
		require.Error(t, statusF.Err)
		require.Equal(t, control.USER_STATUS_FAILED, statusF.Code)
		require.Contains(t, statusF.Err.Error(), "browser controller was not initialized")
	})

	t.Run("controller lifecycle with browser API not running", func(t *testing.T) {
		controllerWithUser, statusChan := newController(t)

		go controllerWithUser.Run()

		// Test that controller is running after Run is called
		status := <-statusChan
		require.NoError(t, status.Err)
		require.Equal(t, control.USER_STATUS_STARTED, status.Code)
		require.Equal(t, "browser controller started", status.Info)
		require.Equal(t, 1, status.ControllerId)
		require.Equal(t, controllerWithUser.user, status.User)

		// Drain any INFO messages from init actions (SignUp, Login, JoinTeam)
		var errorStatus control.UserStatus
		for {
			status = <-statusChan
			if status.Code == control.USER_STATUS_ERROR {
				errorStatus = status
				break
			}
			// Skip INFO messages from init actions
			require.Equal(t, control.USER_STATUS_INFO, status.Code)
		}

		require.Error(t, errorStatus.Err)
		require.Equal(t, control.USER_STATUS_ERROR, errorStatus.Code)

		status = <-statusChan
		require.Equal(t, control.USER_STATUS_STOPPED, status.Code)
		require.Equal(t, "browser controller stopped", status.Info)

		close(statusChan)
	})

	t.Run("controller has a fail status", func(t *testing.T) {
		controllerWithFailStatus, statusChan := newController(t)
		defer close(statusChan)

		controllerWithFailStatus.sendFailStatus("test failure")
		status := <-statusChan
		require.Equal(t, control.USER_STATUS_FAILED, status.Code)
		require.Contains(t, status.Err.Error(), "test failure")
	})
}

func TestStop(t *testing.T) {
	t.Run("controller stops properly when running", func(t *testing.T) {
		controllerRunning, statusChanRunning := newController(t)

		go controllerRunning.Run()
		startedStatus := <-statusChanRunning
		require.Equal(t, control.USER_STATUS_STARTED, startedStatus.Code)

		controllerRunning.Stop()

		// Drain any INFO messages and wait for the stop status message
		var stopStatus control.UserStatus
		for {
			status := <-statusChanRunning
			if status.Code == control.USER_STATUS_STOPPED {
				stopStatus = status
				break
			}
			// Skip INFO or ERROR messages from stop process
			require.True(t, status.Code == control.USER_STATUS_INFO || status.Code == control.USER_STATUS_ERROR)
		}

		require.Equal(t, control.USER_STATUS_STOPPED, stopStatus.Code)
		require.Equal(t, "browser controller stopped", stopStatus.Info)

		close(statusChanRunning)
	})

	t.Run("controller has stop status", func(t *testing.T) {
		controllerStopStatus, statusChanStop := newController(t)
		defer close(statusChanStop)

		controllerStopStatus.sendStopStatus()
		status := <-statusChanStop
		require.Equal(t, control.USER_STATUS_STOPPED, status.Code)
		require.Equal(t, "browser controller stopped", status.Info)
	})
}

func newUser(t *testing.T, username, password string, email string) *userentity.UserEntity {
	store, err := memstore.New(nil)
	require.NoError(t, err)
	return userentity.New(userentity.Setup{Store: store}, userentity.Config{
		ServerURL:    "http://localhost:8065",
		WebSocketURL: "ws://localhost:8065",
		Username:     username,
		Email:        email,
		Password:     password,
	})
}

type MockServerConfig struct {
	StatusCode      int
	ResponseType    string
	ErrorCode       string
	ErrorMessage    string
	ValidateRequest bool
	Method          string
}

func createMockServer(t *testing.T, config MockServerConfig) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if config.ValidateRequest {
			method := config.Method
			if method == "" {
				method = "POST"
			}
			require.Equal(t, method, r.Method)
			require.Equal(t, "/browsers", r.URL.Path)
			require.Equal(t, "application/json", r.Header.Get("Content-Type"))

			switch method {
			case http.MethodPost:
				var requestBody AddBrowserRequest
				err := json.NewDecoder(r.Body).Decode(&requestBody)
				require.NoError(t, err)
				require.Equal(t, "testuser", requestBody.User)
				require.Equal(t, "testpass", requestBody.Password)
			case http.MethodDelete:
				queryParams := r.URL.Query()
				require.Equal(t, "testuser", queryParams.Get("user"))
			}
		}

		statusCode := config.StatusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		w.WriteHeader(statusCode)

		switch config.ResponseType {
		case "success":
			response := BrowserAPIResponse{
				Success: true,
				Message: "Browser added successfully",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)

		case "error":
			response := BrowserAPIResponse{
				Success: false,
				Message: "Failed to add browser",
				Error: &struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				}{
					Code:    config.ErrorCode,
					Message: config.ErrorMessage,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)

		case "invalid_json":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json"))

		default:
			return
		}
	}))
}

func TestAddBrowser(t *testing.T) {
	t.Run("browser is added successfully", func(t *testing.T) {
		mockServer := createMockServer(t, MockServerConfig{
			ResponseType:    "success",
			ValidateRequest: true,
			Method:          "POST",
		})
		defer mockServer.Close()

		statusChanValid := make(chan control.UserStatus, 10)
		defer close(statusChanValid)

		controllerValid, err := New(1, newUser(t, "testuser", "testpass", "test@example.com"), "http://localhost:8065", statusChanValid)
		require.NoError(t, err)
		controllerValid.ltBrowserApiUrl = mockServer.URL

		_, err = controllerValid.addBrowser()
		require.NoError(t, err)
	})

	t.Run("fails due to API returned error", func(t *testing.T) {
		mockServerError := createMockServer(t, MockServerConfig{
			ResponseType: "error",
			ErrorCode:    "INVALID_USER",
			ErrorMessage: "User not found",
			Method:       "POST",
		})
		defer mockServerError.Close()

		statusChanError := make(chan control.UserStatus, 10)
		defer close(statusChanError)

		controllerError, err := New(1, newUser(t, "testuser", "testpass", "test@example.com"), "http://localhost:8065", statusChanError)
		require.NoError(t, err)
		controllerError.ltBrowserApiUrl = mockServerError.URL

		_, err = controllerError.addBrowser()
		require.Error(t, err)
		require.Contains(t, err.Error(), "browser API returned error: User not found")
	})

	t.Run("fails due to HTTP error", func(t *testing.T) {
		mockServerHTTPError := createMockServer(t, MockServerConfig{
			StatusCode: http.StatusInternalServerError,
			Method:     "POST",
		})
		defer mockServerHTTPError.Close()

		statusChanHTTPError := make(chan control.UserStatus, 10)
		defer close(statusChanHTTPError)

		controllerHTTPError, err := New(1, newUser(t, "testuser", "testpass", "test@example.com"), "http://localhost:8065", statusChanHTTPError)
		require.NoError(t, err)
		controllerHTTPError.ltBrowserApiUrl = mockServerHTTPError.URL

		_, err = controllerHTTPError.addBrowser()
		require.Error(t, err)
		require.Contains(t, err.Error(), "response status code: 500")
	})

	t.Run("fails due to empty username and email", func(t *testing.T) {
		userEmptyUsername := newUser(t, "", "testpass", "")
		statusChanEmptyUsername := make(chan control.UserStatus, 10)
		defer close(statusChanEmptyUsername)

		controllerEmptyUsername, err := New(1, userEmptyUsername, "http://localhost:8065", statusChanEmptyUsername)
		require.NoError(t, err)

		_, err = controllerEmptyUsername.addBrowser()
		require.Error(t, err)
		require.Contains(t, err.Error(), "username and email both are empty, either username or email is required")
	})

	t.Run("fails due to empty password", func(t *testing.T) {
		userEmptyPassword := newUser(t, "testuser", "", "test@example.com")
		statusChanEmptyPassword := make(chan control.UserStatus, 10)
		defer close(statusChanEmptyPassword)

		controllerEmptyPassword, err := New(1, userEmptyPassword, "http://localhost:8065", statusChanEmptyPassword)
		require.NoError(t, err)

		_, err = controllerEmptyPassword.addBrowser()
		require.Error(t, err)
		require.Contains(t, err.Error(), "password is empty")
	})
}

func TestRemoveBrowser(t *testing.T) {
	t.Run("browser is removed successfully", func(t *testing.T) {
		mockServer := createMockServer(t, MockServerConfig{
			ResponseType:    "success",
			ValidateRequest: true,
			Method:          "DELETE",
		})
		defer mockServer.Close()

		statusChanValid := make(chan control.UserStatus, 10)
		defer close(statusChanValid)

		controllerValid, err := New(1, newUser(t, "testuser", "testpass", "test@example.com"), "http://localhost:8065", statusChanValid)
		require.NoError(t, err)
		controllerValid.ltBrowserApiUrl = mockServer.URL

		err = controllerValid.removeBrowser()
		require.NoError(t, err)
	})

	t.Run("fails due to API returned error", func(t *testing.T) {
		mockServerError := createMockServer(t, MockServerConfig{
			ResponseType: "error",
			ErrorCode:    "BROWSER_NOT_FOUND",
			ErrorMessage: "Browser session not found",
			Method:       "DELETE",
		})
		defer mockServerError.Close()

		statusChanError := make(chan control.UserStatus, 10)
		defer close(statusChanError)

		controllerError, err := New(1, newUser(t, "testuser", "testpass", "test@example.com"), "http://localhost:8065", statusChanError)
		require.NoError(t, err)
		controllerError.ltBrowserApiUrl = mockServerError.URL

		err = controllerError.removeBrowser()
		require.Error(t, err)
		require.Contains(t, err.Error(), "browser API returned error: Browser session not found")
	})

	t.Run("fails due to HTTP error", func(t *testing.T) {
		mockServerHTTPError := createMockServer(t, MockServerConfig{
			StatusCode: http.StatusNotFound,
			Method:     "DELETE",
		})
		defer mockServerHTTPError.Close()

		statusChanHTTPError := make(chan control.UserStatus, 10)
		defer close(statusChanHTTPError)

		controllerHTTPError, err := New(1, newUser(t, "testuser", "testpass", "test@example.com"), "http://localhost:8065", statusChanHTTPError)
		require.NoError(t, err)
		controllerHTTPError.ltBrowserApiUrl = mockServerHTTPError.URL

		err = controllerHTTPError.removeBrowser()
		require.Error(t, err)
		require.Contains(t, err.Error(), "response status code: 404")
	})

	t.Run("fails due to empty username", func(t *testing.T) {
		userEmptyUsername := newUser(t, "", "testpass", "")
		statusChanEmptyUsername := make(chan control.UserStatus, 10)
		defer close(statusChanEmptyUsername)

		controllerEmptyUsername, err := New(1, userEmptyUsername, "http://localhost:8065", statusChanEmptyUsername)
		require.NoError(t, err)

		err = controllerEmptyUsername.removeBrowser()
		require.Error(t, err)
		require.Contains(t, err.Error(), "username and email both are empty, either username or email is required")
	})
}
