// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/cmd/metricswatcher/config"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"

	"github.com/mattermost/mattermost-server/v5/shared/mlog"
)

func checkMetrics(errChan chan error, config *config.MetricsWatcherConfiguration) {
	helper, err := prometheus.NewHelper(config.PrometheusConfiguration.PrometheusURL)

	if err != nil {
		errChan <- err
		return
	}

	for {
		checkQueries(helper, config.Queries)

		time.Sleep(time.Duration(config.PrometheusConfiguration.MetricsUpdateIntervalInMS) * time.Millisecond)
	}
}

func checkQueries(helper *prometheus.Helper, queries []prometheus.Query) {
	for _, query := range queries {
		value, err := helper.VectorFirst(query.Query)

		if err != nil {
			mlog.Error("Error while querying Prometheus:", mlog.String("query_description", query.Description), mlog.Err(err))
			continue
		}

		message := fmt.Sprintf("%s = %2.8f", query.Description, value)
		mlog.Info(message)

		if query.Alert && value >= query.Threshold {
			// TODO: if we need to trigger some event, this would be the place.

			mlog.Warn("Returned value is above the threshold.",
				mlog.String("query_description", query.Description),
				mlog.String("query_returned_value", fmt.Sprintf("%2.8f", value)),
				mlog.String("query_threshold", fmt.Sprintf("%2.8f", query.Threshold)),
			)
		}
	}
}
