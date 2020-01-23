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
	PrometheusURL string
	// The time interval in milliseconds to wait before querying again.
	UpdateIntervalMs int
	// The slice of queries to run.
	Queries []prometheus.Query
}

func (c MonitorConfig) IsValid() (bool, error) {
	if c.PrometheusURL == "" {
		return false, fmt.Errorf("PrometheusURL cannot be empty")
	}
	if c.UpdateIntervalMs < 1000 {
		return false, fmt.Errorf("UpdateInterval cannot be less than 1000")
	}
	if len(c.Queries) == 0 {
		return false, fmt.Errorf("Queries cannot be empty")
	}
	return true, nil
}
