//go:generate go-bindata -nometadata -mode 0644 -pkg main -o ./bindata.go -prefix "assets/" assets/
// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"text/template"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/report"

	"github.com/spf13/cobra"
)

func RunGenerateReportCmdF(cmd *cobra.Command, args []string) error {
	err := cobra.MinimumNArgs(2)(cmd, args)
	if err != nil {
		return err
	}

	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	const layout = "2006-01-02 15:04:05"
	startTime, err := time.Parse(layout, args[0])
	if err != nil {
		return fmt.Errorf("start-time in incorrect format: %w", err)
	}

	endTime, err := time.Parse(layout, args[1])
	if err != nil {
		return fmt.Errorf("end-time in incorrect format: %w", err)
	}

	file, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	label, err := cmd.Flags().GetString("label")
	if err != nil {
		return err
	}

	if endTime.Before(startTime) {
		return errors.New("end-time is before start-time")
	}

	if _, err := os.Stat(file); err == nil {
		fmt.Printf("File %s exists. Overwrite ? (Y/n) ", file)
		var confirm string
		fmt.Scanln(&confirm)
		if !regexp.MustCompile(`(?i)^(y|yes)?$`).MatchString(confirm) {
			return nil
		}
	}

	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	promURL, err := cmd.Flags().GetString("prometheus-url")
	if err != nil {
		return err
	}

	if promURL == "" {
		t := terraform.New(nil)
		output, err := t.Output()
		if err != nil {
			return fmt.Errorf("could not parse output: %w", err)
		}
		promURL = "http://" + output.MetricsServer.PublicIP + ":9090"
	}

	helper, err := prometheus.NewHelper(promURL)
	if err != nil {
		return fmt.Errorf("failed to create prometheus.Helper: %w", err)
	}

	g := report.New(label, helper, config.Report)
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

func generateDashboard(reports []report.Report) error {
	dashboardFile, err := os.Create("dashboard.json")
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer dashboardFile.Close()

	baseReport := reports[0]
	newReport := reports[1]
	baseLabel := baseReport.Label
	newLabel := newReport.Label
	from := newReport.StartTime
	to := newReport.EndTime
	offset := newReport.StartTime.Sub(baseReport.StartTime)

	// We swap everything if it happens that the load-tests were done in the
	// inverse than expected order (next then base).
	if baseReport.StartTime.After(newReport.StartTime) {
		from = baseReport.StartTime
		to = baseReport.EndTime
		offset = baseReport.StartTime.Sub(newReport.StartTime)
		baseLabel, newLabel = newLabel, baseLabel
	}

	tmpl, err := template.New("").Parse(MustAssetString("comparison.tmpl.json"))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	data := map[string]interface{}{
		"title":     "Comparison - " + newReport.Label,
		"offset":    offset.Seconds(),
		"baseLabel": baseLabel,
		"newLabel":  newLabel,
		"from":      from.Format(time.RFC3339),
		"to":        to.Format(time.RFC3339),
	}
	if err := tmpl.Execute(dashboardFile, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
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

	if genGraph {
		if _, err := exec.LookPath("gnuplot"); err != nil {
			return fmt.Errorf("gnuplot is not installed. The --graph option requires it to be installed: %w", err)
		}
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

	genDashboard, err := cmd.Flags().GetBool("dashboard")
	if err != nil {
		return err
	}
	if genDashboard {
		if err := generateDashboard(reports); err != nil {
			return err
		}
	}

	return report.Compare(target, genGraph, reports...)
}
