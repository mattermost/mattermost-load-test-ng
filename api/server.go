// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/json"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"strconv"
	"sync"

	"github.com/mattermost/mattermost-load-test-ng/performance"
	"github.com/mattermost/mattermost-load-test-ng/version"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// api keeps track of the load-test API server state.
type api struct {
	mut       sync.RWMutex
	resources map[string]interface{}
	metrics   *performance.Metrics
	coordLog  *mlog.Logger
	agentLog  *mlog.Logger
}

func (a *api) getResource(id string) (interface{}, bool) {
	a.mut.RLock()
	defer a.mut.RUnlock()
	val, ok := a.resources[id]
	return val, ok
}

func (a *api) setResource(id string, res interface{}) bool {
	a.mut.Lock()
	defer a.mut.Unlock()
	if _, ok := a.resources[id]; !ok {
		a.resources[id] = res
		return true
	}
	return false
}

func (a *api) deleteResource(id string) bool {
	a.mut.Lock()
	defer a.mut.Unlock()
	if _, ok := a.resources[id]; ok {
		delete(a.resources, id)
		return true
	}
	return false
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(version.GetInfo()); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (a *api) pprofIndexHandler(w http.ResponseWriter, r *http.Request) {
	html := `
		<html>
			<body>
				<div><a href="/debug/pprof/">Profiling Root</a></div>
				<div><a href="/debug/pprof/heap">Heap profile</a></div>
				<div><a href="/debug/pprof/allocs">Allocs profile</a></div>
				<div><a href="/debug/pprof/profile">CPU profile</a></div>
				<div><a href="/debug/pprof/goroutine">Goroutine profile</a></div>
				<div><a href="/debug/pprof/block">Block profile</a></div>
				<div><a href="/debug/pprof/trace">Trace profile</a></div>
			</body>
		</html>
	`
	w.Write([]byte(html))
}

// SetupAPIRouter creates a router to handle load test API requests.
// Custom loggers for coordinator and agent are given.
func SetupAPIRouter(coordLog, agentLog *mlog.Logger) *mux.Router {
	a := api{
		resources: make(map[string]interface{}),
		metrics:   performance.NewMetrics(),
		coordLog:  coordLog,
		agentLog:  agentLog,
	}

	router := mux.NewRouter()

	// load-test agent API.
	r := router.PathPrefix("/loadagent").Subrouter()
	r.HandleFunc("/create", a.createLoadAgentHandler).Methods("POST").Queries("id", "{^[a-z]+[0-9]*$}")
	r.HandleFunc("/{id}/run", a.runLoadAgentHandler).Methods("POST")
	r.HandleFunc("/{id}/stop", a.stopLoadAgentHandler).Methods("POST")
	r.HandleFunc("/{id}", a.destroyLoadAgentHandler).Methods("DELETE")
	r.HandleFunc("/{id}", a.getLoadAgentStatusHandler).Methods("GET")
	r.HandleFunc("/{id}/status", a.getLoadAgentStatusHandler).Methods("GET")
	r.HandleFunc("/{id}/addusers", a.addUsersHandler).Methods("POST").Queries("amount", "{[0-9]*?}")
	r.HandleFunc("/{id}/removeusers", a.removeUsersHandler).Methods("POST").Queries("amount", "{[0-9]*?}")
	r.HandleFunc("/{id}/inject", a.agentInjectActionHandler).Methods("POST").Queries("action", "{[a-zA-Z]+}")

	// load-test coordinator API.
	c := router.PathPrefix("/coordinator").Subrouter()
	c.HandleFunc("/create", a.createCoordinatorHandler).Methods("POST").Queries("id", "{^[a-z]+[0-9]*$}")
	c.HandleFunc("/{id}", a.destroyCoordinatorHandler).Methods("DELETE")
	c.HandleFunc("/{id}", a.getCoordinatorStatusHandler).Methods("GET")
	c.HandleFunc("/{id}/status", a.getCoordinatorStatusHandler).Methods("GET")
	c.HandleFunc("/{id}/run", a.runCoordinatorHandler).Methods("POST")
	c.HandleFunc("/{id}/stop", a.stopCoordinatorHandler).Methods("POST")
	c.HandleFunc("/{id}/inject", a.coordinatorInjectActionHandler).Methods("POST").Queries("action", "{[a-zA-Z]+}")

	if val, err := strconv.Atoi(os.Getenv("BLOCK_PROFILE_RATE")); err == nil && val > 0 {
		agentLog.Info("setting block profile rate", mlog.Int("value", val))
		runtime.SetBlockProfileRate(val)
	}
	// Debug endpoint.
	p := router.PathPrefix("/debug/pprof").Subrouter()
	p.HandleFunc("/", a.pprofIndexHandler).Methods("GET")
	p.Handle("/heap", pprof.Handler("heap")).Methods("GET")
	p.Handle("/allocs", pprof.Handler("heap")).Methods("GET")
	p.Handle("/goroutine", pprof.Handler("goroutine")).Methods("GET")
	p.Handle("/block", pprof.Handler("block")).Methods("GET")
	p.HandleFunc("/profile", pprof.Profile).Methods("GET")
	p.HandleFunc("/trace", pprof.Trace).Methods("GET")

	// Metrics endpoint.
	router.Handle("/metrics", a.metrics.Handler())

	// Version endpoint
	router.HandleFunc("/version", handleVersion).Methods("GET")

	return router
}
