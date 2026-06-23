// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/logger"
	"github.com/mattermost/mattermost-load-test-ng/version"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/spf13/cobra"
)

func runServerCmdF(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	portOverride, _ := cmd.Flags().GetInt("port")

	cfg, err := ReadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Override port if specified
	if portOverride > 0 {
		cfg.ServerConfig.Port = portOverride
	}

	// Initialize logger
	logger.Init(&cfg.LogSettings)

	versionInfo := version.GetInfo()
	mlog.Info("Mock LibreTranslate server starting",
		mlog.Int("port", cfg.ServerConfig.Port),
		mlog.String("host", cfg.ServerConfig.Host),
		mlog.Bool("latency_enabled", cfg.LatencyConfig.Enabled),
		mlog.Int("base_latency_ms", cfg.LatencyConfig.BaseLatencyMs),
		mlog.Int("max_latency_ms", cfg.LatencyConfig.MaxLatencyMs),
		mlog.String("commit", versionInfo.Commit),
		mlog.String("buildTime", versionInfo.BuildTime.Format(time.RFC3339)),
		mlog.Bool("modified", versionInfo.Modified),
		mlog.String("goVersion", versionInfo.GoVersion))

	srv := newServer(cfg)
	addr := fmt.Sprintf("%s:%d", cfg.ServerConfig.Host, cfg.ServerConfig.Port)

	mlog.Info("Server listening", mlog.String("address", addr))
	return http.ListenAndServe(addr, srv.setupRouter())
}

func main() {
	rootCmd := &cobra.Command{
		Use:          "ltlibre",
		Short:        "Start mock LibreTranslate server for load testing",
		Long:         "A mock LibreTranslate server that simulates the Libre translation API with configurable latency based on payload size.",
		SilenceUsage: true,
		RunE:         runServerCmdF,
	}

	rootCmd.PersistentFlags().StringP("config", "c", "", "Path to config file")
	rootCmd.PersistentFlags().IntP("port", "p", 0, "Port to listen on (overrides config)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
