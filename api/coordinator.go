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

func writeCoordinatorResponse(w http.ResponseWriter, status int, resp *coordinatorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func (a *api) getCoordinatorById(w http.ResponseWriter, r *http.Request) (*coordinator.Coordinator, error) {
	vars := mux.Vars(r)
	id := vars["id"]
	val, ok := a.getResource(id)
	if !ok || val == nil {
		err := fmt.Errorf("load-test coordinator with id %s not found", id)
		writeCoordinatorResponse(w, http.StatusNotFound, &CoordinatorResponse{
			Error: err.Error(),
		})
		return nil, err
	}

	c, ok := val.(*coordinator.Coordinator)
	if !ok {
		err := fmt.Errorf("resource with id %s is not a load-test coordinator", id)
		writeCoordinatorResponse(w, http.StatusBadRequest, &CoordinatorResponse{
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
		writeCoordinatorResponse(w, http.StatusBadRequest, &CoordinatorResponse{
			Error: fmt.Sprintf("could not read request: %s", err),
		})
		return
	}

	ltConfig := data.LoadTestConfig
	if err := defaults.Validate(ltConfig); err != nil {
		writeCoordinatorResponse(w, http.StatusBadRequest, &CoordinatorResponse{
			Error: fmt.Sprintf("could not validate load-test config: %s", err),
		})
		return
	}

	config := data.CoordinatorConfig

	id := r.FormValue("id")
	if val, ok := a.getResource(id); ok && val != nil {
		if _, ok := val.(*coordinator.Coordinator); ok {
			writeCoordinatorResponse(w, http.StatusBadRequest, &CoordinatorResponse{
				Error: fmt.Sprintf("load-test coordinator with id %s already exists", id),
			})
		} else {
			writeCoordinatorResponse(w, http.StatusBadRequest, &CoordinatorResponse{
				Error: fmt.Sprintf("resource with id %s already exists", id),
			})
		}
		return
	}

	c, err := coordinator.New(&config, ltConfig, a.coordLog)
	if err != nil {
		writeCoordinatorResponse(w, http.StatusBadRequest, &CoordinatorResponse{
			Id:      id,
			Message: "load-test coordinator creation failed",
			Error:   fmt.Sprintf("could not create coordinator: %s", err),
		})
		return
	}

	a.setResource(id, c)

	writeCoordinatorResponse(w, http.StatusCreated, &coordinatorResponse{
		Id:      id,
		Message: "load-test coordinator created",
	})
}

func (a *api) destroyCoordinatorHandler(w http.ResponseWriter, r *http.Request) {
	c, err := a.getCoordinatorById(w, r)
	if err != nil {
		return
	}

	_ = c.Stop() // we are ignoring the error here in case the coordinator was previously stopped

	a.deleteResource(mux.Vars(r)["id"])
	status := c.Status()
	writeCoordinatorResponse(w, http.StatusOK, &coordinatorResponse{
		Message: "load-test coordinator destroyed",
		Status:  &status,
	})
}

func (a *api) runCoordinatorHandler(w http.ResponseWriter, r *http.Request) {
	c, err := a.getCoordinatorById(w, r)
	if err != nil {
		return
	}

	if _, err := c.Run(); err != nil {
		writeCoordinatorResponse(w, http.StatusBadRequest, &CoordinatorResponse{
			Message: "running coordinator failed",
			Error:   fmt.Sprintf("could not run coordinator: %s", err),
		})
		return
	}

	status := c.Status()
	writeCoordinatorResponse(w, http.StatusOK, &coordinatorResponse{
		Message: "load-test coordinator started",
		Status:  &status,
	})
}

func (a *api) stopCoordinatorHandler(w http.ResponseWriter, r *http.Request) {
	c, err := a.getCoordinatorById(w, r)
	if err != nil {
		return
	}

	if err := c.Stop(); err != nil {
		writeCoordinatorResponse(w, http.StatusBadRequest, &CoordinatorResponse{
			Message: "stopping coordinator failed",
			Error:   fmt.Sprintf("could not stop coordinator: %s", err),
		})
		return
	}

	status := c.Status()
	writeCoordinatorResponse(w, http.StatusOK, &coordinatorResponse{
		Message: "load-test coordinator stopped",
		Status:  &status,
	})
}

func (a *api) getCoordinatorStatusHandler(w http.ResponseWriter, r *http.Request) {
	c, err := a.getCoordinatorById(w, r)
	if err != nil {
		return
	}
	status := c.Status()
	writeCoordinatorResponse(w, http.StatusOK, &CoordinatorResponse{
		Status: &status,
	})
}
