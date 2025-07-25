// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"os"

	"github.com/spf13/cobra"
)

func MakeGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate LDIF files for LDAP user and group management",
	}

	cmd.AddCommand(MakeGenerateUsersCommand())
	return cmd
}

func MakeLdapCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "ltldap",
		SilenceUsage: true,
		Short:        "Utilities to interact with OpenLDAP deployments",
	}
	rootCmd.PersistentFlags().StringP("config", "c", "", "path to the deployer configuration file to use")

	return rootCmd
}

func main() {
	rootCmd := MakeLdapCommand()
	commands := []*cobra.Command{
		MakeGenerateCommand(),
	}
	rootCmd.AddCommand(commands...)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
