package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
)

// remove once https://github.com/mattermost/mattermost-load-test-ng/pull/87 is merged
type MockStatus struct {
	State           loadtest.State // State of the the load test.
	NumUsers        int            // Number of active users.
	NumUsersAdded   int            // Number of users added since the start of the test.
	NumUsersRemoved int            // Number of users removed since the start of the test.
	NumErrors       int32          // Number of errors that have occurred.
	StartTime       time.Time      // Time when the load test was started. This only logs the time when the load test was first started, and does not get reset if it was subsequently restarted.
}

type API struct {
	agents map[string]*loadtest.LoadTester
}

func writeJsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func (a *API) createLoadAgentHandler(w http.ResponseWriter, r *http.Request) {
	var config config.LoadTestConfig
	err := json.NewDecoder(r.Body).Decode(&config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	u, err := uuid.NewV4()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newSimpleController := func(id int, status chan<- control.UserStatus) control.UserController {
		ueConfig := userentity.Config{
			ServerURL:    config.ConnectionConfiguration.ServerURL,
			WebSocketURL: config.ConnectionConfiguration.WebSocketURL,
		}
		ue := userentity.New(memstore.New(), ueConfig)
		return simplecontroller.New(id, ue, status)
	}

	lt := loadtest.New(&config, newSimpleController)
	a.agents[u.String()] = lt

	writeJsonResponse(w, map[string]string{"loadAgentId": u.String()})
}

func (a *API) getLoadTestById(w http.ResponseWriter, r *http.Request) (*loadtest.LoadTester, error) {
	vars := mux.Vars(r)
	id := vars["id"]
	lt, ok := a.agents[id]
	if !ok {
		err := fmt.Errorf("Load-test agent with id %s not found", id)
		http.Error(w, err.Error(), http.StatusNotFound)
		return nil, err
	}
	return lt, nil
}

func (a *API) runLoadAgentHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}
	if err = lt.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJsonResponse(w, map[string]interface{}{"message": "Load-test agent started", "status": MockStatus{}})
}

func (a *API) stopLoadAgentHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}
	if err = lt.Stop(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJsonResponse(w, map[string]interface{}{"message": "Load-test agent stopped", "status": MockStatus{}})
}

func (a *API) destroyLoadTestHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}

	_ = lt.Stop() // we are ignoring the error here in case the load test was previously stopped

	delete(a.agents, mux.Vars(r)["id"])
	writeJsonResponse(w, map[string]string{"message": "Load-test agent destroyed"})
}

func (a *API) getLoadAgentStatusHandler(w http.ResponseWriter, r *http.Request) {
	_, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]MockStatus{"status": {}})
}

func getAmount(r *http.Request) int {
	amountStr := r.FormValue("amount")
	amount := 1

	if amountStr != "" {
		if a, err := strconv.ParseInt(amountStr, 10, 16); err == nil && a > 0 {
			amount = int(a)
		}
	}
	return amount
}

func (a *API) addUserHandler(w http.ResponseWriter, r *http.Request) {
	amount := getAmount(r)
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}

	i := 0
	var addError string
	for ; i < amount; i++ {
		if err := lt.AddUser(); err != nil {
			addError = err.Error()
			break // stop on first error, result is reported as part of status
		}
	}

	writeJsonResponse(w, map[string]interface{}{"message": fmt.Sprintf("%d users added", i+1), "error": addError, "status": MockStatus{}})
}

func (a *API) removeUserHandler(w http.ResponseWriter, r *http.Request) {
	amount := getAmount(r)

	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}

	i := 0
	var removeError string
	for ; i < amount; i++ {
		if err = lt.RemoveUser(); err != nil {
			removeError = err.Error()
			break // stop on first error, result is reported as part of status
		}
	}

	writeJsonResponse(w, map[string]interface{}{"message": fmt.Sprintf("%d users removed", i+1), "error": removeError, "status": MockStatus{}})
}

func SetupAPIRouter() *mux.Router {
	router := mux.NewRouter()
	r := router.PathPrefix("/loadagent").Subrouter()

	agent := API{agents: make(map[string]*loadtest.LoadTester)}
	r.HandleFunc("/create", agent.createLoadAgentHandler).Methods("POST")
	r.HandleFunc("/{id}/run", agent.runLoadAgentHandler).Methods("POST")
	r.HandleFunc("/{id}/stop", agent.stopLoadAgentHandler).Methods("POST")
	r.HandleFunc("/{id}", agent.destroyLoadTestHandler).Methods("DELETE")
	r.HandleFunc("/{id}/status", agent.getLoadAgentStatusHandler).Methods("GET")
	r.HandleFunc("/{id}/user/add", agent.addUserHandler).Methods("POST").Queries("amount", "{[0-9]*?}")
	r.HandleFunc("/{id}/user/remove", agent.removeUserHandler).Methods("POST").Queries("amount", "{[0-9]*?}")
	return router
}
