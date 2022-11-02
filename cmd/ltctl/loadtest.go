// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"math"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"

	"github.com/spf13/cobra"
)

func getUsersCount(helper *prometheus.Helper) (int, error) {
	query := "sum(mattermost_http_websockets_total)"
	value, err := helper.VectorFirst(query)
	if err != nil {
		return 0, fmt.Errorf("failed to query Prometheus with %q: %w", query, err)
	}
	return int(value), nil
}

func getErrorsInfo(helper *prometheus.Helper, startTime time.Time) (map[string]int64, error) {
	timeRange := int(time.Since(startTime).Round(time.Second).Seconds())
	queries := []struct {
		description string
		query       string
	}{
		{
			"Timeouts",
			fmt.Sprintf("sum(increase(loadtest_http_timeouts_total[%ds]))", timeRange),
		},
		{
			"HTTP 5xx",
			fmt.Sprintf("sum(increase(loadtest_http_errors_total{status_code=~\"5..\"}[%ds]))", timeRange),
		},
		{
			"HTTP 4xx",
			fmt.Sprintf("sum(increase(loadtest_http_errors_total{status_code=~\"4..\"}[%ds]))", timeRange),
		},
	}

	info := make(map[string]int64, len(queries)+1)
	for _, q := range queries {
		value, err := helper.VectorFirst(q.query)
		if err != nil {
			fmt.Printf("failed to query Prometheus with %q: %s\n", q.query, err.Error())
			continue
		}
		info[q.description] = int64(math.Round(value))
		info["total"] += int64(math.Round(value))
	}

	return info, nil
}

func RunLoadTestStartCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}
	return t.StartCoordinator(nil)
}

func RunLoadTestStopCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}
	_, err = t.StopCoordinator()
	return err
}

func printCoordinatorStatus(status coordinator.Status, errInfo map[string]int64, usersCount int) {
	fmt.Println("==================================================")
	fmt.Println("load-test status:")
	fmt.Println("")
	fmt.Println("State:", status.State)
	fmt.Println("Start time:", status.StartTime.Format(time.UnixDate))
	if status.State == coordinator.Done {
		fmt.Println("Stop time:", status.StopTime.Format(time.UnixDate))
		fmt.Println("Duration:", status.StopTime.Sub(status.StartTime).Round(time.Second))
	} else if status.State == coordinator.Running {
		fmt.Println("Running time:", time.Since(status.StartTime).Round(time.Second))
	}
	fmt.Println("Active users:", status.ActiveUsers)
	fmt.Println("Connected users:", usersCount)
	numErrs := status.NumErrors
	if numErrs < errInfo["total"] {
		numErrs = errInfo["total"]
	}
	fmt.Println("Number of errors:", numErrs)
	for k, v := range errInfo {
		if k != "total" {
			fmt.Printf("  - %s: %d (%.2f%%)\n", k, v, float64(v)/float64(numErrs)*100)
		}
	}
	if status.State == coordinator.Done {
		fmt.Println("Supported users:", status.SupportedUsers)
	}
	fmt.Println("==================================================")
}

func RunLoadTestStatusCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}

	status, err := t.GetCoordinatorStatus()
	if err != nil {
		return err
	}

	tfOutput, err := t.Output()
	if err != nil {
		return err
	}

	prometheusURL := fmt.Sprintf("http://%s:9090", tfOutput.MetricsServer.PublicIP)
	helper, err := prometheus.NewHelper(prometheusURL)
	if err != nil {
		return fmt.Errorf("failed to create prometheus.Helper: %w", err)
	}

	errInfo, err := getErrorsInfo(helper, status.StartTime)
	if err != nil {
		return err
	}

	usersCount, err := getUsersCount(helper)
	if err != nil {
		return err
	}

	printCoordinatorStatus(status, errInfo, usersCount)

	return nil
}
