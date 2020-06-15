// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/noopcontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost-load-test-ng/performance"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-server/v5/mlog"
)

// API contains information about all load tests.
type API struct {
	agents  map[string]*loadtest.LoadTester
	metrics *performance.Metrics
}

// Response contains the data returned by the HTTP server.
type Response struct {
	Id      string           `json:"id,omitempty"`      // The load-test agent unique identifier.
	Message string           `json:"message,omitempty"` // Message contains information about the response.
	Status  *loadtest.Status `json:"status,omitempty"`  // Status contains the current status of the load test.
	Error   string           `json:"error,omitempty"`   // Error is set if there was an error during the operation.
}

func writeResponse(w http.ResponseWriter, status int, response *Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func (a *API) createLoadAgentHandler(w http.ResponseWriter, r *http.Request) {
	var data struct {
		LoadTestConfig         loadtest.Config
		SimpleControllerConfig *simplecontroller.Config `json:",omitempty"`
		SimulControllerConfig  *simulcontroller.Config  `json:",omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		writeResponse(w, http.StatusBadRequest, &Response{
			Error: fmt.Sprintf("could not read request: %s", err),
		})
		return
	}
	ltConfig := data.LoadTestConfig
	if err := defaults.Validate(ltConfig); err != nil {
		writeResponse(w, http.StatusBadRequest, &Response{
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
		writeResponse(w, http.StatusBadRequest, &Response{
			Error: fmt.Sprintf("could not read controller configuration: %s", err),
		})
		return
	}
	if ucConfig != nil {
		if err := defaults.Validate(ucConfig); err != nil {
			writeResponse(w, http.StatusBadRequest, &Response{
				Error: fmt.Sprintf("could not validate controller configuration: %s", err),
			})
			return
		}
	}

	agentId := r.FormValue("id")
	if a.agents[agentId] != nil {
		writeResponse(w, http.StatusBadRequest, &Response{
			Error: fmt.Sprintf("load-test agent with id %s already exists", agentId),
		})
		return
	}

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

	newControllerFn := func(id int, status chan<- control.UserStatus) (control.UserController, error) {
		ueConfig := userentity.Config{
			ServerURL:    ltConfig.ConnectionConfiguration.ServerURL,
			WebSocketURL: ltConfig.ConnectionConfiguration.WebSocketURL,
			Username:     fmt.Sprintf("%s-user%d", agentId, id),
			Email:        fmt.Sprintf("%s-user%d@example.com", agentId, id),
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
			Metrics:   a.metrics.UserEntityMetrics(),
		}
		ue := userentity.New(ueSetup, ueConfig)
		switch ltConfig.UserControllerConfiguration.Type {
		case loadtest.UserControllerSimple:
			return simplecontroller.New(id, ue, ucConfig.(*simplecontroller.Config), status)
		case loadtest.UserControllerSimulative:
			return simulcontroller.New(id, ue, ucConfig.(*simulcontroller.Config), status)
		case loadtest.UserControllerNoop:
			return noopcontroller.New(id, ue, status)
		default:
			panic("controller type must be valid")
		}
	}

	lt, err := loadtest.New(&ltConfig, newControllerFn)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, &Response{
			Id:      agentId,
			Message: "load-test agent creation failed",
			Error:   fmt.Sprintf("could not create agent: %s", err),
		})
		return
	}
	a.agents[agentId] = lt

	writeResponse(w, http.StatusCreated, &Response{
		Id:      agentId,
		Message: "load-test agent created",
	})
}

func (a *API) getLoadAgentById(w http.ResponseWriter, r *http.Request) (*loadtest.LoadTester, error) {
	vars := mux.Vars(r)
	id := vars["id"]
	lt, ok := a.agents[id]
	if !ok {
		err := fmt.Errorf("load-test agent with id %s not found", id)
		writeResponse(w, http.StatusNotFound, &Response{
			Error: err.Error(),
		})
		return nil, err
	}
	return lt, nil
}

func (a *API) runLoadAgentHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadAgentById(w, r)
	if err != nil {
		return
	}
	if err = lt.Run(); err != nil {
		writeResponse(w, http.StatusOK, &Response{
			Error: err.Error(),
		})
		return
	}
	writeResponse(w, http.StatusOK, &Response{
		Message: "load-test agent started",
		Status:  lt.Status(),
	})
}

func (a *API) stopLoadAgentHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadAgentById(w, r)
	if err != nil {
		return
	}
	if err = lt.Stop(); err != nil {
		writeResponse(w, http.StatusOK, &Response{
			Error: err.Error(),
		})
		return
	}
	writeResponse(w, http.StatusOK, &Response{
		Message: "load-test agent stopped",
		Status:  lt.Status(),
	})
}

func (a *API) destroyLoadAgentHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadAgentById(w, r)
	if err != nil {
		return
	}

	_ = lt.Stop() // we are ignoring the error here in case the load test was previously stopped

	delete(a.agents, mux.Vars(r)["id"])
	writeResponse(w, http.StatusOK, &Response{
		Message: "load-test agent destroyed",
		Status:  lt.Status(),
	})
}

func (a *API) getLoadAgentStatusHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadAgentById(w, r)
	if err != nil {
		return
	}
	writeResponse(w, http.StatusOK, &Response{
		Status: lt.Status(),
	})
}

func getAmount(r *http.Request) (int, error) {
	amountStr := r.FormValue("amount")
	amount, err := strconv.ParseInt(amountStr, 10, 16)
	return int(amount), err
}

func (a *API) addUsersHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadAgentById(w, r)
	if err != nil {
		return
	}

	amount, err := getAmount(r)
	if amount <= 0 || err != nil {
		writeResponse(w, http.StatusBadRequest, &Response{
			Error: fmt.Sprintf("invalid amount: %s", r.FormValue("amount")),
		})
		return
	}

	var res Response
	n, err := lt.AddUsers(amount)
	if err != nil {
		res.Error = err.Error()
	}
	res.Message = fmt.Sprintf("%d users added", n)
	res.Status = lt.Status()
	writeResponse(w, http.StatusOK, &res)
}

func (a *API) removeUsersHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadAgentById(w, r)
	if err != nil {
		return
	}

	amount, err := getAmount(r)
	if amount <= 0 || err != nil {
		writeResponse(w, http.StatusBadRequest, &Response{
			Error: fmt.Sprintf("invalid amount: %s", r.FormValue("amount")),
		})
		return
	}

	var res Response
	n, err := lt.RemoveUsers(amount)
	if err != nil {
		res.Error = err.Error()
	}

	res.Message = fmt.Sprintf("%d users removed", n)
	res.Status = lt.Status()
	writeResponse(w, http.StatusOK, &res)
}

func (a *API) pprofIndexHandler(w http.ResponseWriter, r *http.Request) {
	html := `
		<html>
			<body>
				<div><a href="/debug/pprof/">Profiling Root</a></div>
				<div><a href="/debug/pprof/heap">Heap profile</a></div>
				<div><a href="/debug/pprof/profile">CPU profile</a></div>
				<div><a href="/debug/pprof/trace">Trace profile</a></div>
			</body>
		</html>
	`

	w.Write([]byte(html))
}

// SetupAPIRouter creates a router to handle load test API requests.
func SetupAPIRouter() *mux.Router {
	router := mux.NewRouter()
	r := router.PathPrefix("/loadagent").Subrouter()

	api := API{
		agents:  make(map[string]*loadtest.LoadTester),
		metrics: performance.NewMetrics(),
	}
	r.HandleFunc("/create", api.createLoadAgentHandler).Methods("POST").Queries("id", "{^[a-z]+[0-9]*$}")
	r.HandleFunc("/{id}/run", api.runLoadAgentHandler).Methods("POST")
	r.HandleFunc("/{id}/stop", api.stopLoadAgentHandler).Methods("POST")
	r.HandleFunc("/{id}", api.destroyLoadAgentHandler).Methods("DELETE")
	r.HandleFunc("/{id}", api.getLoadAgentStatusHandler).Methods("GET")
	r.HandleFunc("/{id}/status", api.getLoadAgentStatusHandler).Methods("GET")
	r.HandleFunc("/{id}/addusers", api.addUsersHandler).Methods("POST").Queries("amount", "{[0-9]*?}")
	r.HandleFunc("/{id}/removeusers", api.removeUsersHandler).Methods("POST").Queries("amount", "{[0-9]*?}")

	// Add profile endpoints
	p := router.PathPrefix("/debug/pprof").Subrouter()
	p.HandleFunc("/", api.pprofIndexHandler).Methods("GET")
	p.Handle("/heap", pprof.Handler("heap")).Methods("GET")
	p.HandleFunc("/profile", pprof.Profile).Methods("GET")
	p.HandleFunc("/trace", pprof.Trace).Methods("GET")

	// Add metrics endpoint
	router.Handle("/metrics", api.metrics.Handler())

	return router
}
