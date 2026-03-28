// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"net/http"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// TranslateRequest represents a translation request from the client.
type TranslateRequest struct {
	Q       string `json:"q"`
	Source  string `json:"source"`
	Target  string `json:"target"`
	Format  string `json:"format,omitempty"`
	APIKey  string `json:"api_key,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

// TranslateResponse represents a translation response.
type TranslateResponse struct {
	TranslatedText   string `json:"translatedText"`
	DetectedLanguage *struct {
		Confidence float64 `json:"confidence"`
		Language   string  `json:"language"`
	} `json:"detectedLanguage,omitempty"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// handleTranslate handles POST /translate requests.
func (s *server) handleTranslate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req TranslateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Q == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required parameter: q")
		return
	}

	if req.Target == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required parameter: target")
		return
	}

	// Validate API key if configured
	if s.cfg.ServerConfig.APIKey != "" && req.APIKey != s.cfg.ServerConfig.APIKey {
		writeErrorResponse(w, http.StatusForbidden, "Invalid API key")
		return
	}

	mlog.Debug("Translate request",
		mlog.String("source", req.Source),
		mlog.String("target", req.Target),
		mlog.Int("content_length", len(req.Q)))

	// Apply latency based on content length
	s.applyLatency(len(req.Q))

	// Build response - echo the input text back (no transformation)
	resp := TranslateResponse{
		TranslatedText: req.Q,
	}

	// If source is "auto", include detected language in response
	if req.Source == "auto" || req.Source == "" {
		resp.DetectedLanguage = &struct {
			Confidence float64 `json:"confidence"`
			Language   string  `json:"language"`
		}{
			Confidence: s.cfg.TranslationConfig.DetectionConfidence,
			Language:   s.cfg.TranslationConfig.DefaultSourceLanguage,
		}
	}

	writeJSONResponse(w, http.StatusOK, resp)
}

// writeJSONResponse writes a JSON response with the given status code.
func writeJSONResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// writeErrorResponse writes an error response with the given status code.
func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	writeJSONResponse(w, statusCode, ErrorResponse{Error: message})
}
