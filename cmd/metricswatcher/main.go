package main

import (
	"os"
	"sync"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:    "metricswatcher",
		RunE:   runMetricsWatcher,
		PreRun: setupMetricsCheck,
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
	configuration, err := GetMetricsCheckConfig()
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
