// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package performance

import (
	"errors"
	"fmt"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

type Monitor struct {
	config     MonitorConfig
	helper     *prometheus.Helper
	stopChan   chan struct{}
	statusChan chan Status
	log        *mlog.Logger
	startTime  time.Time
}

// NewMonitor creates and initializes a new Monitor.
func NewMonitor(config MonitorConfig, log *mlog.Logger) (*Monitor, error) {
	if log == nil {
		return nil, errors.New("logger should not be nil")
	}
	if err := config.IsValid(); err != nil {
		return nil, fmt.Errorf("could not validate configuration: %w", err)
	}
	helper, err := prometheus.NewHelper(config.PrometheusURL)
	if err != nil {
		return nil, fmt.Errorf("performance: failed to create prometheus.Helper: %w", err)
	}
	return &Monitor{
		config:     config,
		helper:     helper,
		stopChan:   make(chan struct{}),
		statusChan: make(chan Status),
		log:        log,
		startTime:  time.Now(),
	}, nil
}

// Run will start the performance monitoring process.
func (m *Monitor) Run() <-chan Status {
	go func() {
		m.log.Info("monitor: started")
		for {
			m.statusChan <- m.runQueries()
			select {
			case <-m.stopChan:
				m.log.Info("monitor: shutting down")
				return
			case <-time.After(time.Duration(m.config.UpdateIntervalMs) * time.Millisecond):
			}
		}
	}()
	return m.statusChan
}

// Stop will stop the monitoring process.
func (m *Monitor) Stop() {
	m.log.Info("monitor: stop")
	close(m.stopChan)
}

func (m *Monitor) runQueries() Status {
	var status Status
	for _, query := range m.config.Queries {
		select {
		case <-m.stopChan:
			m.log.Info("monitor: exiting query loop")
			return Status{}
		default:
		}
		if time.Now().Before(m.startTime.Add(time.Duration(query.MinIntervalSec) * time.Second)) {
			m.log.Info("monitor: MinIntervalSec has not passed yet, skipping query")
			continue
		}
		value, err := m.helper.VectorFirst(query.Query)
		if err != nil {
			m.log.Warn("monitor: error while querying Prometheus:", mlog.String("query_description", query.Description), mlog.Err(err))
			continue
		}

		m.log.Debug("monitor: ran query",
			mlog.String("query_description", query.Description),
			mlog.String("query_returned_value", fmt.Sprintf("%2.8f", value)),
			mlog.String("query_threshold", fmt.Sprintf("%2.8f", query.Threshold)),
		)
		if query.Alert && value >= query.Threshold {
			m.log.Warn("monitor: returned value is above the threshold",
				mlog.String("query_description", query.Description),
				mlog.String("query_returned_value", fmt.Sprintf("%2.8f", value)),
				mlog.String("query_threshold", fmt.Sprintf("%2.8f", query.Threshold)),
			)
			status = Status{Alert: true}
			break
		}
	}
	return status
}
