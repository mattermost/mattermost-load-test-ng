package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/metricswatcher/prometheushelper"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

func checkMetrics() {
	var (
		prometheusConfiguration = configuration.PrometheusConfiguration
		prometheusURL           = prometheusConfiguration.PrometheusURL
		prometheusHelper, err   = prometheushelper.NewPrometheusHelper(prometheusURL)
	)

	if err != nil {
		mlog.Critical("Error while trying to initialize metricswatcher: %s", mlog.Err(err))
		os.Exit(1)
	}

	for {
		printRequestDuration(prometheusHelper)
		printCurrentWebsockets(prometheusHelper)

		time.Sleep(time.Duration(prometheusConfiguration.UpdateIntervalInMS) * time.Millisecond)
	}
}

func printRequestDuration(prometheus *prometheushelper.PrometheusHelper) {
	query := `rate(mattermost_http_request_duration_seconds_sum[5m])/rate(mattermost_http_request_duration_seconds_count[5m])`

	if requestDuration, err := prometheus.VectorFirst(query); err == nil {
		message := fmt.Sprintf("Request duration is %2.8f", requestDuration)
		mlog.Info(message)
	} else {
		mlog.Error("Error while querying Prometheus for request duration: %s", mlog.Err(err))
	}
}

func printCurrentWebsockets(prometheus *prometheushelper.PrometheusHelper) {
	requestDurationQuery := `mattermost_http_websockets_total`

	if amountOfWebsockets, err := prometheus.VectorFirst(requestDurationQuery); err == nil {
		message := fmt.Sprintf("Current amount of websockets is %1.0f", amountOfWebsockets)
		mlog.Info(message)
	} else {
		mlog.Error("Error while querying Prometheus for current amount of websockets: %s", mlog.Err(err))
	}
}
