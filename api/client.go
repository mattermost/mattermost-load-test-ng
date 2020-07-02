// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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
