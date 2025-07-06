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

	// Test that the controller is created successfully
	controller, err := New(1, user, statusChan)
	require.NoError(t, err)
	require.NotNil(t, controller)
	require.Equal(t, 1, controller.id)
	require.Equal(t, user, controller.user)
	require.Equal(t, LT_BROWSER_API_URL, controller.ltBrowserApiUrl)
	require.Equal(t, 1.0, controller.rate)
	require.NotNil(t, controller.httpClient)
	require.False(t, controller.isRunning)

	// Test that the controller is not created if the user is nil
	controller, err = New(1, nil, statusChan)
	require.Error(t, err)
	require.Nil(t, controller)
	require.Contains(t, err.Error(), "user cannot be nil")

	// Test that the controller is not created if the status channel is nil
	controller, err = New(1, user, nil)
	require.Error(t, err)
	require.Nil(t, controller)
	require.Contains(t, err.Error(), "status channel cannot be nil")

	close(statusChan)
}

func newController(t *testing.T) (*BrowserController, chan control.UserStatus) {
	t.Helper()

	// Create a test user
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

	controller, err := New(1, user, statusChan)
	require.NoError(t, err)
	require.NotNil(t, controller)

	return controller, statusChan
}

func TestRun(t *testing.T) {
	// Create a controller with a nil user
	statusChanWithoutUser := make(chan control.UserStatus, 10)
	controllerWithEmptyUser := &BrowserController{
		id:              1,
		user:            nil,
		status:          statusChanWithoutUser,
		rate:            1.0,
		stopChan:        make(chan struct{}),
		stoppedChan:     make(chan struct{}),
		ltBrowserApiUrl: LT_BROWSER_API_URL,
		isRunning:       false,
	}

	// Test that controller's Run method fails if the user is nil
	go controllerWithEmptyUser.Run()
	statusF := <-statusChanWithoutUser
	require.Error(t, statusF.Err)
	require.Equal(t, control.USER_STATUS_FAILED, statusF.Code)
	require.Contains(t, statusF.Err.Error(), "browser controller was not initialized")
	close(statusChanWithoutUser)

	// Create a controller with a user
	controllerWithUser, statusChan := newController(t)

	// Test that controller is not running initially and run the controller
	require.False(t, controllerWithUser.isRunning)
	go controllerWithUser.Run()

	// Test that controller is running after Run is called
	status := <-statusChan
	require.NoError(t, status.Err)
	require.Equal(t, control.USER_STATUS_STARTED, status.Code)
	require.Equal(t, "browser controller started", status.Info)
	require.Equal(t, 1, status.ControllerId)
	require.Equal(t, controllerWithUser.user, status.User)
	require.True(t, controllerWithUser.isRunning)

	// Test that controller fails as BrowserAPI is not running
	status = <-statusChan
	require.Error(t, status.Err)
	require.Equal(t, control.USER_STATUS_ERROR, status.Code)
	require.Contains(t, status.Err.Error(), "HTTP error")

	// Wait for stopped status as controller is stopped automatically after error
	status = <-statusChan
	require.Equal(t, control.USER_STATUS_STOPPED, status.Code)
	require.Equal(t, "browser controller stopped", status.Info)

	// Create a controller that has a fail status
	controllerWithFailStatus, statusChan := newController(t)
	controllerWithFailStatus.sendFailStatus("test failure")
	status = <-statusChan
	require.Equal(t, control.USER_STATUS_FAILED, status.Code)
	require.Contains(t, status.Err.Error(), "test failure")
	close(statusChan)
}

func TestStop(t *testing.T) {
	// Create a controller that is not ran yet
	controllerNotRunning, statusChanNotRunning := newController(t)
	require.False(t, controllerNotRunning.isRunning)

	// Test that stopping a controller that is not running does not cause issues
	controllerNotRunning.Stop()
	require.False(t, controllerNotRunning.isRunning)
	close(statusChanNotRunning)

	// Create a controller that will be run and will be stopped later
	controllerRunning, statusChanRunning := newController(t)
	go controllerRunning.Run()
	startedStatus := <-statusChanRunning
	require.Equal(t, control.USER_STATUS_STARTED, startedStatus.Code)
	require.True(t, controllerRunning.isRunning)
	controllerRunning.Stop()

	// Test that the controller is not running anymore
	require.False(t, controllerRunning.isRunning)
	close(statusChanRunning)

	// Create a controller that will have a stop status
	controllerStopStatus, statusChanStop := newController(t)
	controllerStopStatus.sendStopStatus()
	status := <-statusChanStop

	// Test that the controller has a stop status
	require.Equal(t, control.USER_STATUS_STOPPED, status.Code)
	require.Equal(t, "browser controller stopped", status.Info)
	close(statusChanStop)
}

func newUser(t *testing.T, username, password string) *userentity.UserEntity {
	store, err := memstore.New(nil)
	require.NoError(t, err)
	return userentity.New(userentity.Setup{Store: store}, userentity.Config{
		ServerURL:    "http://localhost:8065",
		WebSocketURL: "ws://localhost:8065",
		Username:     username,
		Email:        "test@example.com",
		Password:     password,
	})
}

type MockServerConfig struct {
	StatusCode      int
	ResponseType    string // "success", "error", "invalid_json"
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
				method = "POST" // Default to POST for backward compatibility
			}
			require.Equal(t, method, r.Method)
			require.Equal(t, "/browsers", r.URL.Path)
			require.Equal(t, "application/json", r.Header.Get("Content-Type"))

			// Only validate request body for methods that typically have a body
			if method == "POST" || method == "DELETE" {
				if method == "POST" {
					var requestBody AddBrowserRequest
					err := json.NewDecoder(r.Body).Decode(&requestBody)
					require.NoError(t, err)
					require.Equal(t, "testuser", requestBody.UserID)
					require.Equal(t, "testpass", requestBody.Password)
				} else if method == "DELETE" {
					var requestBody RemoveBrowserRequest
					err := json.NewDecoder(r.Body).Decode(&requestBody)
					require.NoError(t, err)
					require.Equal(t, "testuser", requestBody.UserID)
				}
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
			// For HTTP errors, just return the status code (no body)
			return
		}
	}))
}

func TestAddBrowser(t *testing.T) {
	// Create a mock server that returns a successful response
	mockServer := createMockServer(t, MockServerConfig{
		ResponseType:    "success",
		ValidateRequest: true,
		Method:          "POST",
	})
	defer mockServer.Close()

	statusChanValid := make(chan control.UserStatus, 10)
	controllerValid, err := New(1, newUser(t, "testuser", "testpass"), statusChanValid)
	require.NoError(t, err)
	controllerValid.ltBrowserApiUrl = mockServer.URL

	// Test that the browser is added successfully
	err = controllerValid.addBrowser()
	require.NoError(t, err)
	close(statusChanValid)

	// Create a mock server that returns an API error response
	mockServerError := createMockServer(t, MockServerConfig{
		ResponseType: "error",
		ErrorCode:    "INVALID_USER",
		ErrorMessage: "User not found",
		Method:       "POST",
	})
	defer mockServerError.Close()

	statusChanError := make(chan control.UserStatus, 10)
	controllerError, err := New(1, newUser(t, "testuser", "testpass"), statusChanError)
	require.NoError(t, err)
	controllerError.ltBrowserApiUrl = mockServerError.URL

	// Test that the addBrowser fails due to API returned error
	err = controllerError.addBrowser()
	require.Error(t, err)
	require.Contains(t, err.Error(), "browser API returned error: User not found")
	close(statusChanError)

	// Create a mock server that will return an HTTP error
	mockServerHTTPError := createMockServer(t, MockServerConfig{
		StatusCode: http.StatusInternalServerError,
		Method:     "POST",
	})
	defer mockServerHTTPError.Close()

	statusChanHTTPError := make(chan control.UserStatus, 10)
	controllerHTTPError, err := New(1, newUser(t, "testuser", "testpass"), statusChanHTTPError)
	require.NoError(t, err)
	controllerHTTPError.ltBrowserApiUrl = mockServerHTTPError.URL

	// Test that the addBrowser fails due to HTTP error
	err = controllerHTTPError.addBrowser()
	require.Error(t, err)
	require.Contains(t, err.Error(), "HTTP error: 500")
	close(statusChanHTTPError)

	// Test that the addBrowser fails due to empty username
	userEmptyUsername := newUser(t, "", "testpass")
	statusChanEmptyUsername := make(chan control.UserStatus, 10)
	controllerEmptyUsername, err := New(1, userEmptyUsername, statusChanEmptyUsername)
	require.NoError(t, err)
	err = controllerEmptyUsername.addBrowser()

	require.Error(t, err)
	require.Contains(t, err.Error(), "username is empty")
	close(statusChanEmptyUsername)

	// Test that the addBrowser fails due to empty password
	userEmptyPassword := newUser(t, "testuser", "")
	statusChanEmptyPassword := make(chan control.UserStatus, 10)
	controllerEmptyPassword, err := New(1, userEmptyPassword, statusChanEmptyPassword)
	require.NoError(t, err)
	err = controllerEmptyPassword.addBrowser()

	require.Error(t, err)
	require.Contains(t, err.Error(), "password is empty")
	close(statusChanEmptyPassword)
}

func TestRemoveBrowser(t *testing.T) {
	// Create a mock server that returns a successful response
	mockServer := createMockServer(t, MockServerConfig{
		ResponseType:    "success",
		ValidateRequest: true,
		Method:          "DELETE",
	})
	defer mockServer.Close()

	statusChanValid := make(chan control.UserStatus, 10)
	controllerValid, err := New(1, newUser(t, "testuser", "testpass"), statusChanValid)
	require.NoError(t, err)
	controllerValid.ltBrowserApiUrl = mockServer.URL

	// Test that the browser is removed successfully
	err = controllerValid.removeBrowser()
	require.NoError(t, err)
	close(statusChanValid)

	// Create a mock server that returns an API error response
	mockServerError := createMockServer(t, MockServerConfig{
		ResponseType: "error",
		ErrorCode:    "BROWSER_NOT_FOUND",
		ErrorMessage: "Browser session not found",
		Method:       "DELETE",
	})
	defer mockServerError.Close()

	statusChanError := make(chan control.UserStatus, 10)
	controllerError, err := New(1, newUser(t, "testuser", "testpass"), statusChanError)
	require.NoError(t, err)
	controllerError.ltBrowserApiUrl = mockServerError.URL

	// Test that the removeBrowser fails due to API returned error
	err = controllerError.removeBrowser()
	require.Error(t, err)
	require.Contains(t, err.Error(), "browser API returned error: BROWSER_NOT_FOUND-Browser session not found")
	close(statusChanError)

	// Create a mock server that will return an HTTP error
	mockServerHTTPError := createMockServer(t, MockServerConfig{
		StatusCode: http.StatusNotFound,
		Method:     "DELETE",
	})
	defer mockServerHTTPError.Close()

	statusChanHTTPError := make(chan control.UserStatus, 10)
	controllerHTTPError, err := New(1, newUser(t, "testuser", "testpass"), statusChanHTTPError)
	require.NoError(t, err)
	controllerHTTPError.ltBrowserApiUrl = mockServerHTTPError.URL

	// Test that the removeBrowser fails due to HTTP error
	err = controllerHTTPError.removeBrowser()
	require.Error(t, err)
	require.Contains(t, err.Error(), "HTTP error: 404")
	close(statusChanHTTPError)

	// Test that the removeBrowser fails due to empty username
	userEmptyUsername := newUser(t, "", "testpass")
	statusChanEmptyUsername := make(chan control.UserStatus, 10)
	controllerEmptyUsername, err := New(1, userEmptyUsername, statusChanEmptyUsername)
	require.NoError(t, err)
	err = controllerEmptyUsername.removeBrowser()

	require.Error(t, err)
	require.Contains(t, err.Error(), "username is empty")
	close(statusChanEmptyUsername)
}
