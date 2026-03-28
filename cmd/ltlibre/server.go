// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

// server holds the state for the mock LibreTranslate server.
type server struct {
	cfg *Config
}

// newServer creates a new server instance.
func newServer(cfg *Config) *server {
	return &server{
		cfg: cfg,
	}
}

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status string `json:"status"`
}

// handleHealth handles GET / and GET /health requests.
func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	writeJSONResponse(w, http.StatusOK, HealthResponse{Status: "ok"})
}

// setupRouter creates and configures the HTTP router.
func (s *server) setupRouter() *mux.Router {
	r := mux.NewRouter()

	// Health check endpoints
	r.HandleFunc("/", s.handleHealth).Methods(http.MethodGet)
	r.HandleFunc("/health", s.handleHealth).Methods(http.MethodGet)

	// LibreTranslate API endpoints
	r.HandleFunc("/translate", s.handleTranslate).Methods(http.MethodPost)
	r.HandleFunc("/detect", s.handleDetect).Methods(http.MethodPost)
	r.HandleFunc("/languages", s.handleLanguages).Methods(http.MethodGet)

	return r
}
