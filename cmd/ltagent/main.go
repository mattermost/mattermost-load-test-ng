// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := MakeLoadTestCommand()
	commands := []*cobra.Command{
		MakeInitCommand(),
	}
	rootCmd.AddCommand(commands...)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
