// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"net/http"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// DetectRequest represents a language detection request.
type DetectRequest struct {
	Q      string `json:"q"`
	APIKey string `json:"api_key,omitempty"`
}

// DetectResponse represents a language detection response.
type DetectResponse []struct {
	Confidence float64 `json:"confidence"`
	Language   string  `json:"language"`
}

// handleDetect handles POST /detect requests.
func (s *server) handleDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req DetectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Q == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required parameter: q")
		return
	}

	// Validate API key if configured
	if s.cfg.ServerConfig.APIKey != "" && req.APIKey != s.cfg.ServerConfig.APIKey {
		writeErrorResponse(w, http.StatusForbidden, "Invalid API key")
		return
	}

	mlog.Debug("Detect request",
		mlog.Int("content_length", len(req.Q)))

	// Apply latency based on content length
	s.applyLatency(len(req.Q))

	// Return mock detection results
	resp := DetectResponse{
		{
			Confidence: s.cfg.TranslationConfig.DetectionConfidence,
			Language:   s.cfg.TranslationConfig.DefaultSourceLanguage,
		},
	}

	writeJSONResponse(w, http.StatusOK, resp)
}
