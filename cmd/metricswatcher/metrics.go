package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/cmd/metricswatcher/config"
	"github.com/mattermost/mattermost-load-test-ng/cmd/metricswatcher/prometheushelper"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

type PrometheusQuery struct {
	Description string
	Query       string
	Threshold   float64
	Alert       bool
}

func checkMetrics(errChan chan error, configuration *config.MetricsCheckConfig, queryFile string) {
	prometheusQueries, err := readPrometheusQueriesFile(queryFile)

	if err != nil {
		errChan <- err
		return
	}

	prometheusHelper, err := prometheushelper.NewPrometheusHelper(configuration.PrometheusConfiguration.PrometheusURL)

	if err != nil {
		errChan <- err
		return
	}

	for {
		checkQueries(prometheusHelper, prometheusQueries)

		time.Sleep(time.Duration(configuration.PrometheusConfiguration.MetricsUpdateIntervalInMS) * time.Millisecond)
	}
}

func readPrometheusQueriesFile(queryFile string) ([]PrometheusQuery, error) {
	jsonFile, err := os.Open(queryFile)

	if err != nil {
		return []PrometheusQuery{}, fmt.Errorf("error while opening queries file: %w", err)
	}

	defer jsonFile.Close()
	fileBytes, _ := ioutil.ReadAll(jsonFile)

	var queries []PrometheusQuery

	if err := json.Unmarshal(fileBytes, &queries); err != nil {
		return []PrometheusQuery{}, fmt.Errorf("error while trying to parse queries file: %w", err)
	}

	return queries, nil
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
