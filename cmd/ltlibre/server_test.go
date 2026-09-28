// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) *server {
	t.Helper()
	cfg := &Config{
		ServerConfig: ServerConfig{
			Port:   5000,
			Host:   "0.0.0.0",
			APIKey: "",
		},
		LatencyConfig: LatencyConfig{
			Enabled:                  false, // Disable latency for tests
			BaseLatencyMs:            50,
			LatencyPerHundredCharsMs: 20,
			MaxLatencyMs:             40000,
			JitterPercent:            0, // No jitter for deterministic tests
		},
		TranslationConfig: TranslationConfig{
			DefaultSourceLanguage: "en",
			DetectionConfidence:   95.5,
		},
	}
	return newServer(cfg)
}

func TestHealthEndpoint(t *testing.T) {
	srv := newTestServer(t)
	router := srv.setupRouter()

	tests := []struct {
		name     string
		endpoint string
	}{
		{"root", "/"},
		{"health", "/health"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.endpoint, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var resp HealthResponse
			err := json.NewDecoder(w.Body).Decode(&resp)
			require.NoError(t, err)
			assert.Equal(t, "ok", resp.Status)
		})
	}
}

func TestLanguagesEndpoint(t *testing.T) {
	srv := newTestServer(t)
	router := srv.setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/languages", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var languages []Language
	err := json.NewDecoder(w.Body).Decode(&languages)
	require.NoError(t, err)
	assert.NotEmpty(t, languages)

	// Verify expected languages are present
	languageCodes := make(map[string]bool)
	for _, lang := range languages {
		languageCodes[lang.Code] = true
	}

	assert.True(t, languageCodes["en"], "English should be present")
	assert.True(t, languageCodes["es"], "Spanish should be present")
	assert.True(t, languageCodes["fr"], "French should be present")
}

func TestTranslateEndpointEchosInput(t *testing.T) {
	srv := newTestServer(t)
	router := srv.setupRouter()

	testText := "Hello, world! This is a test message."
	reqBody := TranslateRequest{
		Q:      testText,
		Source: "en",
		Target: "es",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/translate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp TranslateResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	// Verify input is echoed back
	assert.Equal(t, testText, resp.TranslatedText)

	// Verify no detected language when source is explicitly set
	assert.Nil(t, resp.DetectedLanguage)
}

func TestTranslateEndpointAutoDetection(t *testing.T) {
	srv := newTestServer(t)
	router := srv.setupRouter()

	testText := "Hello, world!"
	reqBody := TranslateRequest{
		Q:      testText,
		Source: "auto",
		Target: "es",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/translate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp TranslateResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	// Verify input is echoed back
	assert.Equal(t, testText, resp.TranslatedText)

	// Verify detected language is included when source is "auto"
	require.NotNil(t, resp.DetectedLanguage)
	assert.Equal(t, "en", resp.DetectedLanguage.Language)
	assert.Equal(t, 95.5, resp.DetectedLanguage.Confidence)
}

func TestTranslateEndpointMissingParameters(t *testing.T) {
	srv := newTestServer(t)
	router := srv.setupRouter()

	tests := []struct {
		name     string
		request  TranslateRequest
		expected string
	}{
		{
			name:     "missing q",
			request:  TranslateRequest{Source: "en", Target: "es"},
			expected: "Missing required parameter: q",
		},
		{
			name:     "missing target",
			request:  TranslateRequest{Q: "Hello", Source: "en"},
			expected: "Missing required parameter: target",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(tc.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/translate", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var resp ErrorResponse
			err = json.NewDecoder(w.Body).Decode(&resp)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, resp.Error)
		})
	}
}

func TestDetectEndpoint(t *testing.T) {
	srv := newTestServer(t)
	router := srv.setupRouter()

	testText := "Hello, world!"
	reqBody := DetectRequest{
		Q: testText,
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/detect", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp DetectResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	require.Len(t, resp, 1)
	assert.Equal(t, "en", resp[0].Language)
	assert.Equal(t, 95.5, resp[0].Confidence)
}

func TestDetectEndpointMissingQ(t *testing.T) {
	srv := newTestServer(t)
	router := srv.setupRouter()

	reqBody := DetectRequest{}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/detect", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "Missing required parameter: q", resp.Error)
}

func TestAPIKeyValidation(t *testing.T) {
	cfg := &Config{
		ServerConfig: ServerConfig{
			Port:   5000,
			Host:   "0.0.0.0",
			APIKey: "test-api-key",
		},
		LatencyConfig: LatencyConfig{
			Enabled: false,
		},
		TranslationConfig: TranslationConfig{
			DefaultSourceLanguage: "en",
			DetectionConfidence:   95.5,
		},
	}
	srv := newServer(cfg)
	router := srv.setupRouter()

	tests := []struct {
		name       string
		apiKey     string
		expectCode int
	}{
		{
			name:       "valid api key",
			apiKey:     "test-api-key",
			expectCode: http.StatusOK,
		},
		{
			name:       "invalid api key",
			apiKey:     "wrong-key",
			expectCode: http.StatusForbidden,
		},
		{
			name:       "missing api key",
			apiKey:     "",
			expectCode: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := TranslateRequest{
				Q:      "Hello",
				Source: "en",
				Target: "es",
				APIKey: tc.apiKey,
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/translate", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectCode, w.Code)
		})
	}
}

func TestLatencyCalculation(t *testing.T) {
	cfg := &Config{
		LatencyConfig: LatencyConfig{
			Enabled:                  true,
			BaseLatencyMs:            50,
			LatencyPerHundredCharsMs: 20,
			MaxLatencyMs:             40000,
			JitterPercent:            0, // No jitter for deterministic tests
		},
	}
	srv := newServer(cfg)

	tests := []struct {
		name          string
		contentLength int
		expectedMs    int
	}{
		{"50 chars", 50, 50},         // 50 + 0*20 = 50ms
		{"100 chars", 100, 70},       // 50 + 1*20 = 70ms
		{"200 chars", 200, 90},       // 50 + 2*20 = 90ms
		{"500 chars", 500, 150},      // 50 + 5*20 = 150ms
		{"1000 chars", 1000, 250},    // 50 + 10*20 = 250ms
		{"2000 chars", 2000, 450},    // 50 + 20*20 = 450ms
		{"10000 chars", 10000, 2050}, // 50 + 100*20 = 2050ms
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			latency := srv.calculateLatency(tc.contentLength)
			assert.Equal(t, tc.expectedMs, int(latency.Milliseconds()))
		})
	}
}

func TestLatencyCalculationCapped(t *testing.T) {
	cfg := &Config{
		LatencyConfig: LatencyConfig{
			Enabled:                  true,
			BaseLatencyMs:            50,
			LatencyPerHundredCharsMs: 20,
			MaxLatencyMs:             40000,
			JitterPercent:            0,
		},
	}
	srv := newServer(cfg)

	// Very large content should be capped at max latency
	// 200000 chars would be 50 + 2000*20 = 40050ms, but capped at 40000ms
	latency := srv.calculateLatency(200000)
	assert.Equal(t, 40000, int(latency.Milliseconds()))
}

func TestLatencyDisabled(t *testing.T) {
	cfg := &Config{
		LatencyConfig: LatencyConfig{
			Enabled:                  false,
			BaseLatencyMs:            50,
			LatencyPerHundredCharsMs: 20,
			MaxLatencyMs:             40000,
			JitterPercent:            0,
		},
	}
	srv := newServer(cfg)

	latency := srv.calculateLatency(1000)
	assert.Equal(t, 0, int(latency.Milliseconds()))
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid config",
			config: Config{
				ServerConfig:      ServerConfig{Port: 5000},
				LatencyConfig:     LatencyConfig{JitterPercent: 20},
				TranslationConfig: TranslationConfig{DetectionConfidence: 95.5},
			},
			expectError: false,
		},
		{
			name: "invalid port",
			config: Config{
				ServerConfig: ServerConfig{Port: 0},
			},
			expectError: true,
		},
		{
			name: "negative base latency",
			config: Config{
				ServerConfig:  ServerConfig{Port: 5000},
				LatencyConfig: LatencyConfig{BaseLatencyMs: -1},
			},
			expectError: true,
		},
		{
			name: "jitter too high",
			config: Config{
				ServerConfig:  ServerConfig{Port: 5000},
				LatencyConfig: LatencyConfig{JitterPercent: 101},
			},
			expectError: true,
		},
		{
			name: "invalid detection confidence",
			config: Config{
				ServerConfig:      ServerConfig{Port: 5000},
				TranslationConfig: TranslationConfig{DetectionConfidence: 101},
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.IsValid()
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
