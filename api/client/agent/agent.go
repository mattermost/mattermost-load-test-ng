// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"
)

var (
	ErrAgentNotFound = errors.New("client: agent not found")
)

// Agent represents a load-test agent.
// It exposes methods to manage a load-test agent resource through API.
type Agent struct {
	id     string
	apiURL string
	client *http.Client
}

// AgentResponse contains the data returned by the load-test agent API.
type AgentResponse struct {
	Id      string           `json:"id,omitempty"`      // The load-test agent unique identifier.
	Message string           `json:"message,omitempty"` // Message contains information about the response.
	Status  *loadtest.Status `json:"status,omitempty"`  // Status contains the current status of the load test.
	Error   string           `json:"error,omitempty"`   // Error is set if there was an error during the operation.
}

func (a *Agent) apiRequest(req *http.Request) (AgentResponse, error) {
	var res AgentResponse
	resp, err := a.client.Do(req)
	if err != nil {
		return res, fmt.Errorf("agent: post request failed: %w", err)
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return res, fmt.Errorf("agent: failed to decode load-test agent api response: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return res, ErrAgentNotFound
	} else if res.Error != "" {
		return res, fmt.Errorf("agent: load-test agent api request error: %s", res.Error)
	} else if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return res, fmt.Errorf("agent: bad response status code %d", resp.StatusCode)
	}

	return res, nil
}

func (a *Agent) apiGet(url string) (AgentResponse, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return AgentResponse{}, fmt.Errorf("agent: failed to build request: %w", err)
	}
	return a.apiRequest(req)
}

func (a *Agent) apiPost(url string, data []byte) (AgentResponse, error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return AgentResponse{}, fmt.Errorf("agent: failed to build request: %w", err)
	}
	return a.apiRequest(req)
}

// New creates and initializes a new instance of Agent.
// Returns an error in case of failure.
func New(id, serverURL string, client *http.Client) (*Agent, error) {
	if id == "" {
		return nil, errors.New("agent: id should not be empty")
	}
	if serverURL == "" {
		return nil, errors.New("agent: serverURL should not be empty")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Agent{
		id:     id,
		apiURL: serverURL + "/loadagent/",
		client: client,
	}, nil
}

// Id returns the unique identifier for the load-test agent resource.
func (a *Agent) Id() string {
	return a.id
}

// Create creates a new load-test agent resource with the given configs.
// Returns the load-test agent status or an error in case of failure.
func (a *Agent) Create(ltConfig *loadtest.Config, ucConfig interface{}) (loadtest.Status, error) {
	var status loadtest.Status
	if ltConfig == nil {
		return status, errors.New("client: ltConfig should not be nil")
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
		if ucConfig == nil {
			return status, errors.New("client: ucConfig should not be nil")
		}

		var scc *simplecontroller.Config
		scc, ok := ucConfig.(*simplecontroller.Config)
		if !ok {
			return status, errors.New("client: ucConfig has the wrong type")
		}
		data.SimpleControllerConfig = scc
	case loadtest.UserControllerSimulative:
		if ucConfig == nil {
			return status, errors.New("client: ucConfig should not be nil")
		}

		scc, ok := ucConfig.(*simulcontroller.Config)
		if !ok {
			return status, errors.New("client: ucConfig has the wrong type")
		}
		data.SimulControllerConfig = scc
	case loadtest.UserControllerNoop:
	default:
		return status, errors.New("client: UserController type is not set")
	}

	configData, err := json.Marshal(data)
	if err != nil {
		return status, err
	}
	resp, err := a.apiPost(a.apiURL+"create?id="+a.id, configData)
	if err != nil {
		return status, err
	}

	status = *resp.Status
	return status, nil
}

// Status retrieves and returns the status for the load-test agent.
// It also returns an error in case of failure.
func (a *Agent) Status() (loadtest.Status, error) {
	var status loadtest.Status
	resp, err := a.apiGet(a.apiURL + a.id)
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
	resp, err := a.apiPost(a.apiURL+a.id+"/run", nil)

	// create a new endpot '/run-browser' for browser agent
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
	resp, err := a.apiPost(a.apiURL+a.id+"/stop", nil)
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
	resp, err := a.apiPost(a.apiURL+a.id+"/addusers?amount="+strconv.Itoa(numUsers), nil)
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
	resp, err := a.apiPost(a.apiURL+a.id+"/removeusers?amount="+strconv.Itoa(numUsers), nil)
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
	req, err := http.NewRequest("DELETE", a.apiURL+a.id, nil)
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

// InjectAction injects an action that is run once, at the next possible
// opportunity.
func (a *Agent) InjectAction(actionID string) (loadtest.Status, error) {
	var status loadtest.Status
	resp, err := a.apiPost(a.apiURL+a.id+"/inject?action="+actionID, nil)
	if err != nil {
		return status, err
	}
	status = *resp.Status
	return status, nil
}
