package main

import (
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-load-test-ng/api"
	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/spf13/cobra"
)

func RunServerCmdF(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")

	mlog.Info("API server started, listening on", mlog.Int("port", port))
	return http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), api.SetupAPIRouter())
}

func MakeServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "server",
		Short:  "Start API agent",
		RunE:   RunServerCmdF,
		PreRun: SetupLoadTest,
	}
	cmd.PersistentFlags().IntP("port", "p", 4000, "Port to listen on")

	return cmd
}
