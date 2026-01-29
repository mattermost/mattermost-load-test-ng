// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/logger"
)

// Config holds the configuration for the mock LibreTranslate server.
type Config struct {
	ServerConfig      ServerConfig      // Port, Host, APIKey
	LatencyConfig     LatencyConfig     // Latency simulation settings
	TranslationConfig TranslationConfig // Mock translation behavior
	LogSettings       logger.Settings
}

// ServerConfig holds the server configuration.
type ServerConfig struct {
	Port   int    `default:"5000"`
	Host   string `default:"0.0.0.0"`
	APIKey string `default:""`
}

// LatencyConfig holds the latency simulation settings.
type LatencyConfig struct {
	Enabled                  bool `default:"true"`
	BaseLatencyMs            int  `default:"50"`    // Minimum delay
	LatencyPerHundredCharsMs int  `default:"20"`    // Additional delay per 100 chars
	MaxLatencyMs             int  `default:"40000"` // Cap (40s to simulate slow providers)
	JitterPercent            int  `default:"20"`    // Random variance
}

// TranslationConfig holds the mock translation behavior settings.
type TranslationConfig struct {
	DefaultSourceLanguage string  `default:"en"`
	DetectionConfidence   float64 `default:"95.5"`
}

// IsValid validates the configuration and returns an error if invalid.
func (c *Config) IsValid() error {
	if c.ServerConfig.Port < 1 || c.ServerConfig.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if c.LatencyConfig.BaseLatencyMs < 0 {
		return fmt.Errorf("base latency must be non-negative")
	}
	if c.LatencyConfig.LatencyPerHundredCharsMs < 0 {
		return fmt.Errorf("latency per hundred chars must be non-negative")
	}
	if c.LatencyConfig.MaxLatencyMs < 0 {
		return fmt.Errorf("max latency must be non-negative")
	}
	if c.LatencyConfig.JitterPercent < 0 || c.LatencyConfig.JitterPercent > 100 {
		return fmt.Errorf("jitter percent must be between 0 and 100")
	}
	if c.TranslationConfig.DetectionConfidence < 0 || c.TranslationConfig.DetectionConfidence > 100 {
		return fmt.Errorf("detection confidence must be between 0 and 100")
	}
	return nil
}

// ReadConfig reads the configuration from a file and returns it.
func ReadConfig(path string) (*Config, error) {
	var cfg Config
	if err := defaults.Set(&cfg); err != nil {
		return nil, fmt.Errorf("failed to set defaults: %w", err)
	}

	if path == "" {
		return &cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}
