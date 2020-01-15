package main

import (
	"os"

	"github.com/mattermost/mattermost-server/v5/mlog"

	"github.com/mattermost/mattermost-load-test-ng/cmd/metricswatcher/config"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:           "metricswatcher",
		RunE:          runMetricsWatcher,
		PreRun:        config.SetupMetricsCheck,
		SilenceErrors: true, // Since we're printing our logs, we don't need cobra to print the errors
		SilenceUsage:  true, // For some reason cobra prints the usage when some error happens in Execute()
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

	errChan := make(chan error, 1)
	defer close(errChan)

	go healthcheck(errChan, configuration)
	go checkMetrics(errChan, configuration, jsonQueryFile)

	err = <-errChan

	mlog.Error(err.Error())

	return err

}
