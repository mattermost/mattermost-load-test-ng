// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package performance

import (
	"errors"
	"fmt"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// MonitorConfig holds the necessary information to create a Monitor.
type MonitorConfig struct {
	// The URL of the Prometheus server to query.
	PrometheusURL string `default:"http://localhost:9090" validate:"url"`
	// The time interval in milliseconds to wait before querying again.
	// DEPRECATED as of MM-61922. It defaults to 1000 and values in config files are ignored.
	UpdateIntervalMs int `default:"1000"`
	// The slice of queries to run.
	Queries []prometheus.Query `default_size:"0"`
}

// IsValid checks whether a MonitorConfig is valid or not.
// Returns an error if the validation fails.
func (c MonitorConfig) IsValid() error {
	if c.PrometheusURL == "" {
		return errors.New("PrometheusURL cannot be empty")
	}
	if c.UpdateIntervalMs != defaultUpdateIntervalMs {
		mlog.Warn(fmt.Sprintf("monitor: UpdateIntervalMs (%v) is deprecated and will be ignored. Its value always defaults to 1000ms.", c.UpdateIntervalMs))
	}
	return nil
}
