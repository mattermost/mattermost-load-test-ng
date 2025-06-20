// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package browsercontroller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

// TODO: Get this from the coordinator file
const BROWSER_AGENT_API_URL = "BROWSER_AGENT_API_URL"

// BrowserController is a controller that manages browser sessions
// by communicating with the browserLt API server.
type BrowserController struct {
	id            int
	user          user.User
	status        chan<- control.UserStatus
	rate          float64
	stopChan      chan struct{}
	stoppedChan   chan struct{}
	wg            *sync.WaitGroup
	browserAPIURL string
	httpClient    *http.Client
	isRunning     bool
	mu            sync.Mutex
}

// AddBrowserRequest represents the request body for adding a browser session
type AddBrowserRequest struct {
	UserID   string `json:"userId"`
	Password string `json:"password"`
}

// RemoveBrowserRequest represents the request body for removing a browser session
type RemoveBrowserRequest struct {
	UserID string `json:"userId"`
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

	// Get browser API URL from environment variable (same as browser server uses)
	browserAPIURL := os.Getenv("BROWSER_AGENT_API_URL")
	if browserAPIURL == "" {
		return nil, errors.New("BROWSER_AGENT_API_URL environment variable is required")
	}

	return &BrowserController{
		id:            id,
		user:          user,
		status:        status,
		rate:          1.0,
		stopChan:      make(chan struct{}),
		stoppedChan:   make(chan struct{}),
		wg:            &sync.WaitGroup{},
		browserAPIURL: browserAPIURL,
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
		c.sendFailStatus("controller was not initialized")
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
		c.sendStopStatus()
		close(c.stoppedChan)
	}()

	// Create browser session (equivalent to adding a user)
	if err := c.createBrowserSession(); err != nil {
		c.status <- c.newErrorStatus(control.NewUserError(err))
		return
	}

	c.status <- c.newInfoStatus("browser session created successfully")

	// Wait until stop is called
	<-c.stopChan
}

// SetRate sets the relative speed of execution. For browser controller,
// this is a no-op since browser actions are managed by the Node.js server.
func (c *BrowserController) SetRate(rate float64) error {
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

	// Remove browser session before stopping
	if err := c.removeBrowserSession(); err != nil {
		c.status <- c.newErrorStatus(control.NewUserError(err))
	} else {
		c.status <- c.newInfoStatus("browser session removed successfully")
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

// createBrowserSession makes a POST request to create a new browser session
func (c *BrowserController) createBrowserSession() error {
	// Get user credentials from the user entity
	userStore := c.user.Store()

	requestBody := AddBrowserRequest{
		UserID:   userStore.Username(),
		Password: userStore.Password(),
	}

	// If username is empty, use email as userId
	if requestBody.UserID == "" {
		requestBody.UserID = userStore.Email()
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("%s/browsers", c.browserAPIURL)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create browser session: %w", err)
	}
	defer resp.Body.Close()

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

// removeBrowserSession makes a DELETE request to remove the browser session
func (c *BrowserController) removeBrowserSession() error {
	// Get user credentials from the user entity
	userStore := c.user.Store()

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

	url := fmt.Sprintf("%s/browsers", c.browserAPIURL)
	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to remove browser session: %w", err)
	}
	defer resp.Body.Close()

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
