// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package browsercontroller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

// This is the URL of the LTBrowser API server ran from /browser
const LTBrowserApi = "http://localhost:5000"

// BrowserController is a controller that manages browser sessions
// by communicating with the browserLt API server.
type BrowserController struct {
	id              int
	user            user.User
	status          chan<- control.UserStatus
	stopChan        chan struct{}
	stoppedChan     chan struct{}
	wg              *sync.WaitGroup
	ltBrowserApiUrl string
	mmServerUrl     string
	httpClient      *http.Client
}

// AddBrowserRequest represents the request body for adding a browser session
type AddBrowserRequest struct {
	// User is the username or email of the user to add a browser session
	User      string `json:"user"`
	Password  string `json:"password"`
	ServerURL string `json:"server_url"`
}

// BrowserAPIResponse represents the response from the browser API
type BrowserAPIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// New creates and initializes a new BrowserController with given parameters.
// TODO: serverURL should be removed from here and LTBrowser API should be updated to use the serverURL from the config.json
func New(id int, user user.User, serverURL string, status chan<- control.UserStatus) (*BrowserController, error) {
	if user == nil {
		return nil, errors.New("user cannot be nil")
	}
	if status == nil {
		return nil, errors.New("status channel cannot be nil")
	}

	return &BrowserController{
		id:              id,
		user:            user,
		status:          status,
		stopChan:        make(chan struct{}),
		stoppedChan:     make(chan struct{}),
		wg:              &sync.WaitGroup{},
		ltBrowserApiUrl: LTBrowserApi,
		mmServerUrl:     serverURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Run starts the browser controller by creating a browser session.
// This corresponds to adding a user in the browser load test.
func (c *BrowserController) Run() {
	if c.user == nil {
		c.sendFailStatus("browser controller was not initialized")
		return
	}

	c.status <- control.UserStatus{
		ControllerId: c.id,
		User:         c.user,
		Info:         "browser controller started",
		Code:         control.USER_STATUS_STARTED,
	}

	defer func() {
		c.user.ClearUserData()
		c.sendStopStatus()
		close(c.stoppedChan)
	}()

	initActions := []control.UserAction{
		control.SignUp,
	}

	for _, action := range initActions {
		if resp := action(c.user); resp.Err != nil {
			c.status <- c.newErrorStatus(resp.Err)
			return
		} else if resp.Info != "" {
			c.status <- c.newInfoStatus(resp.Info)
		}

		// If stop signal is received then stop the controller
		select {
		case <-c.stopChan:
			return
		default:
		}
	}

	response, err := c.addBrowser()
	if err != nil {
		fmt.Println("failed to add browser with the browser controller", err, response)
		c.status <- c.newErrorStatus(control.NewUserError(err))
		return
	}
	c.status <- c.newInfoStatus(fmt.Sprintf("browser added successfully, response: %+v", response.Message))

	// Wait until stop signal is received to stop the controller
	<-c.stopChan
}

// SetRate is a no-op for browser controller since actions and their speed are
// managed by the Browser service.
func (c *BrowserController) SetRate(rate float64) error {
	return nil
}

// Stop stops the browser controller and removes the browser session.
// This corresponds to removing a user in the browser load test.
func (c *BrowserController) Stop() {
	// Remove browser before stopping
	if err := c.removeBrowser(); err != nil {
		c.status <- c.newErrorStatus(control.NewUserError(err))
	} else {
		c.status <- c.newInfoStatus("browser removed successfully")
	}

	close(c.stopChan)
	<-c.stoppedChan

	// Re-initialize channels for potential reuse
	c.stopChan = make(chan struct{})
	c.stoppedChan = make(chan struct{})
}

// InjectAction is a no-op for browser controller since actions are managed
// by the Browser service.
func (c *BrowserController) InjectAction(actionID string) error {
	return nil
}

// This is HTTP client helper which is used to make API requests to the LTBrowser API server.
func (c *BrowserController) makeRequestToLTBrowserApi(method string, requestBody interface{}, queryParams url.Values) (*BrowserAPIResponse, error) {
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	const browserRoute = "/browsers"

	// Use url.JoinPath to properly construct the URL
	apiURL, err := url.JoinPath(c.ltBrowserApiUrl, browserRoute)
	if err != nil {
		return nil, fmt.Errorf("failed to construct URL: %w", err)
	}

	if queryParams != nil {
		apiURL += "?" + queryParams.Encode()
	}

	req, err := http.NewRequest(method, apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create %s request: %w", method, err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make %s request: %w", method, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed as response is in-between (200,300]: response status code: %d, response body: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResponse BrowserAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResponse.Success {
		errorMsg := "unknown error message"
		errorCode := "unknown error code"
		if apiResponse.Error != nil {
			errorMsg = apiResponse.Error.Message
			errorCode = apiResponse.Error.Code
		}
		return nil, fmt.Errorf("browser API returned error: %s, error code: %s", errorMsg, errorCode)
	}

	return &apiResponse, nil
}

func (c *BrowserController) addBrowser() (*BrowserAPIResponse, error) {
	userStore := c.user.Store()

	if userStore.Username() == "" && userStore.Email() == "" {
		return nil, fmt.Errorf("username and email both are empty, either username or email is required")
	}

	userNameOrEmail := userStore.Username()
	if userNameOrEmail == "" {
		userNameOrEmail = userStore.Email()
	}

	if userStore.Password() == "" {
		return nil, fmt.Errorf("password is empty")
	}

	requestBody := AddBrowserRequest{
		User:      userNameOrEmail,
		Password:  userStore.Password(),
		ServerURL: c.mmServerUrl,
	}

	response, err := c.makeRequestToLTBrowserApi(http.MethodPost, requestBody, nil)
	return response, err
}

func (c *BrowserController) removeBrowser() error {
	userStore := c.user.Store()

	if userStore.Username() == "" && userStore.Email() == "" {
		return fmt.Errorf("username and email both are empty, either username or email is required")
	}

	// If username is empty, use email as userId
	userNameOrEmail := userStore.Username()
	if userNameOrEmail == "" {
		userNameOrEmail = userStore.Email()
	}

	userIdQueryParams := url.Values{}
	userIdQueryParams.Set("user", userNameOrEmail)

	_, err := c.makeRequestToLTBrowserApi(http.MethodDelete, nil, userIdQueryParams)
	return err
}

// Helper methods for status reporting
func (c *BrowserController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{
		ControllerId: c.id,
		User:         c.user,
		Code:         control.USER_STATUS_FAILED,
		Err:          errors.New(reason),
	}
}

func (c *BrowserController) sendStopStatus() {
	c.status <- control.UserStatus{
		ControllerId: c.id,
		User:         c.user,
		Info:         "browser controller stopped",
		Code:         control.USER_STATUS_STOPPED,
	}
}

func (c *BrowserController) newErrorStatus(err error) control.UserStatus {
	return control.UserStatus{
		ControllerId: c.id,
		User:         c.user,
		Code:         control.USER_STATUS_ERROR,
		Err:          err,
	}
}

func (c *BrowserController) newInfoStatus(info string) control.UserStatus {
	return control.UserStatus{
		ControllerId: c.id,
		User:         c.user,
		Info:         info,
		Code:         control.USER_STATUS_INFO,
	}
}

// Ensure BrowserController implements UserController interface
var _ control.UserController = (*BrowserController)(nil)
