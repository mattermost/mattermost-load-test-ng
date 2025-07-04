// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost-load-test-ng/logger"
	"github.com/mattermost/mattermost-load-test-ng/version"
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/spf13/cobra"
)

func RunCreateCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return fmt.Errorf("failed to create SSH agent: %w", err)
	}

	initData := config.DBDumpURI == "" && config.ExternalDBSettings.DataSource == ""
	if err = t.Create(extAgent, initData); err != nil {
		return fmt.Errorf("failed to create terraform env: %w", err)
	}

	if err := t.PostProcessDatabase(extAgent); err != nil {
		return fmt.Errorf("failed to post-process database: %w", err)
	}

	return nil
}

func destroyAllButMetrics(config deployment.Config) error {
	// Override all created resources so we destroy everything but the metrics
	// instance.
	config.MarkForDestroyAllButMetrics()

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return fmt.Errorf("failed to create SSH agent: %w", err)
	}

	return t.Create(extAgent, false)
}

func RunDestroyCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	maintainMetrics, err := cmd.Flags().GetBool("do-not-destroy-metrics-instance")
	if err != nil {
		return fmt.Errorf("failed getting the --do-not-destroy-metrics-instance flag: %w", err)
	}

	if maintainMetrics {
		return destroyAllButMetrics(config)
	}

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}

	return t.Destroy()
}

func RunInfoCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}

	return t.Info()
}

func RunSyncCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}

	return t.Sync()
}

func RunStopDBCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}

	return t.StopDB()
}

func RunStartDBCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}

	return t.StartDB()
}

func RunDBStatusCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}

	status, err := t.DBStatus()
	if err != nil {
		return fmt.Errorf("failed to get DB status: %w", err)
	}

	fmt.Println("Status: ", status)

	return nil
}

func RunVersionCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}

	output, err := t.Output()
	if err != nil {
		return fmt.Errorf("could not parse output: %w", err)
	}

	if len(output.Agents) == 0 {
		return fmt.Errorf("no agents found in deployment")
	}

	// Get the first agent's IP
	agentIP := output.Agents[0].PublicIP
	if agentIP == "" {
		return fmt.Errorf("agent has no public IP")
	}

	// Query the agent's API for version info
	url := fmt.Sprintf("http://%s:4000/version", agentIP)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to query agent version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK response: %d", resp.StatusCode)
	}

	var versionInfo version.VersionInfo

	if err := json.NewDecoder(resp.Body).Decode(&versionInfo); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Println("Agent Version Information:")
	fmt.Println(versionInfo.String())

	return nil
}

func RunSSHListCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}

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
		for _, inst := range output.Proxies {
			fmt.Printf(" - %s\n", inst.Tags.Name)
		}
	}
	if output.HasMetrics() {
		fmt.Printf(" - %s\n", output.MetricsServer.Tags.Name)
	}
	if output.HasKeycloak() {
		fmt.Printf(" - %s\n", output.KeycloakServer.Tags.Name)
	}
	if output.HasOpenLDAP() {
		fmt.Printf(" - %s\n", output.OpenLDAPServer.Tags.Name)
	}
	return nil
}

func getConfig(cmd *cobra.Command) (deployment.Config, error) {
	configFilePath, _ := cmd.Flags().GetString("config")
	cfg, err := deployment.ReadConfig(configFilePath)
	if err != nil {
		return deployment.Config{}, fmt.Errorf("failed to read config: %w", err)
	}

	if err := defaults.Validate(cfg); err != nil {
		return deployment.Config{}, fmt.Errorf("failed to validate config: %w", err)
	}

	logger.Init(&cfg.LogSettings)
	return *cfg, nil
}

func setServiceEnv(cmd *cobra.Command) {
	serviceEnv, _ := cmd.Flags().GetString("service_environment")
	// Set it to test if it's neither prod nor test.
	if serviceEnv != model.ServiceEnvironmentProduction && serviceEnv != model.ServiceEnvironmentTest {
		serviceEnv = model.ServiceEnvironmentTest
	}
	os.Setenv("MM_SERVICEENVIRONMENT", serviceEnv)
}

func main() {
	rootCmd := &cobra.Command{
		Use:          "ltctl",
		SilenceUsage: true,
		Short:        "Manage and control load-test deployments",
	}
	rootCmd.PersistentFlags().StringP("config", "c", "", "path to the deployer configuration file to use")
	rootCmd.PersistentFlags().StringP("service_environment", "s", model.ServiceEnvironmentTest, "value of the MM_SERVICEENVIRONMENT variable. Valid values are production, test")

	deploymentCmd := &cobra.Command{
		Use:               "deployment",
		Short:             "Manage a load-test deployment",
		PersistentPreRun:  func(cmd *cobra.Command, _ []string) { setServiceEnv(cmd) },
		PersistentPostRun: func(_ *cobra.Command, _ []string) { os.Unsetenv("MM_SERVICEENVIRONMENT") },
	}

	deploymentCommands := []*cobra.Command{
		{
			Use:   "create",
			Short: "Create a new load-test deployment",
			RunE:  RunCreateCmdF,
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
		{
			Use:   "stop-db",
			Short: "Stops the DB cluster and syncs the changes.",
			RunE:  RunStopDBCmdF,
		},
		{
			Use:   "start-db",
			Short: "Starts the DB cluster and syncs the changes.",
			RunE:  RunStartDBCmdF,
		},
		{
			Use:   "db-info",
			Short: "Display info about the DB cluster.",
			RunE:  RunDBStatusCmdF,
		},
		{
			Use:   "version",
			Short: "Display version information of the deployed binaries",
			RunE:  RunVersionCmdF,
		},
	}

	destroyCmd := &cobra.Command{
		Use:     "destroy",
		Short:   "Destroy the current load-test deployment",
		PreRunE: checkDoNotDestroyMetricsInstanceFlag,
		RunE:    RunDestroyCmdF,
	}
	destroyCmd.Flags().Bool("do-not-destroy-metrics-instance", false, "Destroy everything but the metrics instance, so you can still analyze already stored data.")
	deploymentCommands = append(deploymentCommands, destroyCmd)

	deploymentCmd.AddCommand(deploymentCommands...)
	rootCmd.AddCommand(deploymentCmd)

	loadtestCmd := &cobra.Command{
		Use:               "loadtest",
		Short:             "Manage the load-test",
		PersistentPreRun:  func(cmd *cobra.Command, _ []string) { setServiceEnv(cmd) },
		PersistentPostRun: func(_ *cobra.Command, _ []string) { os.Unsetenv("MM_SERVICEENVIRONMENT") },
	}

	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset and re-initialize target instance database",
		RunE:  RunResetCmdF,
	}
	resetCmd.Flags().Bool("confirm", false, "Confirm you really want to reset the database and re-initialize it.")

	ltStartCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the coordinator in the current load-test deployment",
		RunE:  RunLoadTestStartCmdF,
	}
	ltStartCmd.Flags().Bool("sync", false, "Changes the command to not return until the test has finished, and then stops the DB after that")

	loadtestComands := []*cobra.Command{
		ltStartCmd,
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
		{
			Use:   "inject actionId",
			Short: "Injects the action into the current load-test",
			RunE:  RunInjectActionCmdF,
			Args:  cobra.ExactArgs(1),
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

			config, err := getConfig(cmd)
			if err != nil {
				return err
			}

			t, err := terraform.New("", config)
			if err != nil {
				return fmt.Errorf("failed to create terraform engine: %w", err)
			}

			runCmd, _ := cmd.Flags().GetString("run")
			if runCmd != "" {
				return t.RunSSHCommand(args[0], strings.Split(runCmd, " "))
			}
			return t.OpenSSHFor(args[0])
		},
	}

	sshListCmd := &cobra.Command{
		Use:   "list",
		Short: "lists available resources",
		RunE:  RunSSHListCmdF,
		Args:  cobra.NoArgs,
	}
	sshCmd.Flags().StringP("run", "r", "", "command to run")
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

			config, err := getConfig(cmd)
			if err != nil {
				return err
			}

			t, err := terraform.New("", config)
			if err != nil {
				return fmt.Errorf("failed to create terraform engine: %w", err)
			}
			return t.OpenBrowserFor(args[0])
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
		Use:               "comparison",
		Short:             "Manage fully automated load-test comparisons environments",
		PersistentPreRun:  func(cmd *cobra.Command, _ []string) { setServiceEnv(cmd) },
		PersistentPostRun: func(_ *cobra.Command, _ []string) { os.Unsetenv("MM_SERVICEENVIRONMENT") },
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

	collectComparisonCmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect logs and configurations from all deployments",
		RunE:  CollectComparisonCmdF,
	}

	destroyComparisonCmd := &cobra.Command{
		Use:     "destroy",
		Short:   "Destroy the current load-test comparison environment",
		PreRunE: checkDoNotDestroyMetricsInstanceFlag,
		RunE:    DestroyComparisonCmdF,
	}
	destroyComparisonCmd.Flags().Bool("do-not-destroy-metrics-instance", false, "Destroy everything but the metrics instance, so you can still analyze already stored data.")

	comparisonCmd.AddCommand(runComparisonCmd, destroyComparisonCmd, collectComparisonCmd)
	rootCmd.AddCommand(comparisonCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
