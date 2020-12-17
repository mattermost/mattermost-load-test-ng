// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	client "github.com/mattermost/mattermost-load-test-ng/api/client/agent"
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

func writeAgentResponse(w http.ResponseWriter, status int, resp *client.AgentResponse) {
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
		writeAgentResponse(w, http.StatusBadRequest, &client.AgentResponse{
			Error: fmt.Sprintf("could not read request: %s", err),
		})
		return
	}
	ltConfig := data.LoadTestConfig
	if err := defaults.Validate(ltConfig); err != nil {
		writeAgentResponse(w, http.StatusBadRequest, &client.AgentResponse{
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
		writeAgentResponse(w, http.StatusBadRequest, &client.AgentResponse{
			Error: fmt.Sprintf("could not read controller configuration: %s", err),
		})
		return
	}
	if ucConfig != nil {
		if err := defaults.Validate(ucConfig); err != nil {
			writeAgentResponse(w, http.StatusBadRequest, &client.AgentResponse{
				Error: fmt.Sprintf("could not validate controller configuration: %s", err),
			})
			return
		}
	}

	agentId := r.FormValue("id")
	if val, ok := a.getResource(agentId); ok && val != nil {
		if _, ok := val.(*loadtest.LoadTester); ok {
			writeAgentResponse(w, http.StatusConflict, &client.AgentResponse{
				Error: fmt.Sprintf("load-test agent with id %s already exists", agentId),
			})
		} else {
			writeAgentResponse(w, http.StatusConflict, &client.AgentResponse{
				Error: fmt.Sprintf("resource with id %s already exists", agentId),
			})
		}
		return
	}

	newC, err := NewControllerWrapper(&ltConfig, ucConfig, 0, agentId, a.metrics)
	if err != nil {
		writeAgentResponse(w, http.StatusBadRequest, &client.AgentResponse{
			Id:      agentId,
			Message: "load-test agent creation failed",
			Error:   fmt.Sprintf("could not create agent: %s", err),
		})
	}
	lt, err := loadtest.New(&ltConfig, newC, a.agentLog)
	if err != nil {
		writeAgentResponse(w, http.StatusBadRequest, &client.AgentResponse{
			Id:      agentId,
			Message: "load-test agent creation failed",
			Error:   fmt.Sprintf("could not create agent: %s", err),
		})
		return
	}
	if ok := a.setResource(agentId, lt); !ok {
		writeAgentResponse(w, http.StatusConflict, &client.AgentResponse{
			Error: fmt.Sprintf("resource with id %s already exists", agentId),
		})
		return
	}

	writeAgentResponse(w, http.StatusCreated, &client.AgentResponse{
		Id:      agentId,
		Message: "load-test agent created",
		Status:  lt.Status(),
	})
}

func (a *api) getLoadAgentById(w http.ResponseWriter, r *http.Request) (*loadtest.LoadTester, error) {
	vars := mux.Vars(r)
	id := vars["id"]

	val, ok := a.getResource(id)
	if !ok || val == nil {
		err := fmt.Errorf("load-test agent with id %s not found", id)
		writeAgentResponse(w, http.StatusNotFound, &client.AgentResponse{
			Error: err.Error(),
		})
		return nil, err
	}

	lt, ok := val.(*loadtest.LoadTester)
	if !ok {
		err := fmt.Errorf("resource with id %s is not a load-test agent", id)
		writeAgentResponse(w, http.StatusBadRequest, &client.AgentResponse{
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
		writeAgentResponse(w, http.StatusOK, &client.AgentResponse{
			Error: err.Error(),
		})
		return
	}
	writeAgentResponse(w, http.StatusOK, &client.AgentResponse{
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
		writeAgentResponse(w, http.StatusOK, &client.AgentResponse{
			Error: err.Error(),
		})
		return
	}
	writeAgentResponse(w, http.StatusOK, &client.AgentResponse{
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

	id := mux.Vars(r)["id"]
	if ok := a.deleteResource(id); !ok {
		writeAgentResponse(w, http.StatusNotFound, &client.AgentResponse{
			Error: fmt.Sprintf("load-test agent with id %s not found", id),
		})
		return
	}
	writeAgentResponse(w, http.StatusOK, &client.AgentResponse{
		Message: "load-test agent destroyed",
		Status:  lt.Status(),
	})
}

func (a *api) getLoadAgentStatusHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadAgentById(w, r)
	if err != nil {
		return
	}
	writeAgentResponse(w, http.StatusOK, &client.AgentResponse{
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
		writeAgentResponse(w, http.StatusBadRequest, &client.AgentResponse{
			Error: fmt.Sprintf("invalid amount: %s", r.FormValue("amount")),
		})
		return
	}

	var resp client.AgentResponse
	n, err := lt.AddUsers(amount)
	if err != nil {
		resp.Error = err.Error()
	}
	resp.Message = fmt.Sprintf("%d users added", n)
	resp.Status = lt.Status()
	writeAgentResponse(w, http.StatusOK, &resp)
}

func (a *api) removeUsersHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadAgentById(w, r)
	if err != nil {
		return
	}

	amount, err := getAmount(r)
	if amount <= 0 || err != nil {
		writeAgentResponse(w, http.StatusBadRequest, &client.AgentResponse{
			Error: fmt.Sprintf("invalid amount: %s", r.FormValue("amount")),
		})
		return
	}

	var resp client.AgentResponse
	n, err := lt.RemoveUsers(amount)
	if err != nil {
		resp.Error = err.Error()
	}

	resp.Message = fmt.Sprintf("%d users removed", n)
	resp.Status = lt.Status()
	writeAgentResponse(w, http.StatusOK, &resp)
}

func getServerVersion(serverURL string) (string, error) {
	var version string
	resp, err := http.Get(serverURL)
	if err != nil {
		return version, fmt.Errorf("failed to get server version: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)

	header := resp.Header["X-Version-Id"]
	if len(header) > 0 {
		version = header[0]
	}

	if version == "" {
		return version, errors.New("server version is empty")
	}

	return version, nil
}

// NewControllerWrapper returns a constructor function used to create
// a new UserController.
func NewControllerWrapper(config *loadtest.Config, controllerConfig interface{}, userOffset int, namePrefix string, metrics *performance.Metrics) (loadtest.NewController, error) {
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

	var err error
	serverVersion := config.UserControllerConfiguration.ServerVersion
	if serverVersion == "" {
		serverVersion, err = getServerVersion(config.ConnectionConfiguration.ServerURL)
		if err != nil {
			mlog.Error("Failed to get server version", mlog.Err(err))
		}
	}

	creds, err := getUserCredentials(config.UsersConfiguration.UsersFilePath, config)
	if err != nil {
		return nil, err
	}

	return func(id int, status chan<- control.UserStatus) (control.UserController, error) {
		id += userOffset

		var username, password, email string

		if len(creds) > 0 {
			// The bounds check is already done during creation of the creds slice.
			username = creds[id].username
			email = creds[id].email
			password = creds[id].password
		} else {
			username = fmt.Sprintf("%s-%d", namePrefix, id)
			email = fmt.Sprintf("%s-%d@example.com", namePrefix, id)
			password = "testPass123$"
		}

		ueConfig := userentity.Config{
			ServerURL:    config.ConnectionConfiguration.ServerURL,
			WebSocketURL: config.ConnectionConfiguration.WebSocketURL,
			Username:     username,
			Email:        email,
			Password:     password,
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

		if err := store.SetServerVersion(serverVersion); err != nil {
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
	}, nil
}

type user struct {
	email    string
	username string
	password string
}

func getUserCredentials(usersFilePath string, config *loadtest.Config) ([]user, error) {
	var users []user
	if usersFilePath == "" {
		return users, nil
	}
	f, err := os.Open(usersFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %q: %w", usersFilePath, err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Emails and passwords are separated by space.
		split := strings.Split(line, " ")
		if len(split) < 2 {
			return nil, fmt.Errorf("user credential %q does not have space in between", line)
		}
		email := split[0]
		password := split[1]
		// Quick and dirty hack to extract username from email.
		// This is not terribly important to be correct.
		username := strings.Split(email, "@")[0]
		username = strings.Replace(username, "+", "-", -1)

		users = append(users, user{
			email:    email,
			username: username,
			password: password,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read from %s: %w", f.Name(), err)
	}
	// We do not allow to mix user-defined and auto signup modes.
	// If the user specifies a custom login file, then it should contain
	// all the users that is expected to login during a load-test.
	if len(users) < config.UsersConfiguration.MaxActiveUsers+1 {
		return nil, fmt.Errorf("number of lines in %q is %d, which is less than MaxActiveUsers+1(%d)", usersFilePath, len(users), config.UsersConfiguration.MaxActiveUsers+1)
	}

	return users, nil
}
