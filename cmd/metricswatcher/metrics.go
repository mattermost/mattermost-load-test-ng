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
		checkQueries(prometheusHelper)

		time.Sleep(time.Duration(prometheusConfiguration.UpdateIntervalInMS) * time.Millisecond)
	}
}

type PrometheusQuery struct {
	Description string
	Query       string
	Threshold   float64
	Alert       bool
}

var queries = []PrometheusQuery{
	PrometheusQuery{
		Description: "Request duration",
		Query:       `rate(mattermost_http_request_duration_seconds_sum[5m])/rate(mattermost_http_request_duration_seconds_count[5m])`,
		Threshold:   1.0,
		Alert:       true,
	},
	PrometheusQuery{
		Description: "Total amount of websockets",
		Query:       `mattermost_http_websockets_total`,
		Threshold:   0.0,
		Alert:       false,
	},
}

func checkQueries(prometheus *prometheushelper.PrometheusHelper) {
	for _, query := range queries {
		value, err := prometheus.VectorFirst(query.Query)

		if err != nil {
			mlog.Error("Error while querying Prometheus for %s: %s", mlog.String("query_description", query.Description), mlog.Err(err))
			continue
		}

		message := fmt.Sprintf("%s = %2.8f", query.Description, value)
		mlog.Debug(message)

		if query.Alert && value >= query.Threshold {
			// TODO: if we need to trigger some event, this would be the place.

			mlog.Warn("%s value is %s, threshold is %s",
				mlog.String("query_description", query.Description),
				mlog.String("query_returned_value", fmt.Sprintf("%2.8f", value)),
				mlog.String("query_threshold", fmt.Sprintf("%2.8f", query.Threshold)),
			)
		}
	}
}
