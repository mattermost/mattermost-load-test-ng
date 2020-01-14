package main

import (
	"os"
	"sync"

	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-load-test-ng/config"
)

func main() {
	rootCmd := &cobra.Command{
		Use:    "metricswatcher",
		RunE:   runMetricsWatcher,
		PreRun: config.Setup,
	}

	persistentFlags := rootCmd.PersistentFlags()
	persistentFlags.StringP("config", "c", "", "path to the configuration file to use")
	persistentFlags.StringP("queries", "q", "", "path to the JSON file with Prometheus queries")
	cobra.MarkFlagRequired(persistentFlags, "queries")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runMetricsWatcher(cmd *cobra.Command, args []string) error {
	configuration, err := config.GetMetricsCheckConfig()
	jsonQueryFile, _ := cmd.Flags().GetString("queries")

	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go healthcheck(configuration)
	go checkMetrics(configuration, jsonQueryFile)

	wg.Wait()

	return nil
}
