// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-load-test-ng/cmd/loadtest/config"
	"os"

	"github.com/mattermost/mattermost-load-test-ng/example"

	"github.com/spf13/cobra"
)

func RunExampleCmdF(cmd *cobra.Command, args []string) error {
	lt := example.New("http://localhost:8065")
	return lt.Run(4)
}

func main() {
	rootCmd := MakeLoadTestCommand()

	commands := []*cobra.Command{
		{
			Use:    "example",
			Short:  "Run example implementation",
			RunE:   RunExampleCmdF,
			PreRun: config.SetupLoadTest,
		},
		MakeInitCommand(),
		MakeServerCommand(),
	}

	rootCmd.AddCommand(commands...)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
