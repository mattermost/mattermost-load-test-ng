package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"

	"github.com/mattermost/mattermost-load-test-ng/cmd/loadtest/config"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
)

// API contains information about all load tests.
type API struct {
	agents map[string]*loadtest.LoadTester
}

// APIResponse contains the data returned by the HTTP server.
type APIResponse struct {
	Message string          `json:"message,omitempty"` // Message contains information about the response.
	Status  loadtest.Status `json:"status"`            // Status contains the current status of the load test.
	Error   error           `json:"error,omitempty"`   // Error is set if there was an error during the operation.
}

func writeResponse(w http.ResponseWriter, status int, response *APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func (a *API) createLoadAgentHandler(w http.ResponseWriter, r *http.Request) {
	var config config.LoadTestConfig
	err := json.NewDecoder(r.Body).Decode(&config)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, &APIResponse{
			Error: err,
		})
		return
	}

	if ok, err := config.IsValid(); !ok {
		writeResponse(w, http.StatusBadRequest, &APIResponse{
			Error: err,
		})
		return
	}

	u, err := uuid.NewV4()
	if err != nil {
		writeResponse(w, http.StatusBadRequest, &APIResponse{
			Error: err,
		})
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

	writeResponse(w, http.StatusCreated, &APIResponse{
		Message: fmt.Sprintf("load-test started with agent id: %s", u.String()),
	})
}

func (a *API) getLoadTestById(w http.ResponseWriter, r *http.Request) (*loadtest.LoadTester, error) {
	vars := mux.Vars(r)
	id := vars["id"]
	lt, ok := a.agents[id]
	if !ok {
		err := fmt.Errorf("load-test agent with id %s not found", id)
		writeResponse(w, http.StatusNotFound, &APIResponse{
			Error: err,
		})
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
		writeResponse(w, http.StatusOK, &APIResponse{
			Error:  err,
			Status: lt.Status(),
		})
		return
	}
	writeResponse(w, http.StatusOK, &APIResponse{
		Message: "load-test agent started",
		Status:  lt.Status(),
	})
}

func (a *API) stopLoadAgentHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}
	if err = lt.Stop(); err != nil {
		writeResponse(w, http.StatusOK, &APIResponse{
			Error:  err,
			Status: lt.Status(),
		})
		return
	}
	writeResponse(w, http.StatusOK, &APIResponse{
		Message: "load-test agent stopped",
		Status:  lt.Status(),
	})
}

func (a *API) destroyLoadAgentHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}

	_ = lt.Stop() // we are ignoring the error here in case the load test was previously stopped

	delete(a.agents, mux.Vars(r)["id"])
	writeResponse(w, http.StatusOK, &APIResponse{
		Message: "load-test agent destroyed",
		Status:  lt.Status(),
	})
}

func (a *API) getLoadAgentStatusHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}
	writeResponse(w, http.StatusOK, &APIResponse{
		Status: lt.Status(),
	})
}

func getAmount(r *http.Request) (int, error) {
	amountStr := r.FormValue("amount")
	amount, err := strconv.ParseInt(amountStr, 10, 16)
	return int(amount), err
}

func (a *API) addUserHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}

	amount, err := getAmount(r)
	if amount <= 0 || err != nil {
		writeResponse(w, http.StatusBadRequest, &APIResponse{
			Status: lt.Status(),
			Error:  fmt.Errorf("invalid amount: %s", r.FormValue("amount")),
		})
		return
	}

	i := 0
	var addError error
	for ; i < amount; i++ {
		if err := lt.AddUser(); err != nil {
			addError = err
			break // stop on first error, result is reported as part of status
		}
	}

	writeResponse(w, http.StatusOK, &APIResponse{
		Message: fmt.Sprintf("%d users added", i),
		Status:  lt.Status(),
		Error:   addError,
	})
}

func (a *API) removeUserHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}

	amount, err := getAmount(r)
	if amount <= 0 || err != nil {
		writeResponse(w, http.StatusBadRequest, &APIResponse{
			Status: lt.Status(),
			Error:  fmt.Errorf("invalid amount: %s", r.FormValue("amount")),
		})
		return
	}

	i := 0
	var removeError error
	for ; i < amount; i++ {
		if err = lt.RemoveUser(); err != nil {
			removeError = err
			break // stop on first error, result is reported as part of status
		}
	}

	writeResponse(w, http.StatusOK, &APIResponse{
		Message: fmt.Sprintf("%d users removed", i),
		Status:  lt.Status(),
		Error:   removeError,
	})
}

// SetupAPIRouter creates a router to handle load test API requests.
func SetupAPIRouter() *mux.Router {
	router := mux.NewRouter()
	r := router.PathPrefix("/loadagent").Subrouter()

	agent := API{agents: make(map[string]*loadtest.LoadTester)}
	r.HandleFunc("/create", agent.createLoadAgentHandler).Methods("POST")
	// TODO: add a middleware which refactors getLoadTestById and passes the load test
	// in a request context.
	r.HandleFunc("/{id}/run", agent.runLoadAgentHandler).Methods("POST")
	r.HandleFunc("/{id}/stop", agent.stopLoadAgentHandler).Methods("POST")
	r.HandleFunc("/{id}", agent.destroyLoadAgentHandler).Methods("DELETE")
	r.HandleFunc("/{id}/status", agent.getLoadAgentStatusHandler).Methods("GET")
	r.HandleFunc("/{id}/user/add", agent.addUserHandler).Methods("POST").Queries("amount", "{[0-9]*?}")
	r.HandleFunc("/{id}/user/remove", agent.removeUserHandler).Methods("POST").Queries("amount", "{[0-9]*?}")
	return router
}
