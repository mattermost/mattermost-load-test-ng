// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/clustercontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/gencontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/noopcontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost-load-test-ng/performance"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/v5/mlog"
)

// AgentResponse contains the data returned by load-test agent API.
type AgentResponse struct {
	Id      string           `json:"id,omitempty"`      // The load-test agent unique identifier.
	Message string           `json:"message,omitempty"` // Message contains information about the response.
	Status  *loadtest.Status `json:"status,omitempty"`  // Status contains the current status of the load test.
	Error   string           `json:"error,omitempty"`   // Error is set if there was an error during the operation.
}

func writeResponse(w http.ResponseWriter, status int, resp *AgentResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func getAmount(r *http.Request) (int, error) {
	amountStr := r.FormValue("amount")
	amount, err := strconv.ParseInt(amountStr, 10, 16)
	return int(amount), err
}

func (a *api) createLoadAgentHandler(w http.ResponseWriter, r *http.Request) {
	var data struct {
		LoadTestConfig         loadtest.Config
		SimpleControllerConfig *simplecontroller.Config `json:",omitempty"`
		SimulControllerConfig  *simulcontroller.Config  `json:",omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		writeResponse(w, http.StatusBadRequest, &AgentResponse{
			Error: fmt.Sprintf("could not read request: %s", err),
		})
		return
	}
	ltConfig := data.LoadTestConfig
	if err := defaults.Validate(ltConfig); err != nil {
		writeResponse(w, http.StatusBadRequest, &AgentResponse{
			Error: fmt.Sprintf("could not validate config: %s", err),
		})
		return
	}

	var ucConfig interface{}
	var err error
	switch ltConfig.UserControllerConfiguration.Type {
	case loadtest.UserControllerSimple:
		if data.SimpleControllerConfig == nil {
			mlog.Warn("could not read controller config from the request")
			ucConfig, err = simplecontroller.ReadConfig("")
			break
		}
		ucConfig = data.SimpleControllerConfig
	case loadtest.UserControllerSimulative:
		if data.SimulControllerConfig == nil {
			mlog.Warn("could not read controller config from the request")
			ucConfig, err = simulcontroller.ReadConfig("")
			break
		}
		ucConfig = data.SimulControllerConfig
	}
	if err != nil {
		writeResponse(w, http.StatusBadRequest, &AgentResponse{
			Error: fmt.Sprintf("could not read controller configuration: %s", err),
		})
		return
	}
	if ucConfig != nil {
		if err := defaults.Validate(ucConfig); err != nil {
			writeResponse(w, http.StatusBadRequest, &AgentResponse{
				Error: fmt.Sprintf("could not validate controller configuration: %s", err),
			})
			return
		}
	}

	agentId := r.FormValue("id")
	if a.agents[agentId] != nil {
		writeResponse(w, http.StatusBadRequest, &AgentResponse{
			Error: fmt.Sprintf("load-test agent with id %s already exists", agentId),
		})
		return
	}

	lt, err := loadtest.New(&ltConfig, NewControllerWrapper(&ltConfig, ucConfig, 0, agentId, a.metrics))
	if err != nil {
		writeResponse(w, http.StatusBadRequest, &AgentResponse{
			Id:      agentId,
			Message: "load-test agent creation failed",
			Error:   fmt.Sprintf("could not create agent: %s", err),
		})
		return
	}
	a.agents[agentId] = lt

	writeResponse(w, http.StatusCreated, &AgentResponse{
		Id:      agentId,
		Message: "load-test agent created",
	})
}

func (a *api) getLoadAgentById(w http.ResponseWriter, r *http.Request) (*loadtest.LoadTester, error) {
	vars := mux.Vars(r)
	id := vars["id"]
	lt, ok := a.agents[id]
	if !ok {
		err := fmt.Errorf("load-test agent with id %s not found", id)
		writeResponse(w, http.StatusNotFound, &AgentResponse{
			Error: err.Error(),
		})
		return nil, err
	}
	return lt, nil
}

func (a *api) runLoadAgentHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadAgentById(w, r)
	if err != nil {
		return
	}
	if err = lt.Run(); err != nil {
		writeResponse(w, http.StatusOK, &AgentResponse{
			Error: err.Error(),
		})
		return
	}
	writeResponse(w, http.StatusOK, &AgentResponse{
		Message: "load-test agent started",
		Status:  lt.Status(),
	})
}

func (a *api) stopLoadAgentHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadAgentById(w, r)
	if err != nil {
		return
	}
	if err = lt.Stop(); err != nil {
		writeResponse(w, http.StatusOK, &AgentResponse{
			Error: err.Error(),
		})
		return
	}
	writeResponse(w, http.StatusOK, &AgentResponse{
		Message: "load-test agent stopped",
		Status:  lt.Status(),
	})
}

func (a *api) destroyLoadAgentHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadAgentById(w, r)
	if err != nil {
		return
	}

	_ = lt.Stop() // we are ignoring the error here in case the load test was previously stopped

	delete(a.agents, mux.Vars(r)["id"])
	writeResponse(w, http.StatusOK, &AgentResponse{
		Message: "load-test agent destroyed",
		Status:  lt.Status(),
	})
}

func (a *api) getLoadAgentStatusHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadAgentById(w, r)
	if err != nil {
		return
	}
	writeResponse(w, http.StatusOK, &AgentResponse{
		Status: lt.Status(),
	})
}

func (a *api) addUsersHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadAgentById(w, r)
	if err != nil {
		return
	}

	amount, err := getAmount(r)
	if amount <= 0 || err != nil {
		writeResponse(w, http.StatusBadRequest, &AgentResponse{
			Error: fmt.Sprintf("invalid amount: %s", r.FormValue("amount")),
		})
		return
	}

	var resp AgentResponse
	n, err := lt.AddUsers(amount)
	if err != nil {
		resp.Error = err.Error()
	}
	resp.Message = fmt.Sprintf("%d users added", n)
	resp.Status = lt.Status()
	writeResponse(w, http.StatusOK, &resp)
}

func (a *api) removeUsersHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadAgentById(w, r)
	if err != nil {
		return
	}

	amount, err := getAmount(r)
	if amount <= 0 || err != nil {
		writeResponse(w, http.StatusBadRequest, &AgentResponse{
			Error: fmt.Sprintf("invalid amount: %s", r.FormValue("amount")),
		})
		return
	}

	var resp AgentResponse
	n, err := lt.RemoveUsers(amount)
	if err != nil {
		resp.Error = err.Error()
	}

	resp.Message = fmt.Sprintf("%d users removed", n)
	resp.Status = lt.Status()
	writeResponse(w, http.StatusOK, &resp)
}

// NewControllerWrapper returns a constructor function used to create
// a new UserController.
func NewControllerWrapper(config *loadtest.Config, controllerConfig interface{}, userOffset int, namePrefix string, metrics *performance.Metrics) loadtest.NewController {
	// http.Transport to be shared amongst all clients.
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxConnsPerHost:       500,
		MaxIdleConns:          500,
		MaxIdleConnsPerHost:   500,
		ResponseHeaderTimeout: 5 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   1 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return func(id int, status chan<- control.UserStatus) (control.UserController, error) {
		id += userOffset

		ueConfig := userentity.Config{
			ServerURL:    config.ConnectionConfiguration.ServerURL,
			WebSocketURL: config.ConnectionConfiguration.WebSocketURL,
			Username:     fmt.Sprintf("%s-%d", namePrefix, id),
			Email:        fmt.Sprintf("%s-%d@example.com", namePrefix, id),
			Password:     "testPass123$",
		}
		store, err := memstore.New(&memstore.Config{
			MaxStoredPosts:          500,
			MaxStoredUsers:          1000,
			MaxStoredChannelMembers: 1000,
			MaxStoredStatuses:       1000,
		})
		if err != nil {
			return nil, err
		}
		ueSetup := userentity.Setup{
			Store:     store,
			Transport: transport,
		}
		if metrics != nil {
			ueSetup.Metrics = metrics.UserEntityMetrics()
		}
		ue := userentity.New(ueSetup, ueConfig)

		switch config.UserControllerConfiguration.Type {
		case loadtest.UserControllerSimple:
			return simplecontroller.New(id, ue, controllerConfig.(*simplecontroller.Config), status)
		case loadtest.UserControllerSimulative:
			return simulcontroller.New(id, ue, controllerConfig.(*simulcontroller.Config), status)
		case loadtest.UserControllerGenerative:
			return gencontroller.New(id, ue, controllerConfig.(*gencontroller.Config), status)
		case loadtest.UserControllerNoop:
			return noopcontroller.New(id, ue, status)
		case loadtest.UserControllerCluster:
			// For cluster controller, we only use the sysadmin
			// because we are just testing system console APIs.
			ueConfig.Username = ""
			ueConfig.Email = config.ConnectionConfiguration.AdminEmail
			ueConfig.Password = config.ConnectionConfiguration.AdminPassword

			admin := userentity.New(ueSetup, ueConfig)
			return clustercontroller.New(id, admin, status)
		default:
			panic("controller type must be valid")
		}
	}
}
