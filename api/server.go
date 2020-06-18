// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"net/http"
	"net/http/pprof"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/performance"

	"github.com/gorilla/mux"
)

// API keeps track of the load-test API server state.
type API struct {
	agents  map[string]*loadtest.LoadTester
	metrics *performance.Metrics
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
