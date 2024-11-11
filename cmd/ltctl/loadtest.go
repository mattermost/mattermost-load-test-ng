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

const (
	dbAvailable = "available"
	dbStopped   = "stopped"
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
	config, err := getDeployerConfig(cmd)
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

	if status == dbStopped {
		if err := t.StartDB(); err != nil {
			return err
		}

		fmt.Println("=====================")
		fmt.Println("Looping until the DB is fully available. You can cancel the command and start the test after some time, or don't do anything and it will automatically start the test after the DB is ready")
		fmt.Println("=====================")
		// Now we loop until the DB is available.

		for {
			status, err := t.DBStatus()
			if err != nil {
				return fmt.Errorf("failed to get DB status: %w", err)
			}
			if status == dbAvailable {
				break
			}
			fmt.Println("Sleeping... ")
			time.Sleep(30 * time.Second)
		}
	} else if status != dbAvailable {
		fmt.Printf("The database isn't available at the moment. Its status is %q. Please wait until it has finished, and then try again. \n", status)
		return nil
	}

	isSync, err := cmd.Flags().GetBool("sync")
	if err != nil {
		return fmt.Errorf("unable to check -sync flag: %w", err)
	}

	// We simply return in async mode, which is the default.
	if !isSync {
		return t.StartCoordinator(nil, nil)
	}

	err = t.StartCoordinator(nil, nil)
	if err != nil {
		return fmt.Errorf("error in starting coordinator: %w", err)
	}

	// Now we keep checking the status of the coordinator until it's done.
	var coordStatus coordinator.Status
	for {
		coordStatus, err = t.GetCoordinatorStatus()
		if err != nil {
			return err
		}

		if coordStatus.State == coordinator.Done {
			fmt.Println("load-test has completed")
			break
		}

		fmt.Println("Sleeping ...")
		// Sleeping for 5 minutes gives 12 lines an hour.
		// For an avg unbounded test of 4-5 hours, it gives around 50 lines,
		// which should be acceptable.
		time.Sleep(5 * time.Minute)
	}

	// Now we stop the DB.
	return t.StopDB()
}

func RunLoadTestStopCmdF(cmd *cobra.Command, args []string) error {
	config, err := getDeployerConfig(cmd)
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
	config, err := getDeployerConfig(cmd)
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

func RunInjectActionCmdF(cmd *cobra.Command, args []string) error {
	config, err := getDeployerConfig(cmd)
	if err != nil {
		return err
	}

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}

	action := args[0]

	_, err = t.InjectAction(action)
	if err != nil {
		fmt.Println("Failed to inject action ", action, ": ", err)
		return err
	}

	fmt.Println("Action ", action, " injected successfully.")

	return nil
}
