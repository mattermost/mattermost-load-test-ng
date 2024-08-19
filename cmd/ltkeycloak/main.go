// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/spf13/cobra"

	gocloak "github.com/Nerzal/gocloak/v13"
)

const (
	keycloakMigratedGroup = "mattermost-migrated-users"

	// How much time to wait for a single operation to complete (all requests used during the
	// migration of an user)
	operationTimeout = 30 * time.Second
)

func RunSyncFromKeycloakCommandF(cmd *cobra.Command, _ []string) error {
	ltConfigPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}

	cfg, err := loadtest.ReadConfig(ltConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read load test configuration: %w", err)
	}

	// Guess where the deployment configuration is located
	deploymentConfig, err := deployment.ReadConfig(filepath.Join(filepath.Dir(ltConfigPath), "deployer"+filepath.Ext(ltConfigPath)))
	if err != nil {
		return fmt.Errorf("failed to read deployment configuration: %w", err)
	}

	var keycloakHost string
	keycloakHost, err = cmd.Flags().GetString("keycloak-host")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}

	// Use the terraform output terraform host if a manual one is not provided. Useful for development.
	if keycloakHost == "" {
		t, err := terraform.New("", *deploymentConfig)
		if err != nil {
			return fmt.Errorf("failed to create terraform client: %w", err)
		}
		terraformOutput, err := t.Output()
		if err != nil {
			return fmt.Errorf("failed to get terraform output: %w", err)
		}
		if len(terraformOutput.KeycloakDatabaseCluster.Endpoints) == 0 {
			return fmt.Errorf("keycloak database cluster not found in terraform output")
		}
		keycloakHost = terraformOutput.KeycloakDatabaseCluster.Endpoints[0]
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

	userPassword, err := cmd.Flags().GetString("set-user-password-to")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}

	updateExistingUsers, err := cmd.Flags().GetBool("update-existing-users")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}

	keycloakClient := gocloak.NewClient(keycloakHost)
	keycloakCtx, keycloakCtxCancel := context.WithTimeout(context.Background(), operationTimeout)
	defer keycloakCtxCancel()

	token, err := keycloakClient.LoginAdmin(
		keycloakCtx,
		deploymentConfig.ExternalAuthProviderSettings.KeycloakAdminUser,
		deploymentConfig.ExternalAuthProviderSettings.KeycloakAdminPassword,
		"master", // TODO: Allow specifying the master realm
	)
	if err != nil {
		return fmt.Errorf("failed to login to keycloak: %w", err)
	}

	if !dryRun {
		// Create group for migrated users if it does not exist
		keycloakCtx, keycloakCtxCancel := context.WithTimeout(context.Background(), operationTimeout)
		defer keycloakCtxCancel()

		_, err := keycloakClient.CreateGroup(keycloakCtx, token.AccessToken, deploymentConfig.ExternalAuthProviderSettings.KeycloakRealmName, gocloak.Group{
			Name: gocloak.StringP(keycloakMigratedGroup),
		})
		if err != nil {
			// Ignore if group already exists
			if apiErr, ok := err.(*gocloak.APIError); ok && apiErr.Code != 409 {
				return fmt.Errorf("failed to create group for user migrations in keycloak: %w", err)
			}
		}
	}

	start := 0
	perPage := 100
	for {
		requestCtx, cancel := context.WithTimeout(context.Background(), operationTimeout)
		defer cancel()
		users, err := keycloakClient.GetUsers(requestCtx, token.AccessToken, deploymentConfig.ExternalAuthProviderSettings.KeycloakRealmName, gocloak.GetUsersParams{
			First: &start,
			Max:   &perPage,
		})
		if err != nil {
			return fmt.Errorf("failed to get users from keycloak: %w", err)
		}

		for _, user := range users {
			// Check if user is already migrated
			if user.Groups != nil && slices.Contains(*user.Groups, keycloakMigratedGroup) {
				continue
			}

			// Check if user already exists in Mattermost
			mmUser, resp, err := mmClient.GetUserByUsername(requestCtx, *user.Username, "")
			if err != nil && resp.StatusCode != 404 {
				return fmt.Errorf("failed to get user from mattermost: %w", err)
			}

			// If user exists in Mattermost and we are not updating existing users, skip
			if mmUser != nil && !updateExistingUsers {
				continue
			}

			if !dryRun && false {
				if mmUser == nil {
					// If user does not exist in Mattermost, create it
					_, _, err = mmClient.CreateUser(requestCtx, &model.User{
						Username:    *user.Username,
						Email:       *user.Email,
						Password:    userPassword,
						AuthService: model.UserAuthServiceSaml,
						AuthData:    user.ID,
					})
					if err != nil {
						return fmt.Errorf("failed to create user in mattermost: %w", err)
					}
				} else {
					// If user exists in Mattermost, update it with the new auth data
					mmUser.AuthData = user.ID
					mmUser.AuthService = model.UserAuthServiceSaml
					mmUser.Password = ""
					_, _, err = mmClient.UpdateUser(requestCtx, mmUser)
					if err != nil {
						return fmt.Errorf("failed to update user in mattermost: %w", err)
					}
				}

				// Add user to migrated group in keycloak to avoid syncing them again
				if err := keycloakClient.AddUserToGroup(requestCtx, token.AccessToken, deploymentConfig.ExternalAuthProviderSettings.KeycloakRealmName, *user.ID, keycloakMigratedGroup); err != nil {
					return fmt.Errorf("failed to mark user migrated in keycloak: %w", err)
				}
			}

			mlog.Info("migrated user", mlog.String("username", *user.Username))

			if len(users) == 0 {
				break
			}

			start += perPage
		}

		return nil
	}
}

func RunSyncFromMattermostCommandF(cmd *cobra.Command, _ []string) error {
	started := time.Now()

	ltConfigPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}

	// Guess where the deployment configuration is located
	deploymentConfig, err := deployment.ReadConfig(filepath.Join(filepath.Dir(ltConfigPath), "deployer"+filepath.Ext(ltConfigPath)))
	if err != nil {
		return fmt.Errorf("failed to read deployment configuration: %w", err)
	}

	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}

	mmClient := model.NewAPIv4Client("http://" + deploymentConfig.SiteURL)

	_, _, err = mmClient.Login(cmd.Context(), deploymentConfig.AdminEmail, deploymentConfig.AdminPassword)
	if err != nil {
		return fmt.Errorf("failed to login to mattermost: %w", err)
	}

	var keycloakHost string
	keycloakHost, err = cmd.Flags().GetString("keycloak-host")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}

	// Use the terraform output terraform host if a manual one is not provided. Useful for development.
	if keycloakHost == "" {
		t, err := terraform.New("", *deploymentConfig)
		if err != nil {
			return fmt.Errorf("failed to create terraform client: %w", err)
		}
		terraformOutput, err := t.Output()
		if err != nil {
			return fmt.Errorf("failed to get terraform output: %w", err)
		}
		if terraformOutput.KeycloakServer.PublicDNS == "" {
			return fmt.Errorf("keycloak database cluster not found in terraform output")
		}
		keycloakHost = terraformOutput.KeycloakServer.PublicDNS
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

	keycloakToken, err := keycloakClient.LoginAdmin(
		ctx,
		deploymentConfig.ExternalAuthProviderSettings.KeycloakAdminUser,
		deploymentConfig.ExternalAuthProviderSettings.KeycloakAdminPassword,
		"master", // TODO: Allow specifying the master realm
	)
	if err != nil {
		return fmt.Errorf("failed to refresh keycloak token: %w", err)
	}

	doneChan := make(chan struct{})
	workers := runtime.NumCPU()
	if workers > 4 {
		workers = 4
	}
	usersTxtChan := make(chan string, workers*2)
	usersChan := make(chan *model.User, workers*2)

	wg := sync.WaitGroup{}

	// Workers to run the sync and call the MM and Keycloak APIs
	for i := 0; i < workers; i++ {
		workerNumber := i + 1
		go func() {
			refreshTokenTicker := time.NewTicker(30 * time.Second)
			defer refreshTokenTicker.Stop()

			for {
				select {
				case user := <-usersChan:
					kcUserID, err := keycloakClient.CreateUser(ctx, keycloakToken.AccessToken, keycloakRealm, gocloak.User{
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
							mlog.Debug("user already exists in keycloak", mlog.String("username", user.Username))
							continue
						}

						mlog.Error("failed to create user in keycloak", mlog.String("err", err.Error()))
						continue
					}

					user.AuthData = model.NewString(kcUserID)
					user.AuthService = model.UserAuthServiceSaml
					user.Password = ""

					_, _, err = mmClient.UpdateUser(ctx, user)
					if err != nil {
						mlog.Error("failed to update user in mattermost", mlog.String("err", err.Error()))
						continue
					}
					mlog.Info("migrated user", mlog.String("username", user.Username))
					wg.Done()

				case <-refreshTokenTicker.C:
					mlog.Info("refreshing keycloak token", mlog.Int("worker", workerNumber))
					keycloakToken, err = keycloakClient.LoginAdmin(
						ctx,
						deploymentConfig.ExternalAuthProviderSettings.KeycloakAdminUser,
						deploymentConfig.ExternalAuthProviderSettings.KeycloakAdminPassword,
						"master", // TODO: Allow specifying the master realm
					)
					if err != nil {
						mlog.Error("failed to refresh keycloak token", mlog.String("err", err.Error()))
						close(doneChan)
						panic(err)
					}

				case <-doneChan:
					return
				}
			}
		}()
	}

	// File writter for users.txt
	go func() {
		usersTxtFile, err := os.OpenFile("users.txt", os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			mlog.Error("failed to open users.txt file")
			panic(err)
		}
		defer usersTxtFile.Close()

		for {
			select {
			case email := <-usersTxtChan:
				_, err := usersTxtFile.Write([]byte(model.UserAuthServiceSaml + ":" + email + " " + userPassword + "\n"))
				if err != nil {
					mlog.Error("failed to write to users.txt", mlog.String("err", err.Error()))
				}

			case <-doneChan:
				return
			}
		}
	}()

	page := 0
	perPage := 100
	for {
		mlog.Info("fetching mattermost users", mlog.Int("page", page), mlog.Int("per_page", perPage))
		users, _, err := mmClient.GetUsers(ctx, page, perPage, "")
		if err != nil {
			return fmt.Errorf("failed to get users from mattermost: %w", err)
		}

		for _, user := range users {
			// Skip bots
			if user.IsBot {
				continue
			}

			// Write the user to the users.txt file
			usersTxtChan <- user.Email

			// Already migrated
			if user.AuthService == model.UserAuthServiceSaml {
				continue
			}

			if !dryRun {
				wg.Add(1)
				usersChan <- user
			} else {
				mlog.Info("dry-run: would migrate user", mlog.String("username", user.Username))
			}
		}

		if len(users) == 0 {
			break
		}

		page++
	}

	wg.Wait()
	close(doneChan)

	finished := time.Now()

	mlog.Info("migration finished", mlog.Duration("duration", finished.Sub(started)))

	return nil
}

func MakeSyncFromMattermostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from_mattermost",
		Short: "Migrate data from Mattermost to Keycloak",
		RunE:  RunSyncFromMattermostCommandF,
	}

	cmd.Flags().String("set-user-password-to", "testpassword", "Set's the user password to the provided value")
	cmd.Flags().String("keycloak-realm", "master", "The Keycloak realm to migrate users to")

	return cmd
}

func MakeSyncFromKeycloakCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from_keycloak",
		Short: "Migrate data from Keycloak to Mattermost",
		RunE:  RunSyncFromKeycloakCommandF,
	}

	cmd.Flags().Bool("update-existing-users", false, "Update existing users in Mattermost")
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
	cmd.PersistentFlags().BoolP("dry-run", "", false, "perform a dry run without making any changes")

	cmd.AddCommand(MakeSyncFromMattermostCommand())
	cmd.AddCommand(MakeSyncFromKeycloakCommand())
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
