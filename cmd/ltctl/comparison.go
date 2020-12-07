// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"io/ioutil"

	"github.com/mattermost/mattermost-load-test-ng/comparison"

	"github.com/spf13/cobra"
)

func printResults(results []comparison.Result) error {
	for i, res := range results {
		fmt.Println("==================================================")
		fmt.Println("Comparison result:")
		if res.Report != "" {
			filename := fmt.Sprintf("report_%d.md", i)
			if err := ioutil.WriteFile(filename, []byte(res.Report), 0660); err != nil {
				return err
			}
			fmt.Printf("Report: %s\n", filename)
		}
		if res.DashboardURL != "" {
			fmt.Printf("Grafana Dashboard: %s\n", res.DashboardURL)
		}
		for _, ltRes := range res.LoadTests {
			fmt.Printf("%s:\n", ltRes.Label)
			fmt.Printf("  Type: %s\n", ltRes.Config.Type)
			fmt.Printf("  DB Engine: %s\n", ltRes.Config.DBEngine)
			if ltRes.Config.Type == comparison.LoadTestTypeBounded {
				fmt.Printf("  Duration: %s\n", ltRes.Config.Duration)
				fmt.Printf("  Users: %d\n", ltRes.Config.NumUsers)
			} else if ltRes.Config.Type == comparison.LoadTestTypeUnbounded {
				fmt.Printf("  Supported Users: %d\n", ltRes.Status.SupportedUsers)
			}
			fmt.Printf("  Errors: %d\n", ltRes.Status.NumErrors)
		}
		fmt.Printf("==================================================\n\n")
	}
	return nil
}

func RunComparisonCmdF(cmd *cobra.Command, args []string) error {
	deployerConfig, err := getConfig(cmd)
	if err != nil {
		return err
	}

	configFilePath, _ := cmd.Flags().GetString("comparison-config")
	cfg, err := comparison.ReadConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read comparison config: %w", err)
	}

	cmp, err := comparison.New(cfg, deployerConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize comparison object: %w", err)
	}

	output, err := cmp.Run()
	if err != nil {
		return fmt.Errorf("failed to run comparisons: %w", err)
	}

	if err := printResults(output.Results); err != nil {
		return err
	}

	return nil
}

func DestroyComparisonCmdF(cmd *cobra.Command, args []string) error {
	deployerConfig, err := getConfig(cmd)
	if err != nil {
		return err
	}

	configFilePath, _ := cmd.Flags().GetString("comparison-config")
	cfg, err := comparison.ReadConfig(configFilePath)
	if err != nil {
		return fmt.Errorf("failed to read comparison config: %w", err)
	}

	cmp, err := comparison.New(cfg, deployerConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize comparison object: %w", err)
	}

	return cmp.Destroy()
}
