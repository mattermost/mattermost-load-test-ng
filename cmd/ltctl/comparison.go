// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/comparison"

	"github.com/spf13/cobra"
)

func createArchive(inPath, outPath string) error {
	files, err := os.ReadDir(inPath)
	if err != nil {
		return err
	}

	zipFile, err := os.Create(filepath.Join(outPath,
		fmt.Sprintf("comparison_%d.zip", time.Now().Unix())))
	if err != nil {
		return err
	}
	defer zipFile.Close()
	wr := zip.NewWriter(zipFile)
	defer wr.Close()

	for _, file := range files {
		fwr, err := wr.Create(file.Name())
		if err != nil {
			return err
		}

		f, err := os.Open(filepath.Join(inPath, file.Name()))
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(fwr, f); err != nil {
			return err
		}
	}

	return nil
}

func getReportFilename(id int, res comparison.Result) string {
	ltConfig := res.LoadTests[0].Config
	name := fmt.Sprintf("%d_%s_%s", id, ltConfig.DBEngine, ltConfig.Type)
	if ltConfig.Type == comparison.LoadTestTypeBounded {
		name += fmt.Sprintf("_%d", ltConfig.NumUsers)
	}
	return fmt.Sprintf("report_%s.md", name)
}

func printResults(results []comparison.Result) {
	for i, res := range results {
		fmt.Println("==================================================")
		fmt.Println("Comparison result:")
		if res.Report != "" {
			fmt.Printf("Report: %s\n", getReportFilename(i, res))
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
}

func writeReports(results []comparison.Result, outPath string) error {
	for i, res := range results {
		if res.Report == "" {
			continue
		}
		filePath := filepath.Join(outPath, getReportFilename(i, res))
		if err := os.WriteFile(filePath, []byte(res.Report), 0660); err != nil {
			return err
		}
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

	if cfg.Output.GenerateGraphs {
		if _, err := exec.LookPath("gnuplot"); err != nil {
			return fmt.Errorf("gnuplot is not installed. The comparison command with generate graph option requires it to be installed: %w", err)
		}
	}

	outputPath, _ := cmd.Flags().GetString("output-dir")
	if outputPath != "" {
		cfg.Output.GraphsPath = outputPath
	}

	archivePath := outputPath
	archive, _ := cmd.Flags().GetBool("archive")
	if archive {
		dir, err := os.MkdirTemp("", "comparison")
		if err != nil {
			return fmt.Errorf("failed to create temp dir: %w", err)
		}
		defer os.RemoveAll(dir)
		cfg.Output.GraphsPath = dir
		outputPath = dir
	}

	cmp, err := comparison.New(cfg, &deployerConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize comparison object: %w", err)
	}

	output, err := cmp.Run()
	if err != nil {
		return fmt.Errorf("failed to run comparisons: %w", err)
	}

	if format, _ := cmd.Flags().GetString("format"); format == "json" {
		f, err := os.Create(filepath.Join(outputPath, "comparison.json"))
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		defer f.Close()

		if err := json.NewEncoder(f).Encode(output); err != nil {
			return fmt.Errorf("failed to encode results: %w", err)
		}
	} else {
		if err := writeReports(output.Results, outputPath); err != nil {
			return fmt.Errorf("failed to write reports: %w", err)
		}
		printResults(output.Results)
	}

	if archive {
		if err := createArchive(outputPath, archivePath); err != nil {
			return fmt.Errorf("failed to create archive: %w", err)
		}
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

	cmp, err := comparison.New(cfg, &deployerConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize comparison object: %w", err)
	}

	return cmp.Destroy()
}
