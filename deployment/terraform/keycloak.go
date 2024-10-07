package terraform

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/assets"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func (t *Terraform) setupKeycloak(extAgent *ssh.ExtAgent) error {
	keycloakDir := "/opt/keycloak/keycloak-" + t.config.ExternalAuthProviderSettings.KeycloakVersion
	keycloakBinPath := filepath.Join(keycloakDir, "bin")

	mlog.Info("Configuring keycloak", mlog.String("host", t.output.KeycloakServer.PrivateIP))
	extraArguments := []string{}

	command := "start"
	if t.config.ExternalAuthProviderSettings.DevelopmentMode {
		command = "start-dev"
	}

	sshc, err := extAgent.NewClient(t.output.KeycloakServer.PrivateIP)
	if err != nil {
		return fmt.Errorf("error in getting ssh connection %w", err)
	}
	defer sshc.Close()

	// Check if the keycloak database exists
	result, err := sshc.RunCommand(`sudo -iu postgres psql -l | grep keycloak | wc -l`)
	if err != nil {
		return fmt.Errorf("failed to check keycloak database: %w", err)
	}

	if strings.TrimSpace(string(result)) == "0" {
		mlog.Info("keycloak database not found, creating it and associated role")

		// Upload and import keycloak initial SQL file``
		_, err := sshc.Upload(strings.NewReader(assets.MustAssetString("keycloak-database.sql")), "/var/lib/postgresql/keycloak-database.sql", true)
		if err != nil {
			return fmt.Errorf("failed to upload keycloak base database sql file: %w", err)
		}

		// Allow postgres user to read the file
		_, err = sshc.RunCommand("sudo chown postgres:postgres /var/lib/postgresql/keycloak-database.sql")
		if err != nil {
			return fmt.Errorf("failed to change permissions on keycloak database sql file: %w", err)
		}

		_, err = sshc.RunCommand(`sudo -iu postgres psql -v ON_ERROR_STOP=on -f /var/lib/postgresql/keycloak-database.sql`)
		if err != nil {
			return fmt.Errorf("failed to setup keycloak database: %w", err)
		}

		if t.config.ExternalAuthProviderSettings.KeycloakDBDumpURI != "" && t.config.ExternalAuthProviderSettings.KeycloakRealmFilePath == "" {
			err := t.IngestKeycloakDump()
			if err != nil {
				return fmt.Errorf("failed to ingest keycloak dump: %w", err)
			}
		}
	}

	// If no keycloak dump URI is provided, check if we should use a custom realm file or
	// proceed with the default one.
	if t.config.ExternalAuthProviderSettings.KeycloakDBDumpURI == "" {
		extraArguments = append(extraArguments, "--import-realm")
		keycloakRealmFile, err := assets.AssetString("mattermost-realm.json")
		if err != nil {
			return fmt.Errorf("failed to read default keycloak realm file: %w", err)
		}

		if t.config.ExternalAuthProviderSettings.KeycloakRealmFilePath != "" {
			mlog.Info("Using provided realm configuration")
			keycloakRealmFile = t.config.ExternalAuthProviderSettings.KeycloakRealmFilePath
		}

		_, err = sshc.Upload(
			strings.NewReader(keycloakRealmFile),
			filepath.Join(keycloakDir, "data/import/mattermost-realm.json"),
			true,
		)
		if err != nil {
			return fmt.Errorf("failed to upload default keycloak realm file: %w", err)
		}
	}

	keycloakEnvFileContents, err := fillConfigTemplate(keycloakEnvFileContents, map[string]string{
		"KeycloakAdminUser":     t.config.ExternalAuthProviderSettings.KeycloakAdminUser,
		"KeycloakAdminPassword": t.config.ExternalAuthProviderSettings.KeycloakAdminPassword,
		"KeycloakLogFilePath":   filepath.Join(keycloakDir, "data/log/keycloak.log"),
	})
	if err != nil {
		return fmt.Errorf("failed to fill keycloak.env file template: %w", err)
	}

	// Upload keycloak.env file
	_, err = sshc.Upload(strings.NewReader(keycloakEnvFileContents), "/etc/systemd/system/keycloak.env", true)
	if err != nil {
		return fmt.Errorf("failed to upload keycloak env file: %w", err)
	}

	// Parse keycloak service file template
	keycloakServiceFileContents, err := fillConfigTemplate(keycloakServiceFileContents, map[string]string{
		"KeycloakVersion": t.config.ExternalAuthProviderSettings.KeycloakVersion,
		"Command":         command + " " + strings.Join(extraArguments, " "),
	})
	if err != nil {
		return fmt.Errorf("failed to execute keycloak service file template: %w", err)
	}

	// Install systemd service
	_, err = sshc.Upload(strings.NewReader(keycloakServiceFileContents), "/etc/systemd/system/keycloak.service", true)
	if err != nil {
		return fmt.Errorf("failed to upload keycloak service file: %w", err)
	}

	_, err = sshc.RunCommand("sudo systemctl daemon-reload")
	if err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Ensure service is enabled
	_, err = sshc.RunCommand("sudo systemctl enable keycloak")
	if err != nil {
		return fmt.Errorf("failed to enable keycloak service: %w", err)
	}

	// Using restart to apply any possible changes to the service
	_, err = sshc.RunCommand("sudo systemctl restart keycloak")
	if err != nil {
		return fmt.Errorf("failed to restart keycloak service: %w", err)
	}

	// Wait for keycloak to start
	url := fmt.Sprintf("http://%s:8080/health", t.output.KeycloakServer.PrivateIP)
	timeout := time.After(120 * time.Second) // yes, is **that** slow
	for {
		resp, err := http.Get(url)
		if err != nil {
			// Avoid error spamming
			if !errors.Is(err, syscall.ECONNREFUSED) {
				mlog.Error("Failed to connect to keycloak", mlog.Err(err))
				continue
			}
		}

		if err == nil {
			resp.Body.Close()
			// Keycloak is ready
			if resp.StatusCode == http.StatusOK {
				break
			}
		}

		mlog.Info("Keycloak service not up yet, waiting...")
		select {
		case <-timeout:
			return errors.New("timeout: keycloak service is not responding")
		case <-time.After(5 * time.Second):
		}
	}

	// Authenticate as admin to execute keycloak commands
	_, err = sshc.RunCommand(fmt.Sprintf(`%s/kcadm.sh config credentials --server http://127.0.0.1:8080 --user "%s" --password '%s' --realm master`, keycloakBinPath, t.config.ExternalAuthProviderSettings.KeycloakAdminUser, strings.ReplaceAll(t.config.ExternalAuthProviderSettings.KeycloakAdminPassword, "'", "\\'")))
	if err != nil {
		return fmt.Errorf("failed to authenticate keycloak admin: %w", err)
	}

	// Disable SSL requirement on master realm to allow http connections on the web interface
	_, err = sshc.RunCommand(keycloakBinPath + "/kcadm.sh update realms/master -s sslRequired=NONE")
	if err != nil {
		return fmt.Errorf("failed to disable ssl requirement: %w", err)
	}

	// Populate users
	if t.config.ExternalAuthProviderSettings.GenerateUsersCount > 0 {
		if err := t.populateKeycloakUsers(sshc); err != nil {
			return fmt.Errorf("failed to populate keycloak users: %w", err)
		}

		mlog.Info("Overriding the users file path with the generated one from keycloak")
		t.config.UsersFilePath = t.getAsset("keycloak-users.txt")
	}

	mlog.Info("Keycloak configured")

	return nil
}

// populateKeycloakUsers creates users in keycloak and writes their credentials to a users file.
// It will use the `GenerateUsersCount` configuration option to create as many users as the configuration
// specifies. If the users file already exists and has the expected number of users, it will skip user creation.
// If the file has less users than expected, it will start creating users from the next number by counting the
// number of lines in the file.
// If the file has more users than expected, it will log and skip the user creation.
func (t *Terraform) populateKeycloakUsers(sshc *ssh.Client) error {
	keycloakBinPath := "/opt/keycloak/keycloak-" + t.config.ExternalAuthProviderSettings.KeycloakVersion + "/bin"
	usersTxtPath := t.getAsset("keycloak-users.txt")
	startNumber := 1

	// Check if users file exists and has the expected number of users. If the file has less users than expected,
	// we will start creating users from the next number, otherwise we will skip user creation.
	if _, err := os.Stat(usersTxtPath); err == nil {
		// Check number of lines in the file to check the number of users already created
		file, err := os.Open(usersTxtPath)
		if err != nil {
			return fmt.Errorf("failed to open keycloak users file: %w", err)
		}
		defer file.Close()

		lines := 0
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			// Avoid empty lines (last line is empty since text files should end with a newline character)
			if scanner.Text() == "" {
				continue
			}

			lines++
		}

		if lines >= t.config.ExternalAuthProviderSettings.GenerateUsersCount {
			mlog.Info(
				"Users file already exists and has the expected number of users (or more), skipping user creation",
				mlog.Int("generate_users_count", t.config.ExternalAuthProviderSettings.GenerateUsersCount),
				mlog.Int("users_count", lines),
			)
			return nil
		}

		if lines < t.config.ExternalAuthProviderSettings.GenerateUsersCount {
			startNumber = lines + 1
			mlog.Info(
				"Users file already exists but has less users than expected, starting from the next number",
				mlog.Int("users_count", lines),
				mlog.Int("start_number", startNumber),
			)
		}
	}

	mlog.Info("Populating keycloak with users", mlog.String("users_file", usersTxtPath), mlog.Int("users_count", t.config.ExternalAuthProviderSettings.GenerateUsersCount))

	// Open users.txt file
	handler, err := os.OpenFile(usersTxtPath, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("failed to open keycloak users file: %w", err)
	}
	defer handler.Close()

	// Create users
	for i := startNumber; i <= t.config.ExternalAuthProviderSettings.GenerateUsersCount; i++ {
		username := fmt.Sprintf("keycloak-auto-%06d", i)
		password := username // we just use the same string for both

		email := username + "@test.mattermost.com"

		_, err := sshc.RunCommand(fmt.Sprintf("%s/kcadm.sh create users -r %s -s username=%s -s enabled=true -s email=%s", keycloakBinPath, t.config.ExternalAuthProviderSettings.KeycloakRealmName, username, email))
		if err != nil {
			return fmt.Errorf("failed to create keycloak user: %w", err)
		}

		_, err = sshc.RunCommand(fmt.Sprintf("%s/kcadm.sh set-password -r %s --username %s --new-password %s", keycloakBinPath, t.config.ExternalAuthProviderSettings.KeycloakRealmName, username, password))
		if err != nil {
			return fmt.Errorf("failed to set keycloak user password: %w", err)
		}

		handler.Write([]byte(fmt.Sprintf("openid:%s %s\n", email, password)))
	}

	return nil
}

func (t *Terraform) IngestKeycloakDump() error {
	mlog.Info("Populating keycloak database with provided dump")

	output, err := t.Output()
	if err != nil {
		return err
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}

	if output.KeycloakServer.PrivateIP == "" {
		return fmt.Errorf("no keycloak instances deployed")
	}

	client, err := extAgent.NewClient(output.KeycloakServer.PrivateIP)
	if err != nil {
		return fmt.Errorf("error in getting ssh connection %w", err)
	}
	defer client.Close()

	// Ensure keycloak is stopped
	_, err = client.RunCommand("sudo systemctl stop keycloak")
	// Ignore errors if the serivce file does not exist yet (on first creation)
	if err != nil && !strings.Contains(err.Error(), "exited with status 5") {
		return fmt.Errorf("failed to stop keycloak before uploading the dump file: %w", err)
	}

	dumpURI := t.config.ExternalAuthProviderSettings.KeycloakDBDumpURI
	fileName := filepath.Base(dumpURI)
	mlog.Info("Provisioning keycloak dump file", mlog.String("uri", dumpURI))
	if err := deployment.ProvisionURL(client, dumpURI, fileName); err != nil {
		return err
	}

	commands := []string{
		"tar xzf /home/ubuntu/" + fileName + " -C /tmp",
		"sudo -iu postgres psql -d keycloak -v ON_ERROR_STOP=on -f /tmp/" + strings.TrimSuffix(fileName, ".tgz"),
	}

	for _, cmd := range commands {
		output, err := client.RunCommand(cmd)
		if err != nil {
			mlog.Error("Error running command", mlog.String("command", cmd), mlog.String("result", string(output)), mlog.Err(err))
			return fmt.Errorf("failed to run command %q: %w", cmd, err)
		}
	}

	return nil
}

// setupKeycloakAppConfig sets up the Keycloak configuration in the Mattermost server for OpenID
// and SAML.
func (t *Terraform) setupKeycloakAppConfig(sshc *ssh.Client, cfg *model.Config) error {
	keycloakScheme := "https"
	if t.config.ExternalAuthProviderSettings.DevelopmentMode {
		keycloakScheme = "http"
	}

	// Setup SAML certificate for Keycloak
	samlIDPCert, err := os.Open(t.getAsset("saml-idp.crt"))
	if err != nil {
		return fmt.Errorf("error opening saml-idp.crt: %w", err)
	}

	if out, err := sshc.Upload(samlIDPCert, "/opt/mattermost/config/saml-idp.crt", false); err != nil {
		return fmt.Errorf("error uploading saml-idp.crt: %s - %w", out, err)
	}

	keycloakUrl := keycloakScheme + "://" + t.output.KeycloakServer.PrivateDNS + ":8080"

	cfg.OpenIdSettings.Enable = model.NewPointer(true)
	cfg.OpenIdSettings.ButtonText = model.NewPointer("OpenID Login")
	cfg.OpenIdSettings.DiscoveryEndpoint = model.NewPointer(keycloakUrl + "/realms/" + t.config.ExternalAuthProviderSettings.KeycloakRealmName + "/.well-known/openid-configuration")
	cfg.OpenIdSettings.Id = model.NewPointer(t.config.ExternalAuthProviderSettings.KeycloakClientID)
	cfg.OpenIdSettings.Secret = model.NewPointer(t.config.ExternalAuthProviderSettings.KeycloakClientSecret)
	cfg.SamlSettings.Enable = model.NewPointer(true)
	cfg.SamlSettings.EnableSyncWithLdap = model.NewPointer(false)
	cfg.SamlSettings.EnableSyncWithLdapIncludeAuth = model.NewPointer(false)
	cfg.SamlSettings.IgnoreGuestsLdapSync = model.NewPointer(false)
	cfg.SamlSettings.Verify = model.NewPointer(false)
	cfg.SamlSettings.Encrypt = model.NewPointer(false)
	cfg.SamlSettings.SignRequest = model.NewPointer(false)
	cfg.SamlSettings.IdpURL = model.NewPointer(keycloakUrl + "/realms/" + t.config.ExternalAuthProviderSettings.KeycloakRealmName + "/protocol/saml")
	cfg.SamlSettings.IdpDescriptorURL = model.NewPointer(keycloakUrl + "/realms/" + t.config.ExternalAuthProviderSettings.KeycloakRealmName)
	cfg.SamlSettings.IdpMetadataURL = model.NewPointer(keycloakUrl + "/realms/" + t.config.ExternalAuthProviderSettings.KeycloakRealmName + "/protocol/saml/descriptor")
	cfg.SamlSettings.ServiceProviderIdentifier = model.NewPointer(t.config.ExternalAuthProviderSettings.KeycloakSAMLClientID)
	cfg.SamlSettings.AssertionConsumerServiceURL = model.NewPointer("http://" + getServerURL(t.output, t.config) + "/login/sso/saml")
	cfg.SamlSettings.SignatureAlgorithm = model.NewPointer("RSAwithSHA1")
	cfg.SamlSettings.CanonicalAlgorithm = model.NewPointer("Canonical1.0")
	cfg.SamlSettings.ScopingIDPProviderId = model.NewPointer("")
	cfg.SamlSettings.ScopingIDPName = model.NewPointer("")
	cfg.SamlSettings.IdpCertificateFile = model.NewPointer("saml-idp.crt")
	cfg.SamlSettings.PublicCertificateFile = model.NewPointer("")
	cfg.SamlSettings.PrivateKeyFile = model.NewPointer("")
	cfg.SamlSettings.IdAttribute = model.NewPointer("id")
	cfg.SamlSettings.GuestAttribute = model.NewPointer("")
	cfg.SamlSettings.EnableAdminAttribute = model.NewPointer(false)
	cfg.SamlSettings.AdminAttribute = model.NewPointer("")
	cfg.SamlSettings.FirstNameAttribute = model.NewPointer("")
	cfg.SamlSettings.LastNameAttribute = model.NewPointer("")
	cfg.SamlSettings.EmailAttribute = model.NewPointer("email")
	cfg.SamlSettings.UsernameAttribute = model.NewPointer("username")
	cfg.SamlSettings.NicknameAttribute = model.NewPointer("")
	cfg.SamlSettings.LocaleAttribute = model.NewPointer("")
	cfg.SamlSettings.PositionAttribute = model.NewPointer("")
	cfg.SamlSettings.LoginButtonText = model.NewPointer("SAML Login")
	cfg.SamlSettings.LoginButtonColor = model.NewPointer("#34a28b")
	cfg.SamlSettings.LoginButtonBorderColor = model.NewPointer("#2389D7")
	cfg.SamlSettings.LoginButtonTextColor = model.NewPointer("#ffffff")

	return nil
}
