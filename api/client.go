// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"
)

const (
	agentApiSuffix = "/loadagent/"
	coordApiSuffix = "/coordinator/"
)

// Client is a simple http.Client wrapper used to create API resources.
type Client struct {
	serverURL  string
	httpClient *http.Client
}

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

// NewClient creates and initializes a new instance of Client.
func NewClient(serverURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		serverURL:  serverURL,
		httpClient: httpClient,
	}
}

// CreateAgent creates a new load-test agent resource with the given id and configs.
// It returns an initialized instance of Agent or an error in case of failure.
func (c *Client) CreateAgent(id string, ltConfig *loadtest.Config, ucConfig interface{}) (*Agent, error) {
	if ltConfig == nil {
		return nil, errors.New("agent: ltConfig should not be nil")
	}
	if ucConfig == nil {
		return nil, errors.New("agent: ucConfig should not be nil")
	}

	data := struct {
		LoadTestConfig         *loadtest.Config
		SimpleControllerConfig *simplecontroller.Config `json:",omitempty"`
		SimulControllerConfig  *simulcontroller.Config  `json:",omitempty"`
	}{
		LoadTestConfig: ltConfig,
	}

	switch ltConfig.UserControllerConfiguration.Type {
	case loadtest.UserControllerSimple:
		var scc *simplecontroller.Config
		scc, ok := ucConfig.(*simplecontroller.Config)
		if !ok {
			return nil, errors.New("agent: ucConfig has the wrong type")
		}
		data.SimpleControllerConfig = scc
	case loadtest.UserControllerSimulative:
		scc, ok := ucConfig.(*simulcontroller.Config)
		if !ok {
			return nil, errors.New("agent: ucConfig has the wrong type")
		}
		data.SimulControllerConfig = scc
	default:
		return nil, errors.New("agent: UserController type is not set")
	}

	configData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := c.serverURL + agentApiSuffix + "create?id=" + id
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(configData))
	if err != nil {
		return nil, fmt.Errorf("agent: post request failed: %w", err)
	}
	defer resp.Body.Close()

	var res agentResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, fmt.Errorf("agent: failed to decode load-test agent api response: %w", err)
	}
	if res.Error != "" {
		return nil, fmt.Errorf("agent: load-test agent api request error: %s", res.Error)
	} else if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("agent: bad response status code %d", resp.StatusCode)
	}

	return &Agent{
		id:     res.Id,
		apiURL: c.serverURL + agentApiSuffix + res.Id,
		client: c,
	}, nil
}

// CreateCoordinator creates a new coordinator resource with the given id and configs.
// It returns an initialized instance of Coordinator or an error in case of failure.
func (c *Client) CreateCoordinator(id string, coordConfig *coordinator.Config, ltConfig *loadtest.Config) (*Coordinator, error) {
	if coordConfig == nil {
		return nil, errors.New("coordinator: coordConfig should not be nil")
	}
	if ltConfig == nil {
		return nil, errors.New("coordinator: ltConfig should not be nil")
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
		return nil, err
	}

	url := c.serverURL + coordApiSuffix + "create?id=" + id
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(configData))
	if err != nil {
		return nil, fmt.Errorf("coordinator: post request failed: %w", err)
	}
	defer resp.Body.Close()

	var res coordinatorResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, fmt.Errorf("coordinator: failed to decode coordinator api response: %w", err)
	}
	if res.Error != "" {
		return nil, fmt.Errorf("coordinator: coordinator api request error: %s", res.Error)
	} else if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("coordinator: bad response status code %d", resp.StatusCode)
	}

	return &Coordinator{
		id:     res.Id,
		apiURL: c.serverURL + coordApiSuffix + res.Id,
		client: c,
	}, nil
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
