// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/mattermost/mattermost-load-test-ng/api"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/spf13/cobra"
)

func RunServerCmdF(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")

	logger.Init(&logger.Settings{
		EnableConsole: true,
		ConsoleLevel:  "ERROR",
		ConsoleJson:   false,
		EnableFile:    true,
		FileLevel:     "INFO",
		FileJson:      true,
		FileLocation:  "ltapi.log",
	})

	mlog.Info("API server started, listening on", mlog.Int("port", port))
	return http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), api.SetupAPIRouter())
}

func main() {
	rootCmd := &cobra.Command{
		Use:          "ltapi",
		Short:        "Start load-test API server",
		SilenceUsage: true,
		RunE:         RunServerCmdF,
	}
	rootCmd.PersistentFlags().IntP("port", "p", 4000, "Port to listen on")
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
