// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/report"

	"github.com/spf13/cobra"
)

func RunGenerateReportCmdF(cmd *cobra.Command, args []string) error {
	st, err := cmd.Flags().GetInt64("start-time")
	if err != nil {
		return err
	}

	et, err := cmd.Flags().GetInt64("end-time")
	if err != nil {
		return err
	}

	file, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	label, err := cmd.Flags().GetString("label")
	if err != nil {
		return err
	}

	startTime := time.Unix(st, 0)
	endTime := time.Unix(et, 0)
	if endTime.Before(startTime) {
		return errors.New("end-time is before start-time")
	}

	if _, err := os.Stat(file); err == nil {
		fmt.Printf("File %s exists. Overwrite ? (Y/n) ", file)
		var confirm string
		fmt.Scanln(&confirm)
		if !regexp.MustCompile(`(?i)^(y|yes)?$`).MatchString(confirm) {
			return errors.New("incorrect response")
		}
	}

	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	t := terraform.New(nil)
	output, err := t.Output()
	if err != nil {
		return fmt.Errorf("could not parse output: %w", err)
	}

	helper, err := prometheus.NewHelper("http://" + output.MetricsServer.Value.PublicIP + ":9090")
	if err != nil {
		return fmt.Errorf("failed to create prometheus.Helper: %w", err)
	}

	g := report.New(label, helper)
	data, err := g.Generate(startTime, endTime)
	if err != nil {
		return fmt.Errorf("error while generating report: %w", err)
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(data)
	if err != nil {
		return fmt.Errorf("error while encoding report to JSON: %w", err)
	}

	return nil
}

func RunCompareReportCmdF(cmd *cobra.Command, args []string) error {
	err := cobra.MinimumNArgs(2)(cmd, args)
	if err != nil {
		return err
	}
	var reports []report.Report
	for _, arg := range args {
		r, err := report.Load(arg)
		if err != nil {
			return fmt.Errorf("error loading report %s: %w", arg, err)
		}
		reports = append(reports, r)
	}

	genGraph, err := cmd.Flags().GetBool("graph")
	if err != nil {
		return err
	}

	file, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	target := os.Stdout
	if file != "" {
		target, err = os.Create(file)
		if err != nil {
			return err
		}
		defer target.Close()
	}

	report.Compare(target, genGraph, reports...)
	return nil
}
