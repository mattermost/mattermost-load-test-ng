// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

func (ue *UserEntity) incWebSocketConnections() {
	if ue.metrics != nil {
		ue.metrics.WebSocketConnections.Inc()
	}
}

func (ue *UserEntity) decWebSocketConnections() {
	if ue.metrics != nil {
		ue.metrics.WebSocketConnections.Dec()
	}
}

func (ue *UserEntity) incHTTPErrors(path, method string, status int) {
	if ue.metrics != nil {
		ue.metrics.HTTPErrors.With(prometheus.Labels{
			"path":        path,
			"method":      method,
			"status_code": strconv.Itoa(status),
		}).Inc()
	}
}

func (ue *UserEntity) observeHTTPRequestTimes(path, method string, status int, elapsed float64) {
	if ue.metrics != nil {
		ue.metrics.HTTPRequestTimes.With(prometheus.Labels{
			"path":        path,
			"method":      method,
			"status_code": strconv.Itoa(status),
		}).Observe(elapsed)
	}
}

func (ue *UserEntity) incHTTPTimeouts(path, method string) {
	if ue.metrics != nil {
		ue.metrics.HTTPTimeouts.With(prometheus.Labels{
			"path":   path,
			"method": method,
		}).Inc()
	}
}
