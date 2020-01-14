package main

import (
	"os"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/cmd/metricswatcher/prometheushealthcheck"

	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-server/v5/mlog"
)

func healthcheck(configuration *config.MetricsCheckConfig) {
	healthCheck, err := prometheushealthcheck.NewHealthProvider(configuration.PrometheusConfiguration.PrometheusURL)

	if err != nil {
		mlog.Error(err.Error())
		os.Exit(1)
	}

	for {
		healthcheckResult := healthCheck.Check()

		if !healthcheckResult.Healthy && healthcheckResult.Error != nil {
			mlog.Error("Prometheus is not healthy:", mlog.Err(healthcheckResult.Error))
		}

		time.Sleep(60 * time.Second)
	}
}
