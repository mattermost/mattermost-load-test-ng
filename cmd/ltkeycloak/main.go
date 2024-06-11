// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/spf13/cobra"

	gocloak "github.com/Nerzal/gocloak/v13"
)

const (
	requestTiemout = 30 * time.Second
)

func RunSyncFromMattermostCommandF(cmd *cobra.Command, _ []string) error {
	ltConfigPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}

	cfg, err := loadtest.ReadConfig(ltConfigPath)
	if err != nil {
		return err
	}

	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}

	mmClient := model.NewAPIv4Client(cfg.ConnectionConfiguration.ServerURL)

	_, _, err = mmClient.Login(cmd.Context(), cfg.ConnectionConfiguration.AdminEmail, cfg.ConnectionConfiguration.AdminPassword)
	if err != nil {
		return fmt.Errorf("failed to login to mattermost: %w", err)
	}

	keycloakHost, err := cmd.Flags().GetString("keycloak-host")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}

	keycloakUsername, err := cmd.Flags().GetString("keycloak-username")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}

	keycloakPassword, err := cmd.Flags().GetString("keycloak-password")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}

	keycloakRealm, err := cmd.Flags().GetString("keycloak-realm")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}

	userPassword, err := cmd.Flags().GetString("set-user-password-to")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}

	keycloakClient := gocloak.NewClient(keycloakHost)
	ctx := context.Background()
	token, err := keycloakClient.LoginAdmin(ctx, keycloakUsername, keycloakPassword, keycloakRealm)
	if err != nil {
		return fmt.Errorf("failed to login to keycloak: %w", err)
	}

	page := 0
	perPage := 100
	for {
		requestCtx, cancel := context.WithTimeout(cmd.Context(), requestTiemout)
		defer cancel()
		users, _, err := mmClient.GetUsers(requestCtx, page, perPage, "")
		if err != nil {
			return fmt.Errorf("failed to get users from mattermost: %w", err)
		}

		for _, user := range users {
			if user.AuthService == model.ServiceOpenid {
				continue
			}

			if !dryRun {

				kcUserID, err := keycloakClient.CreateUser(ctx, token.AccessToken, keycloakRealm, gocloak.User{
					Username:      &user.Username,
					Email:         &user.Email,
					Enabled:       gocloak.BoolP(true),
					EmailVerified: &user.EmailVerified,
					Credentials: &[]gocloak.CredentialRepresentation{
						{
							Temporary: gocloak.BoolP(false),
							Type:      gocloak.StringP("password"),
							Value:     gocloak.StringP(userPassword),
						},
					},
				})
				if err != nil {
					// Ignore already existing users
					if apiErr, ok := err.(*gocloak.APIError); ok && apiErr.Code == 409 {
						continue
					}

					return fmt.Errorf("failed to create user in keycloak: %w", err)
				}

				user.AuthData = model.NewString(kcUserID)
				user.AuthService = model.ServiceOpenid

				_, _, err = mmClient.UpdateUser(requestCtx, user)
				if err != nil {
					return fmt.Errorf("failed to update user in mattermost: %w", err)
				}

				slog.Info("migrated user", slog.String("username", user.Username))
			}
		}

		if len(users) == 0 {
			break
		}

		page++
	}

	return nil
}

func MakeSyncFromMattermostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from_mattermost",
		Short: "Migrate data from Mattermost to Keycloak",
		RunE:  RunSyncFromMattermostCommandF,
	}

	cmd.Flags().String("set-user-password-to", "testpassword", "Set's the user password to the provided value")

	return cmd
}

func MakeSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "sync",
		Short:             "Sync data between Mattermost and Keycloak",
		PersistentPostRun: func(_ *cobra.Command, _ []string) { os.Unsetenv("MM_SERVICEENVIRONMENT") },
	}

	cmd.PersistentFlags().StringP("keycloak-host", "", "http://localhost:8484", "keycloak host")
	cmd.PersistentFlags().StringP("keycloak-realm", "", "mattermost", "keycloak realm")
	cmd.PersistentFlags().StringP("keycloak-username", "", "admin", "keycloak username")
	cmd.PersistentFlags().StringP("keycloak-password", "", "admin", "keycloak password")
	cmd.PersistentFlags().BoolP("dry-run", "", false, "perform a dry run without making any changes")

	cmd.AddCommand(MakeSyncFromMattermostCommand())
	return cmd
}

func MakeKeylcoakCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "ltkeycloak",
		SilenceUsage: true,
		Short:        "Utilities to interact with Mattermost and Keycloak deployments",
	}
	rootCmd.PersistentFlags().StringP("config", "c", "", "path to the deployer configuration file to use")

	return rootCmd
}

func main() {
	rootCmd := MakeKeylcoakCommand()
	commands := []*cobra.Command{
		MakeSyncCommand(),
	}
	rootCmd.AddCommand(commands...)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
