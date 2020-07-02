// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
)

// Agent represents a load-test agent.
// It exposes methods to manage a load-test agent resource through API.
type Agent struct {
	id     string
	apiURL string
	client *Client
}

// agentResponse contains the data returned by the load-test agent API.
type agentResponse struct {
	Id      string           `json:"id,omitempty"`      // The load-test agent unique identifier.
	Message string           `json:"message,omitempty"` // Message contains information about the response.
	Status  *loadtest.Status `json:"status,omitempty"`  // Status contains the current status of the load test.
	Error   string           `json:"error,omitempty"`   // Error is set if there was an error during the operation.
}

func (a *Agent) apiRequest(req *http.Request) (agentResponse, error) {
	var res agentResponse
	resp, err := a.client.httpClient.Do(req)
	if err != nil {
		return res, fmt.Errorf("agent: post request failed: %w", err)
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return res, fmt.Errorf("agent: failed to decode load-test agent api response: %w", err)
	}
	if res.Error != "" {
		return res, fmt.Errorf("agent: load-test agent api request error: %s", res.Error)
	} else if resp.StatusCode != http.StatusOK {
		return res, fmt.Errorf("agent: bad response status code %d", resp.StatusCode)
	}
	return res, nil
}

func (a *Agent) apiGet(url string) (agentResponse, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return agentResponse{}, fmt.Errorf("agent: failed to build request: %w", err)
	}
	return a.apiRequest(req)
}

func (a *Agent) apiPost(url string, data []byte) (agentResponse, error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return agentResponse{}, fmt.Errorf("agent: failed to build request: %w", err)
	}
	return a.apiRequest(req)
}

// Id returns the unique identifier for the load-test agent resource.
func (a *Agent) Id() string {
	return a.id
}

// Status retrieves and returns the status for the load-test agent.
// It also returns an error in case of failure.
func (a *Agent) Status() (loadtest.Status, error) {
	var status loadtest.Status
	resp, err := a.apiGet(a.apiURL)
	if err != nil {
		return status, err
	}
	status = *resp.Status
	return status, nil
}

// Run starts the load-test agent. It starts the execution of a load-test.
// Returns the load-test agent status or an error in case of failure.
func (a *Agent) Run() (loadtest.Status, error) {
	var status loadtest.Status
	resp, err := a.apiPost(a.apiURL+"/run", nil)
	if err != nil {
		return status, err
	}
	status = *resp.Status
	return status, nil
}

// Stop stops the load-test agent. It stops the execution of the running
// Returns the load-test agent status or an error in case of failure.
func (a *Agent) Stop() (loadtest.Status, error) {
	var status loadtest.Status
	resp, err := a.apiPost(a.apiURL+"/stop", nil)
	if err != nil {
		return status, err
	}
	status = *resp.Status
	return status, nil
}

// AddUsers attempts to increment by numUsers the number of active users.
// Returns the load-test agent status or an error in case of failure.
func (a *Agent) AddUsers(numUsers int) (loadtest.Status, error) {
	var status loadtest.Status
	resp, err := a.apiPost(a.apiURL+"/addusers?amount="+strconv.Itoa(numUsers), nil)
	if err != nil {
		return status, err
	}
	status = *resp.Status
	return status, nil
}

// AddUsers attempts to decrease by numUsers the number of active users.
// Returns the load-test agent status or an error in case of failure.
func (a *Agent) RemoveUsers(numUsers int) (loadtest.Status, error) {
	var status loadtest.Status
	resp, err := a.apiPost(a.apiURL+"/removeusers?amount="+strconv.Itoa(numUsers), nil)
	if err != nil {
		return status, err
	}
	status = *resp.Status
	return status, nil
}

// Destroy stops (if running) and destroys the load-test agent resource.
// Returns the load-test agent status or an error in case of failure.
func (a *Agent) Destroy() (loadtest.Status, error) {
	var status loadtest.Status
	req, err := http.NewRequest("DELETE", a.apiURL, nil)
	if err != nil {
		return status, fmt.Errorf("agent: failed to build request: %w", err)
	}
	resp, err := a.apiRequest(req)
	if err != nil {
		return status, err
	}
	status = *resp.Status
	return status, nil
}
