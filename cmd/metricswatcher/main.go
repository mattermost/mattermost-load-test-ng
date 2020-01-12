package main

import (
	"os"
	"sync"

	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-load-test-ng/config"
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
