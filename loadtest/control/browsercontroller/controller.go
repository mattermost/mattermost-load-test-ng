// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package browsercontroller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

// This is the URL of the LTBrowser API server ran from /browser
const LT_BROWSER_API_URL = "http://localhost:5000"

// BrowserController is a controller that manages browser sessions
// by communicating with the browserLt API server.
type BrowserController struct {
	id              int
	user            user.User
	status          chan<- control.UserStatus
	rate            float64
	stopChan        chan struct{}
	stoppedChan     chan struct{}
	wg              *sync.WaitGroup
	ltBrowserApiUrl string
	httpClient      *http.Client
	isRunning       bool
	mu              sync.Mutex
}

// AddBrowserRequest represents the request body for adding a browser session
type AddBrowserRequest struct {
	// UserID is the username or email of the user to add a browser session
	UserID   string `json:"userId" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// RemoveBrowserRequest represents the request body for removing a browser session
type RemoveBrowserRequest struct {
	// UserID is the username or email of the user to remove a browser session
	UserID string `json:"userId" validate:"required"`
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
func New(id int, user user.User, status chan<- control.UserStatus) (*BrowserController, error) {
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
		rate:            1.0,
		stopChan:        make(chan struct{}),
		stoppedChan:     make(chan struct{}),
		wg:              &sync.WaitGroup{},
		ltBrowserApiUrl: LT_BROWSER_API_URL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		isRunning: false,
	}, nil
}

// Run starts the browser controller by creating a browser session.
// This corresponds to adding a user in the browser load test.
func (c *BrowserController) Run() {
	c.mu.Lock()
	if c.isRunning {
		c.mu.Unlock()
		return
	}
	c.isRunning = true
	c.mu.Unlock()

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
		c.mu.Lock()
		c.isRunning = false
		c.mu.Unlock()
		c.user.ClearUserData()
		c.sendStopStatus()
		close(c.stoppedChan)
	}()

	initActions := []control.UserAction{
		control.SignUp,
		control.Login,
		control.JoinTeam,
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

	if err := c.addBrowser(); err != nil {
		c.status <- c.newErrorStatus(control.NewUserError(err))
		return
	}
	c.status <- c.newInfoStatus("browser added successfully")

	// Wait until stop signal is received to stop the controller
	<-c.stopChan
}

// SetRate sets the relative speed of execution. For browser controller,
// this is a no-op since browser actions are managed by the Node.js server.
func (c *BrowserController) SetRate(rate float64) error {
	if rate < 0 {
		return errors.New("rate should be a positive value")
	}

	// Currently unused but should be stored anyways
	c.rate = rate
	return nil
}

// Stop stops the browser controller and removes the browser session.
// This corresponds to removing a user in the browser load test.
func (c *BrowserController) Stop() {
	c.mu.Lock()
	if !c.isRunning {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

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
// by the Node.js server through the browser sessions.
func (c *BrowserController) InjectAction(actionID string) error {
	return nil
}

// addBrowser makes a POST request to BrowserAPI to add a browser
func (c *BrowserController) addBrowser() error {
	// Get user credentials from the user entity
	userStore := c.user.Store()

	if userStore.Username() == "" {
		return fmt.Errorf("username is empty")
	}
	if userStore.Password() == "" {
		return fmt.Errorf("password is empty")
	}

	requestBody := AddBrowserRequest{
		UserID:   userStore.Username(),
		Password: userStore.Password(),
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("%s/browsers", c.ltBrowserApiUrl)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to add browser: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var apiResponse BrowserAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResponse.Success {
		errorMsg := "unknown error"
		if apiResponse.Error != nil {
			errorMsg = apiResponse.Error.Message
		}
		return fmt.Errorf("browser API returned error: %s", errorMsg)
	}

	return nil
}

// removeBrowser makes a DELETE request to BrowserAPI to remove the browser
func (c *BrowserController) removeBrowser() error {
	// Get user credentials from the user entity
	userStore := c.user.Store()

	if userStore.Username() == "" {
		return fmt.Errorf("username is empty")
	}

	requestBody := RemoveBrowserRequest{
		UserID: userStore.Username(),
	}

	// If username is empty, use email as userId
	if requestBody.UserID == "" {
		requestBody.UserID = userStore.Email()
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("%s/browsers", c.ltBrowserApiUrl)
	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to remove browser: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	var apiResponse BrowserAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResponse.Success {
		errorCode := "unknown error"
		errorMsg := "unknown error"
		if apiResponse.Error != nil {
			errorCode = apiResponse.Error.Code
			errorMsg = apiResponse.Error.Message
		}
		return fmt.Errorf("browser API returned error: %s-%s", errorCode, errorMsg)
	}

	return nil
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
