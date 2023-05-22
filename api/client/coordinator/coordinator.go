// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
)

// Coordinator represents a load-test coordinator.
// It exposes methods to manage a load-test coordinator resource through API.
type Coordinator struct {
	id     string
	apiURL string
	client *http.Client
}

// CoordinatorResponse contains the data returned by load-test coordinator API.
type CoordinatorResponse struct {
	Id      string              `json:"id,omitempty"`      // The load-test coordinator unique identifier.
	Message string              `json:"message,omitempty"` // Message contains information about the response.
	Status  *coordinator.Status `json:"status,omitempty"`  // Status contains the current status of the coordinator.
	Error   string              `json:"error,omitempty"`   // Error is set if there was an error during the operation.
}

func (c *Coordinator) apiRequest(req *http.Request) (CoordinatorResponse, error) {
	var res CoordinatorResponse
	resp, err := c.client.Do(req)
	if err != nil {
		return res, fmt.Errorf("coordinator: post request failed: %w", err)
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return res, fmt.Errorf("coordinator: failed to decode coordinator api response: %w", err)
	}
	if res.Error != "" {
		return res, fmt.Errorf("coordinator: coordinator api request error: %s", res.Error)
	} else if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return res, fmt.Errorf("coordinator: bad response status code %d", resp.StatusCode)
	}
	return res, nil
}

func (c *Coordinator) apiGet(url string) (CoordinatorResponse, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return CoordinatorResponse{}, fmt.Errorf("coordinator: failed to build request: %w", err)
	}
	return c.apiRequest(req)
}

func (c *Coordinator) apiPost(url string, data []byte) (CoordinatorResponse, error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return CoordinatorResponse{}, fmt.Errorf("coordinator: failed to build request: %w", err)
	}
	return c.apiRequest(req)
}

// New creates and initializes a new instance of Coordinator.
// Returns an error in case of failure.
func New(id, serverURL string, client *http.Client) (*Coordinator, error) {
	if id == "" {
		return nil, errors.New("coordinator: id should not be empty")
	}
	if serverURL == "" {
		return nil, errors.New("coordinator: serverURL should not be empty")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Coordinator{
		id:     id,
		apiURL: serverURL + "/coordinator/",
		client: client,
	}, nil
}

// Id returns the unique identifier for the coordinator resource.
func (c *Coordinator) Id() string {
	return c.id
}

// Create creates a new coordinator resource with the given configs.
// Returns the coordinator status or an error in case of failure.
func (c *Coordinator) Create(coordConfig *coordinator.Config, ltConfig *loadtest.Config) (coordinator.Status, error) {
	var status coordinator.Status
	if coordConfig == nil {
		return status, errors.New("client: coordConfig should not be nil")
	}
	if ltConfig == nil {
		return status, errors.New("client: ltConfig should not be nil")
	}

	data := struct {
		CoordinatorConfig *coordinator.Config
		LoadTestConfig    *loadtest.Config
	}{
		coordConfig,
		ltConfig,
	}

	configData, err := json.Marshal(data)
	if err != nil {
		return status, err
	}
	resp, err := c.apiPost(c.apiURL+"create?id="+c.id, configData)
	if err != nil {
		return status, err
	}

	status = *resp.Status
	return status, nil
}

// Status retrieves and returns the status for the load-test agent.
// It also returns an error in case of failure.
func (c *Coordinator) Status() (coordinator.Status, error) {
	var status coordinator.Status
	resp, err := c.apiGet(c.apiURL + c.id)
	if err != nil {
		return status, err
	}
	status = *resp.Status
	return status, nil
}

// Run starts the coordinator.
// Returns the coordinator status or an error in case of failure.
func (c *Coordinator) Run() (coordinator.Status, error) {
	var status coordinator.Status
	resp, err := c.apiPost(c.apiURL+c.id+"/run", nil)
	if err != nil {
		return status, err
	}
	status = *resp.Status
	return status, nil
}

// Stop stops the coordinator.
// Returns the coordinator status or an error in case of failure.
func (c *Coordinator) Stop() (coordinator.Status, error) {
	var status coordinator.Status
	resp, err := c.apiPost(c.apiURL+c.id+"/stop", nil)
	if err != nil {
		return status, err
	}
	status = *resp.Status
	return status, nil
}

// Destroy stops (if running) and destroys the coordinator resource.
// Returns the coordinator status or an error in case of failure.
func (c *Coordinator) Destroy() (coordinator.Status, error) {
	var status coordinator.Status
	req, err := http.NewRequest("DELETE", c.apiURL+c.id, nil)
	if err != nil {
		return status, fmt.Errorf("coordinator: failed to build request: %w", err)
	}
	resp, err := c.apiRequest(req)
	if err != nil {
		return status, err
	}
	status = *resp.Status
	return status, nil
}

// Stop stops the coordinator.
// Returns the coordinator status or an error in case of failure.
func (c *Coordinator) InjectAction(actionID string) (coordinator.Status, error) {
	var status coordinator.Status
	resp, err := c.apiPost(c.apiURL+c.id+"/inject?action="+actionID, nil)
	if err != nil {
		return status, err
	}
	status = *resp.Status
	return status, nil
}
