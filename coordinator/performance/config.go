package performance

import (
	"fmt"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"
)

type MonitorConfig struct {
	PrometheusURL  string
	UpdateInterval int
	Queries        []prometheus.Query
}

func (c MonitorConfig) IsValid() (bool, error) {
	if c.PrometheusURL == "" {
		return false, fmt.Errorf("PrometheusURL cannot be empty")
	}
	if c.UpdateInterval < 1000 {
		return false, fmt.Errorf("UpdateInterval cannot be less than 1000")
	}
	if len(c.Queries) == 0 {
		return false, fmt.Errorf("Queries cannot be empty")
	}
	return true, nil
}
