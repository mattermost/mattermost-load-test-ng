package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost-server/mlog"
	"github.com/spf13/cobra"
)

type API struct {
	loadTests map[string]*loadtest.LoadTester
}

func writeJsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func (a *API) createLoadTestHandler(w http.ResponseWriter, r *http.Request) {
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
	a.loadTests[u.String()] = lt

	writeJsonResponse(w, map[string]string{"loadTestId": u.String()})
}

func (a *API) getLoadTestById(w http.ResponseWriter, r *http.Request) (*loadtest.LoadTester, error) {
	vars := mux.Vars(r)
	id := vars["id"]
	lt, ok := a.loadTests[id]
	if !ok {
		err := fmt.Errorf("Load test with id %s not found", id)
		http.Error(w, err.Error(), http.StatusNotFound)
		return nil, err
	}
	return lt, nil
}

func (a *API) runLoadTestHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}
	if err = lt.Run(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJsonResponse(w, map[string]string{"message": "load test started"})
}

func (a *API) stopLoadTestHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}
	if err = lt.Stop(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJsonResponse(w, map[string]string{"message": "load test stopped"})
}

func (a *API) destroyLoadTestHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}
	if err = lt.Stop(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	delete(a.loadTests, mux.Vars(r)["id"])
	writeJsonResponse(w, map[string]string{"message": "load test destroyed"})
}

func (a *API) getLoadTestStatusHandler(w http.ResponseWriter, r *http.Request) {
	_, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "Not yet implemented"})
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

	for i := 0; i < amount; i++ {
		if err = lt.AddUser(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	writeJsonResponse(w, map[string]string{"message": "user added"})
}

func (a *API) removeUserHandler(w http.ResponseWriter, r *http.Request) {
	amount := getAmount(r)

	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}

	for i := 0; i < amount; i++ {
		if err = lt.RemoveUser(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	writeJsonResponse(w, map[string]string{"message": "user removed"})
}

func RunServerCmdF(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")

	router := mux.NewRouter()
	r := router.PathPrefix("/loadtest").Subrouter()

	agent := API{loadTests: make(map[string]*loadtest.LoadTester)}
	r.HandleFunc("/create", agent.createLoadTestHandler).Methods("POST")
	r.HandleFunc("/run/{id}", agent.runLoadTestHandler).Methods("POST")
	r.HandleFunc("/stop/{id}", agent.stopLoadTestHandler).Methods("POST")
	r.HandleFunc("/destroy/{id}", agent.destroyLoadTestHandler).Methods("POST")
	r.HandleFunc("/status/{id}", agent.getLoadTestStatusHandler).Methods("GET")
	r.HandleFunc("/user/{id}", agent.addUserHandler).Methods("PUT").Queries("amount", "{[0-9]*?}")
	r.HandleFunc("/user/{id}", agent.removeUserHandler).Methods("DELETE").Queries("amount", "{[0-9]*?}")

	mlog.Info("Agent started, listening on", mlog.Int("port", port))
	return http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), r)
}

func MakeServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "server",
		Short:  "Start load-test agent",
		RunE:   RunServerCmdF,
		PreRun: initLogger,
	}
	cmd.PersistentFlags().IntP("port", "p", 4000, "Port to listen on")

	return cmd
}
