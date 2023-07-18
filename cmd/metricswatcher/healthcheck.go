// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"time"

	"github.com/mattermost/mattermost-load-test-ng/cmd/metricswatcher/config"

	"github.com/mattermost/mattermost-load-test-ng/cmd/metricswatcher/prometheushealthcheck"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func healthcheck(errChan chan error, configuration *config.MetricsWatcherConfiguration) {
	healthCheck, err := prometheushealthcheck.NewHealthProvider(configuration.PrometheusConfiguration.PrometheusURL)

	if err != nil {
		errChan <- err
		return
	}

	for {
		healthcheckResult := healthCheck.Check()

		if !healthcheckResult.Healthy && healthcheckResult.Error != nil {
			mlog.Error("Prometheus is not healthy:", mlog.Err(healthcheckResult.Error))
		}

		time.Sleep(time.Duration(configuration.PrometheusConfiguration.HealthcheckUpdateIntervalInMS) * time.Millisecond)
	}
}
