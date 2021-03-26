// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"os"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/spf13/cobra"
)

func RunCreateCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t := terraform.New("", config)
	defer t.Cleanup()
	return t.Create(true)
}

func RunDestroyCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t := terraform.New("", config)
	defer t.Cleanup()
	return t.Destroy()
}

func RunInfoCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t := terraform.New("", config)
	return t.Info()
}

func RunSyncCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t := terraform.New("", config)
	return t.Sync()
}

func RunSSHListCmdF(cmd *cobra.Command, args []string) error {
	t := terraform.New("", nil)
	output, err := t.Output()
	if err != nil {
		return fmt.Errorf("could not parse output: %w", err)
	}
	for _, agent := range output.Agents {
		fmt.Printf(" - %s\n", agent.Tags.Name)
	}
	for _, instance := range output.Instances {
		fmt.Printf(" - %s\n", instance.Tags.Name)
	}
	if output.HasProxy() {
		fmt.Printf(" - %s\n", output.Proxy.Tags.Name)
	}
	if output.HasMetrics() {
		fmt.Printf(" - %s\n", output.MetricsServer.Tags.Name)
	}
	return nil
}

func getConfig(cmd *cobra.Command) (*deployment.Config, error) {
	configFilePath, _ := cmd.Flags().GetString("config")
	cfg, err := deployment.ReadConfig(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := defaults.Validate(cfg); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	logger.Init(&cfg.LogSettings)
	return cfg, nil
}

func main() {
	rootCmd := &cobra.Command{
		Use:          "ltctl",
		SilenceUsage: true,
		Short:        "Manage and control load-test deployments",
	}
	rootCmd.PersistentFlags().StringP("config", "c", "", "path to the deployer configuration file to use")

	deploymentCmd := &cobra.Command{
		Use:   "deployment",
		Short: "Manage a load-test deployment",
	}

	deploymentCommands := []*cobra.Command{
		{
			Use:   "create",
			Short: "Create a new load-test deployment",
			RunE:  RunCreateCmdF,
		},
		{
			Use:   "destroy",
			Short: "Destroy the current load-test deployment",
			RunE:  RunDestroyCmdF,
		},
		{
			Use:   "info",
			Short: "Display information about the current load-test deployment",
			RunE:  RunInfoCmdF,
		},
		{
			Use:   "sync",
			Short: "Syncs the local .tfstate file with any changes made remotely",
			RunE:  RunSyncCmdF,
		},
	}

	deploymentCmd.AddCommand(deploymentCommands...)
	rootCmd.AddCommand(deploymentCmd)

	loadtestCmd := &cobra.Command{
		Use:   "loadtest",
		Short: "Manage the load-test",
	}

	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset and re-initialize target instance database",
		RunE:  RunResetCmdF,
	}
	resetCmd.Flags().Bool("confirm", false, "Confirm you really want to reset the database and re-initialize it.")

	loadtestComands := []*cobra.Command{
		{
			Use:   "start",
			Short: "Start the coordinator in the current load-test deployment",
			RunE:  RunLoadTestStartCmdF,
		},
		{
			Use:   "stop",
			Short: "Stop the coordinator in the current load-test deployment",
			RunE:  RunLoadTestStopCmdF,
		},
		{
			Use:   "status",
			Short: "Shows the status of the current load-test",
			RunE:  RunLoadTestStatusCmdF,
		},
		resetCmd,
	}

	loadtestCmd.AddCommand(loadtestComands...)
	rootCmd.AddCommand(loadtestCmd)

	sshCmd := &cobra.Command{
		Use:     "ssh [instance]",
		Short:   "ssh into instance",
		Example: "ltctl ssh agent-0",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Println("Available instances:")
				return RunSSHListCmdF(cmd, args)
			}
			return terraform.New("", nil).OpenSSHFor(args[0])
		},
	}

	sshListCmd := &cobra.Command{
		Use:   "list",
		Short: "lists available resources",
		RunE:  RunSSHListCmdF,
		Args:  cobra.NoArgs,
	}
	sshCmd.AddCommand(sshListCmd)
	rootCmd.AddCommand(sshCmd)

	goCmd := &cobra.Command{
		Use:     "go [instance]",
		Short:   "Open browser for instance",
		Long:    "Open browser for grafana, mattermost or prometheus",
		Example: "ltctl go grafana",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				fmt.Println("Available destinations:")
				for _, arg := range cmd.ValidArgs {
					fmt.Printf("ltctl go %s\n", arg)
				}
				return nil
			}
			return terraform.New("", nil).OpenBrowserFor(args[0])
		},
		Args:      cobra.OnlyValidArgs,
		ValidArgs: []string{"grafana", "mattermost", "prometheus"},
	}
	rootCmd.AddCommand(goCmd)

	collectCmd := &cobra.Command{
		Use:     "collect",
		Short:   "Collect logs and configurations",
		Example: "ltctl collect",
		RunE:    RunCollectCmdF,
	}
	rootCmd.AddCommand(collectCmd)

	reportCmd := &cobra.Command{
		Use:   "report",
		Short: "Get or compare reports from load-tests",
	}

	genReport := &cobra.Command{
		Use:     "generate",
		Short:   "Generate a report from a load-test from a start time to end time.",
		Example: "ltctl report generate --output=base.out --label=base \"2020-06-17 04:37:05\" \"2020-06-17 04:42:00\"",
		RunE:    RunGenerateReportCmdF,
	}
	genReport.Flags().StringP("output", "o", "ltreport.out", "Path to the output file to write the report to.")
	genReport.Flags().StringP("label", "l", "", "A friendly name for the report.")
	genReport.Flags().StringP("prometheus-url", "p", "", "The URL of the Prometheus server. If this is not passed, the value is taken from terraform.tfstate.")

	compareReport := &cobra.Command{
		Use:     "compare",
		Short:   "Compare one or more reports",
		Long:    "Compare one or more reports. The first report is considered to be the base",
		Example: "ltctl report compare report1.out report2.out",
		RunE:    RunCompareReportCmdF,
	}
	compareReport.Flags().StringP("output", "o", "", "Path to the output file to write the comparison to. If this is not set, the report is displayed to stdout.")
	compareReport.Flags().Bool("graph", false, "If set to true, it also generates graphs comparing different metrics from the load tests. This needs gnuplot to be present in the system.")
	compareReport.Flags().Bool("dashboard", false, "If set to true, it also generates a comparative Grafana dashboard between the load tests.")

	reportCmds := []*cobra.Command{genReport, compareReport}
	reportCmd.AddCommand(reportCmds...)
	rootCmd.AddCommand(reportCmd)

	comparisonCmd := &cobra.Command{
		Use:   "comparison",
		Short: "Manage fully automated load-test comparisons environments",
	}
	comparisonCmd.Flags().StringP("comparison-config", "", "", "path to the comparison config file to use")
	runComparisonCmd := &cobra.Command{
		Use:   "run",
		Short: "Run fully automated load-test comparisons",
		RunE:  RunComparisonCmdF,
	}
	runComparisonCmd.Flags().Bool("archive", false, "create zip archive")
	runComparisonCmd.Flags().StringP("output-dir", "d", "", "path to output directory")
	runComparisonCmd.Flags().StringP("format", "f", "plain", "output format [plain, json]")

	destroyComparisonCmd := &cobra.Command{
		Use:   "destroy",
		Short: "Destroy the current load-test comparison environment",
		RunE:  DestroyComparisonCmdF,
	}
	comparisonCmd.AddCommand(runComparisonCmd, destroyComparisonCmd)
	rootCmd.AddCommand(comparisonCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
