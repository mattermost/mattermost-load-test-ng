// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"

	"github.com/gorilla/mux"
)

// CoordinatorResponse contains the data returned by load-test coordinator API.
type coordinatorResponse struct {
	Id      string              `json:"id,omitempty"`      // The load-test coordinator unique identifier.
	Message string              `json:"message,omitempty"` // Message contains information about the response.
	Status  *coordinator.Status `json:"status,omitempty"`  // Status contains the current status of the coordinator.
	Error   string              `json:"error,omitempty"`   // Error is set if there was an error during the operation.
}

func writeCoordinatorResponse(w http.ResponseWriter, status int, resp *coordinatorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func (a *api) getCoordinatorById(w http.ResponseWriter, r *http.Request) (*coordinator.Coordinator, error) {
	vars := mux.Vars(r)
	id := vars["id"]
	val, ok := a.resources.Load(id)
	if !ok || val == nil {
		err := fmt.Errorf("load-test coordinator with id %s not found", id)
		writeCoordinatorResponse(w, http.StatusNotFound, &coordinatorResponse{
			Error: err.Error(),
		})
		return nil, err
	}

	c, ok := val.(*coordinator.Coordinator)
	if !ok {
		err := fmt.Errorf("resource with id %s is not a load-test coordinator", id)
		writeCoordinatorResponse(w, http.StatusBadRequest, &coordinatorResponse{
			Error: err.Error(),
		})
		return nil, err
	}

	return c, nil
}

func (a *api) createCoordinatorHandler(w http.ResponseWriter, r *http.Request) {
	var data struct {
		CoordinatorConfig coordinator.Config
		LoadTestConfig    loadtest.Config
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		writeCoordinatorResponse(w, http.StatusBadRequest, &coordinatorResponse{
			Error: fmt.Sprintf("could not read request: %s", err),
		})
		return
	}

	ltConfig := data.LoadTestConfig
	if err := defaults.Validate(ltConfig); err != nil {
		writeCoordinatorResponse(w, http.StatusBadRequest, &coordinatorResponse{
			Error: fmt.Sprintf("could not validate load-test config: %s", err),
		})
		return
	}

	config := data.CoordinatorConfig

	id := r.FormValue("id")
	if val, ok := a.resources.Load(id); ok && val != nil {
		writeCoordinatorResponse(w, http.StatusBadRequest, &coordinatorResponse{
			Error: fmt.Sprintf("load-test coordinator with id %s already exists", id),
		})
		return
	}

	c, err := coordinator.New(&config, ltConfig, a.coordLog)
	if err != nil {
		writeCoordinatorResponse(w, http.StatusBadRequest, &coordinatorResponse{
			Id:      id,
			Message: "load-test coordinator creation failed",
			Error:   fmt.Sprintf("could not create coordinator: %s", err),
		})
		return
	}

	a.resources.Store(id, c)

	writeCoordinatorResponse(w, http.StatusCreated, &coordinatorResponse{
		Message: "load-test coordinator created",
	})
}

func (a *api) destroyCoordinatorHandler(w http.ResponseWriter, r *http.Request) {
	c, err := a.getCoordinatorById(w, r)
	if err != nil {
		return
	}

	_ = c.Stop() // we are ignoring the error here in case the coordinator was previously stopped

	a.resources.Delete(mux.Vars(r)["id"])
	writeCoordinatorResponse(w, http.StatusOK, &coordinatorResponse{
		Message: "load-test coordinator destroyed",
	})
}

func (a *api) runCoordinatorHandler(w http.ResponseWriter, r *http.Request) {
	c, err := a.getCoordinatorById(w, r)
	if err != nil {
		return
	}

	if _, err := c.Run(); err != nil {
		writeCoordinatorResponse(w, http.StatusBadRequest, &coordinatorResponse{
			Message: "running coordinator failed",
			Error:   fmt.Sprintf("could not run coordinator: %s", err),
		})
		return
	}

	writeCoordinatorResponse(w, http.StatusOK, &coordinatorResponse{
		Message: "load-test coordinator started",
	})
}

func (a *api) stopCoordinatorHandler(w http.ResponseWriter, r *http.Request) {
	c, err := a.getCoordinatorById(w, r)
	if err != nil {
		return
	}

	if err := c.Stop(); err != nil {
		writeCoordinatorResponse(w, http.StatusBadRequest, &coordinatorResponse{
			Message: "stopping coordinator failed",
			Error:   fmt.Sprintf("could not stop coordinator: %s", err),
		})
		return
	}

	writeCoordinatorResponse(w, http.StatusOK, &coordinatorResponse{
		Message: "load-test coordinator stopped",
	})
}

func (a *api) getCoordinatorStatusHandler(w http.ResponseWriter, r *http.Request) {
	c, err := a.getCoordinatorById(w, r)
	if err != nil {
		return
	}
	status := c.Status()
	writeCoordinatorResponse(w, http.StatusOK, &coordinatorResponse{
		Status: &status,
	})
}
