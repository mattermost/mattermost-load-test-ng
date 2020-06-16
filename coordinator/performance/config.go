// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package performance

import (
	"fmt"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"
)

// MonitorConfig holds the necessary information to create a Monitor.
type MonitorConfig struct {
	// The URL of the Prometheus server to query.
	PrometheusURL string `default:"http://localhost:9090" validate:"url"`
	// The time interval in milliseconds to wait before querying again.
	UpdateIntervalMs int `default:"2000" validate:"range:[1000,]"`
	// The slice of queries to run.
	Queries []prometheus.Query `default_size:"1"`
}

// IsValid checks whether a MonitorConfig is valid or not.
// Returns an error if the validation fails.
func (c MonitorConfig) IsValid() error {
	if c.PrometheusURL == "" {
		return fmt.Errorf("PrometheusURL cannot be empty")
	}
	if c.UpdateIntervalMs < 1000 {
		return fmt.Errorf("UpdateInterval cannot be less than 1000")
	}
	if len(c.Queries) == 0 {
		return fmt.Errorf("Queries cannot be empty")
	}
	return nil
}
