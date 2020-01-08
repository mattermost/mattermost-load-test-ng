package main

import (
	"encoding/json"
	"fmt"
	"net/http"

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

type Agent struct {
	loadTests map[string]*loadtest.LoadTester
}

func (a *Agent) createLoadTestHandler(w http.ResponseWriter, r *http.Request) {
	var config config.LoadTestConfig
	err := json.NewDecoder(r.Body).Decode(&config)
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
	a.loadTests[lt.Id] = lt
	// start := time.Now()
	// err = lt.Run()
	// if err != nil {
	// 	return err
	// }

	// mlog.Info("loadtest started")
	// time.Sleep(60 * time.Second)

	// err = lt.Stop()
	// mlog.Info("loadtest done", mlog.String("elapsed", time.Since(start).String()))

	// return err

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"loadTestId": lt.Id})

}

func (a *Agent) getLoadTestById(w http.ResponseWriter, r *http.Request) (*loadtest.LoadTester, error) {
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
func (a *Agent) runLoadTestHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}
	go lt.Run()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "load test started"})
}

func (a *Agent) stopLoadTestHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}
	_ = lt.Stop()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "load test stopped"})
}

func (a *Agent) destroyLoadTestHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}
	_ = lt.Stop()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "load test destroyed"})
	delete(a.loadTests, lt.Id)
}

func (a *Agent) getLoadTestStatusHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]loadtest.State{"status": lt.GetState()})
}

func (a *Agent) addUserHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}
	lt.AddUser()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "user added"})

}

func (a *Agent) removeUserHandler(w http.ResponseWriter, r *http.Request) {
	lt, err := a.getLoadTestById(w, r)
	if err != nil {
		return
	}
	lt.RemoveUser()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "user removed"})
}

func RunServerCmdF(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")

	router := mux.NewRouter()
	r := router.PathPrefix("/loadtest").Subrouter()

	agent := Agent{loadTests: make(map[string]*loadtest.LoadTester)}
	r.HandleFunc("/create", agent.createLoadTestHandler).Methods("POST")
	r.HandleFunc("/run/{id}", agent.runLoadTestHandler).Methods("POST")
	r.HandleFunc("/stop/{id}", agent.stopLoadTestHandler).Methods("POST")
	r.HandleFunc("/destroy/{id}", agent.destroyLoadTestHandler).Methods("POST")
	r.HandleFunc("/status/{id}", agent.getLoadTestStatusHandler).Methods("GET")
	r.HandleFunc("/user/{id}", agent.addUserHandler).Methods("PUT")
	r.HandleFunc("/user/{id}", agent.removeUserHandler).Methods("DELETE")

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
