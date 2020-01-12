package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/cmd/metricswatcher/prometheushelper"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

type PrometheusQuery struct {
	Description string
	Query       string
	Threshold   float64
	Alert       bool
}

func checkMetrics(queryFile string) {
	var (
		prometheusQueries = readPrometheusQueriesFile(queryFile)
		prometheusHelper  = createPrometheusHelper()
	)

	for {
		checkQueries(prometheusHelper, prometheusQueries)

		time.Sleep(time.Duration(configuration.PrometheusConfiguration.UpdateIntervalInMS) * time.Millisecond)
	}
}

func createPrometheusHelper() *prometheushelper.PrometheusHelper {
	var (
		prometheusURL         = configuration.PrometheusConfiguration.PrometheusURL
		prometheusHelper, err = prometheushelper.NewPrometheusHelper(prometheusURL)
	)

	if err != nil {
		mlog.Critical("Error while trying to create Prometheus helper:", mlog.Err(err))
		os.Exit(1)
	}

	return prometheusHelper
}

func readPrometheusQueriesFile(queryFile string) []PrometheusQuery {
	jsonFile, err := os.Open(queryFile)

	if err != nil {
		mlog.Critical("Error while trying to open queries file:", mlog.Err(err))
		os.Exit(1)
	}

	defer jsonFile.Close()
	fileBytes, _ := ioutil.ReadAll(jsonFile)

	var queries []PrometheusQuery

	if err := json.Unmarshal(fileBytes, &queries); err != nil {
		mlog.Critical("Error while trying to parse queries file:", mlog.Err(err))
		os.Exit(1)
	}

	return queries
}

func checkQueries(prometheus *prometheushelper.PrometheusHelper, queries []PrometheusQuery) {
	for _, query := range queries {
		value, err := prometheus.VectorFirst(query.Query)

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
