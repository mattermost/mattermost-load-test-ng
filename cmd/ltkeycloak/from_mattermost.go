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

func userTxtWriter(usersTxtChan chan string, doneChan chan struct{}, errorsChan chan error, workersWg *sync.WaitGroup, userPassword string) {
	workersWg.Add(1)
	defer workersWg.Done()

	usersTxtFile, err := os.OpenFile("users.txt", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		mlog.Error("failed to open users.txt file")
		errorsChan <- err
		return
	}
	defer usersTxtFile.Close()

	for {
		select {
		case email := <-usersTxtChan:
			_, err := usersTxtFile.Write([]byte(model.UserAuthServiceSaml + ":" + email + " " + userPassword + "\n"))
			if err != nil {
				mlog.Error("failed to write to users.txt", mlog.String("err", err.Error()))
				errorsChan <- err
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
	errorsChan       chan error
	mmClient         *model.Client4
	keycloakClient   *gocloak.GoCloak
	keycloakToken    *gocloak.JWT
	keycloakRealm    string
	userPassword     string
	deploymentConfig *deployment.Config
	doneChan         chan struct{}
}

func migrateMattermostUsersToKeycloak(worker *workerConfig) {
	refreshTokenTicker := time.NewTicker(30 * time.Second)
	defer refreshTokenTicker.Stop()
	defer worker.workersWg.Done()

	for {
		select {
		case user := <-worker.usersChan:
			ctx, cancel := context.WithTimeout(context.Background(), operationTimeout)

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
				cancel()
				worker.operationsWg.Done()

				// Ignore already existing users
				if apiErr, ok := err.(*gocloak.APIError); ok && apiErr.Code == 409 {
					mlog.Debug("user already exists in keycloak", mlog.String("username", user.Username))
					continue
				}

				mlog.Error("failed to create user in keycloak", mlog.String("err", err.Error()))
				worker.errorsChan <- err
				return
			}

			user.AuthData = model.NewPointer(kcUserID)
			user.AuthService = model.UserAuthServiceSaml
			user.Password = ""

			_, _, err = worker.mmClient.UpdateUser(ctx, user)
			if err != nil {
				cancel()
				mlog.Error("failed to update user in mattermost", mlog.String("err", err.Error()))

				// Delete keycloak user to maintain consistency
				if deleteErr := worker.keycloakClient.DeleteUser(ctx, worker.keycloakToken.AccessToken, worker.keycloakRealm, kcUserID); deleteErr != nil {
					mlog.Error("failed to delete user in keycloak", mlog.String("err", deleteErr.Error()))
				}

				worker.errorsChan <- err
				worker.operationsWg.Done()
				return
			}
			mlog.Info("migrated user", mlog.String("username", user.Username))
			worker.operationsWg.Done()
			cancel()
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
	errorsChan := make(chan error, workers+1)
	usersTxtChan := make(chan string)
	usersChan := make(chan *model.User)

	operationsWg := &sync.WaitGroup{}
	workersWg := &sync.WaitGroup{}

	// Worker to write to the users.txt file
	go userTxtWriter(usersTxtChan, doneChan, errorsChan, workersWg, userPassword)

	// Workers to run the sync and call the MM and Keycloak APIs
	for i := 0; i < workers; i++ {
		workersWg.Add(1)

		workerNumber := i + 1

		go migrateMattermostUsersToKeycloak(&workerConfig{
			workerNumber:     workerNumber,
			usersChan:        usersChan,
			workersWg:        workersWg,
			operationsWg:     operationsWg,
			errorsChan:       errorsChan,
			mmClient:         mmClient,
			keycloakClient:   keycloakClient,
			keycloakToken:    keycloakToken,
			keycloakRealm:    keycloakRealm,
			userPassword:     userPassword,
			deploymentConfig: deploymentConfig,
			doneChan:         doneChan,
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
			// Skip bots
			if user.IsBot {
				continue
			}

			// Write the user to the users.txt file
			if !dryRun {
				usersTxtChan <- user.Email
			}

			// Already migrated
			if user.AuthService == model.UserAuthServiceSaml {
				continue
			}

			if !dryRun {
				operationsWg.Add(1)
				usersChan <- user
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
