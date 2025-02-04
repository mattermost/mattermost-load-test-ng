package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	gocloak "github.com/Nerzal/gocloak/v13"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/spf13/cobra"
)

func migrateUser(worker *workerConfig, user *model.User) error {
	defer worker.operationsWg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	defer cancel()

	kcUserID, err := worker.keycloakClient.CreateUser(ctx, worker.keycloakToken.AccessToken, worker.keycloakRealm, gocloak.User{
		Username:      &user.Username,
		Email:         &user.Email,
		Enabled:       gocloak.BoolP(true),
		EmailVerified: &user.EmailVerified,
		Credentials: &[]gocloak.CredentialRepresentation{
			{
				Temporary: gocloak.BoolP(false),
				Type:      gocloak.StringP("password"),
				Value:     gocloak.StringP(worker.userPassword),
			},
		},
	})
	if err != nil {
		// Check if the error is due to the user already existing in keycloak
		if apiErr, ok := err.(*gocloak.APIError); ok && apiErr.Code == 409 {
			mlog.Debug("user already exists in keycloak", mlog.String("username", user.Username))

			// Return early if we are not forcing the migration of all the users
			if !worker.forceMigrate {
				return nil
			}

			// If the `--force-migrate` flag is set, we need to get the user ID of the already existing user from
			// keycloak to set up the auth data of the mattermost user.
			mlog.Debug("trying to fetch the user from keycloak to update the mattermost user", mlog.String("username", user.Username))
			users, err := worker.keycloakClient.GetUsers(
				ctx,
				worker.keycloakToken.AccessToken,
				worker.keycloakRealm,
				gocloak.GetUsersParams{
					Username: &user.Username,
					Exact:    gocloak.BoolP(true),
				},
			)
			if err != nil {
				mlog.Error("failed to get user from keycloak", mlog.String("err", err.Error()))
				return fmt.Errorf("failed to get user from keycloak: %w", err)
			}

			if len(users) != 1 {
				return fmt.Errorf("got %d users in keycloak for user %s", len(users), user.Username)
			}

			// This should not happen, but just in case...
			if users[0] == nil || users[0].ID == nil {
				return fmt.Errorf("somehow keycloak returned incorrect data for user %s (nil values)", user.Username)
			}

			kcUserID = *users[0].ID
		} else {
			mlog.Error("failed to create user in keycloak", mlog.String("err", err.Error()))
			return fmt.Errorf("failed to create user in keycloak: %w", err)
		}
	}

	_, _, err = worker.mmClient.UpdateUserAuth(ctx, user.Id, &model.UserAuth{
		AuthData:    model.NewPointer(kcUserID),
		AuthService: model.UserAuthServiceSaml,
	})
	if err != nil {
		mlog.Error("failed to update user in mattermost", mlog.String("err", err.Error()))

		// Delete keycloak user to maintain consistency
		if deleteErr := worker.keycloakClient.DeleteUser(ctx, worker.keycloakToken.AccessToken, worker.keycloakRealm, kcUserID); deleteErr != nil {
			mlog.Error("failed to delete user in keycloak", mlog.String("err", deleteErr.Error()))
		}

		return err
	}
	mlog.Info("migrated user", mlog.String("username", user.Username))

	return nil
}

func userTxtWriter(usersTxtChan chan string, doneChan chan struct{}, workersWg *sync.WaitGroup, userPassword string) {
	workersWg.Add(1)
	defer workersWg.Done()

	usersTxtFile, err := os.OpenFile("users.txt", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		mlog.Error("failed to open users.txt file")
		return
	}
	defer usersTxtFile.Close()

	for {
		select {
		case email := <-usersTxtChan:
			_, err := usersTxtFile.Write([]byte(model.UserAuthServiceSaml + ":" + email + " " + userPassword + "\n"))
			if err != nil {
				mlog.Error("failed to write to users.txt", mlog.String("err", err.Error()))
				return
			}

		case <-doneChan:
			return
		}
	}
}

type workerConfig struct {
	workerNumber     int
	usersChan        chan *model.User
	operationsWg     *sync.WaitGroup
	workersWg        *sync.WaitGroup
	mmClient         *model.Client4
	keycloakClient   *gocloak.GoCloak
	keycloakToken    *gocloak.JWT
	keycloakRealm    string
	userPassword     string
	deploymentConfig *deployment.Config
	doneChan         chan struct{}
	forceMigrate     bool
}

func migrateMattermostUsersToKeycloak(worker *workerConfig) {
	refreshTokenTicker := time.NewTicker(30 * time.Second)
	defer refreshTokenTicker.Stop()
	defer worker.workersWg.Done()

	for {
		select {
		case user := <-worker.usersChan:
			if err := migrateUser(worker, user); err != nil {
				mlog.Error("failed to migrate user", mlog.String("err", err.Error()))
			}
		case <-refreshTokenTicker.C:
			ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)

			mlog.Info("refreshing keycloak token", mlog.Int("worker", worker.workerNumber))
			var err error
			worker.keycloakToken, err = worker.keycloakClient.LoginAdmin(
				ctx,
				worker.deploymentConfig.ExternalAuthProviderSettings.KeycloakAdminUser,
				worker.deploymentConfig.ExternalAuthProviderSettings.KeycloakAdminPassword,
				"master", // TODO: Allow specifying the master realm
			)
			if err != nil {
				cancel()
				mlog.Error("failed to refresh keycloak token", mlog.String("err", err.Error()))
				close(worker.doneChan)
			}

			cancel()
		case <-worker.doneChan:
			return
		}
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

	siteURL := deploymentConfig.SiteURL

	mattermostHost, err := cmd.Flags().GetString("mattermost-host")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}
	if mattermostHost != "" {
		siteURL = mattermostHost
	}

	mmClient := model.NewAPIv4Client("http://" + siteURL)

	_, _, err = mmClient.Login(cmd.Context(), deploymentConfig.AdminEmail, deploymentConfig.AdminPassword)
	if err != nil {
		return fmt.Errorf("failed to login to mattermost: %w", err)
	}

	var keycloakHost string
	keycloakHost, err = cmd.Flags().GetString("keycloak-host")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}

	// Use the terraform output keycloak host if a manual one is not provided. Useful for development.
	if keycloakHost == "" {
		t, err := terraform.New("", *deploymentConfig)
		if err != nil {
			return fmt.Errorf("failed to create terraform client: %w", err)
		}
		terraformOutput, err := t.Output()
		if err != nil {
			return fmt.Errorf("failed to get terraform output: %w", err)
		}
		if terraformOutput.KeycloakServer.GetConnectionDNS() == "" {
			return fmt.Errorf("keycloak database cluster not found in terraform output")
		}
		keycloakHost = terraformOutput.KeycloakServer.GetConnectionDNS()
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

	forceMigrate, err := cmd.Flags().GetBool("force-migrate")
	if err != nil {
		return fmt.Errorf("failed to read flag: %w", err)
	}

	doneChan := make(chan struct{})
	workers := runtime.NumCPU()
	if workers > 4 {
		workers = 4
	}
	usersTxtChan := make(chan string)
	usersChan := make(chan *model.User)

	operationsWg := &sync.WaitGroup{}
	workersWg := &sync.WaitGroup{}

	// Worker to write to the users.txt file
	go userTxtWriter(usersTxtChan, doneChan, workersWg, userPassword)

	// Workers to run the sync and call the MM and Keycloak APIs
	for i := 0; i < workers; i++ {
		workersWg.Add(1)

		workerNumber := i + 1

		go migrateMattermostUsersToKeycloak(&workerConfig{
			workerNumber:     workerNumber,
			usersChan:        usersChan,
			workersWg:        workersWg,
			operationsWg:     operationsWg,
			mmClient:         mmClient,
			keycloakClient:   keycloakClient,
			keycloakToken:    keycloakToken,
			keycloakRealm:    keycloakRealm,
			userPassword:     userPassword,
			deploymentConfig: deploymentConfig,
			doneChan:         doneChan,
			forceMigrate:     forceMigrate,
		})
	}

	page := 0
	perPage := 100

	for {
		mlog.Info("fetching mattermost users", mlog.Int("page", page), mlog.Int("per_page", perPage))
		users, _, err := mmClient.GetUsers(ctx, page, perPage, "")
		if err != nil {
			return fmt.Errorf("failed to get users from mattermost: %w", err)
		}

		for _, user := range users {
			u := *user

			// Skip bots
			if user.IsBot {
				continue
			}

			// Write the user to the users.txt file
			if !dryRun {
				usersTxtChan <- user.Email
			}

			// Already migrated
			if user.AuthService == model.UserAuthServiceSaml && !forceMigrate {
				continue
			}

			if !dryRun {
				operationsWg.Add(1)
				usersChan <- &u
			} else {
				mlog.Info("dry-run: would migrate user", mlog.String("username", user.Username))
			}
		}

		if len(users) < perPage {
			break
		}

		page++
	}

	operationsWg.Wait()
	close(doneChan)
	workersWg.Wait() // Wait for all goroutines to finish after all operations are done

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
