// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
)

// Coordinator represents a load-test coordinator.
// It exposes methods to manage a load-test coordinator resource through API.
type Coordinator struct {
	id     string
	apiURL string
	client *Client
}

// CoordinatorResponse contains the data returned by load-test coordinator API.
type coordinatorResponse struct {
	Id      string              `json:"id,omitempty"`      // The load-test coordinator unique identifier.
	Message string              `json:"message,omitempty"` // Message contains information about the response.
	Status  *coordinator.Status `json:"status,omitempty"`  // Status contains the current status of the coordinator.
	Error   string              `json:"error,omitempty"`   // Error is set if there was an error during the operation.
}

func (c *Coordinator) apiRequest(req *http.Request) (coordinatorResponse, error) {
	var res coordinatorResponse
	resp, err := c.client.httpClient.Do(req)
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
	} else if resp.StatusCode != http.StatusOK {
		return res, fmt.Errorf("coordinator: bad response status code %d", resp.StatusCode)
	}
	return res, nil
}

func (c *Coordinator) apiGet(url string) (coordinatorResponse, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return coordinatorResponse{}, fmt.Errorf("coordinator: failed to build request: %w", err)
	}
	return c.apiRequest(req)
}

func (c *Coordinator) apiPost(url string, data []byte) (coordinatorResponse, error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return coordinatorResponse{}, fmt.Errorf("coordinator: failed to build request: %w", err)
	}
	return c.apiRequest(req)
}

// Id returns the unique identifier for the coordinator resource.
func (c *Coordinator) Id() string {
	return c.id
}

// Status retrieves and returns the status for the load-test agent.
// It also returns an error in case of failure.
func (c *Coordinator) Status() (coordinator.Status, error) {
	var status coordinator.Status
	resp, err := c.apiGet(c.apiURL)
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
	resp, err := c.apiPost(c.apiURL+"/run", nil)
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
	resp, err := c.apiPost(c.apiURL+"/stop", nil)
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
	req, err := http.NewRequest("DELETE", c.apiURL, nil)
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
