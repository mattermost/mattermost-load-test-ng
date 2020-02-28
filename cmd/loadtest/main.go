// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
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
			PreRun: SetupLoadTest,
		},
		MakeInitCommand(),
		MakeServerCommand(),
		MakeEnvCommand(),
	}

	rootCmd.AddCommand(commands...)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
