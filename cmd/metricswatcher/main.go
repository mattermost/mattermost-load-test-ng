package main

import (
	"os"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/metricswatcher/prometheushelper"
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-server/v5/mlog"
	prometheus "github.com/prometheus/client_golang/api"
	apiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

var configuration *config.Configuration

func main() {
	rootCmd := &cobra.Command{
		Use:    "metricswatcher",
		RunE:   runMetricsWatcher,
		PreRun: config.Setup,
	}

	rootCmd.PersistentFlags().StringP("config", "c", "", "path to the configuration file to use")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runMetricsWatcher(cmd *cobra.Command, args []string) error {
	var err error
	configuration, err = config.GetConfig()

	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go healthcheck()
	go checkMetrics()

	wg.Wait()

	return nil
}

func checkMetrics() {
	var (
		config      = prometheus.Config{Address: configuration.ConnectionConfiguration.PrometheusURL}
		client, err = prometheus.NewClient(config)
	)

	if err != nil {
		mlog.Critical("Error while trying to initialize metricswatcher: %s", mlog.Err(err))
		os.Exit(1)
	}

	var (
		api        = apiv1.NewAPI(client)
		prometheus = &prometheushelper.PrometheusHelper{api}
	)

	for {
		printRequestDuration(prometheus)
		printCurrentWebsockets(prometheus)

		time.Sleep(5 * time.Second)
	}
}
