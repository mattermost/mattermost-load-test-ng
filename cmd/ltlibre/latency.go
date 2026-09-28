// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"math/rand"
	"time"
)

// calculateLatency calculates the latency to apply based on content length.
// Formula: baseLatency + (contentLength / 100) * latencyPerHundredChars
// Add jitter: +/- jitterPercent of calculated latency
// Cap at maxLatency
func (s *server) calculateLatency(contentLength int) time.Duration {
	if !s.cfg.LatencyConfig.Enabled {
		return 0
	}

	cfg := s.cfg.LatencyConfig

	// Calculate base latency plus per-character latency
	charLatency := (contentLength / 100) * cfg.LatencyPerHundredCharsMs
	totalMs := cfg.BaseLatencyMs + charLatency

	// Apply jitter if configured
	if cfg.JitterPercent > 0 {
		jitterRange := (totalMs * cfg.JitterPercent) / 100
		// Generate jitter between -jitterRange and +jitterRange
		jitter := rand.Intn(2*jitterRange+1) - jitterRange
		totalMs += jitter
	}

	// Cap at max latency
	if totalMs > cfg.MaxLatencyMs {
		totalMs = cfg.MaxLatencyMs
	}

	// Ensure non-negative
	if totalMs < 0 {
		totalMs = 0
	}

	return time.Duration(totalMs) * time.Millisecond
}

// applyLatency sleeps for the calculated latency duration based on content length.
func (s *server) applyLatency(contentLength int) {
	latency := s.calculateLatency(contentLength)
	if latency > 0 {
		time.Sleep(latency)
	}
}
