package terraform

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/assets"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func (t *Terraform) setupKeycloak(extAgent *ssh.ExtAgent) error {
	keycloakBinPath := "/opt/keycloak/keycloak-" + t.config.ExternalAuthProviderSettings.KeycloakVersion + "/bin"

	mlog.Info("Configuring keycloak", mlog.String("host", t.output.KeycloakServer.PublicIP))

	command := "start-dev"
	extraArguments := []string{}

	if !t.config.ExternalAuthProviderSettings.DevelopmentMode {
		command = "start"
	}

	sshc, err := extAgent.NewClient(t.output.KeycloakServer.PublicIP)
	if err != nil {
		return err
	}

	// Check if the keycloak database exists
	result, _ := sshc.RunCommand(`sudo -iu postgres psql -l | grep keycloak 2> /dev/null`)
	if len(result) == 0 {
		mlog.Info("keycloak database not found, creating it and associated role")
		_, err = sshc.RunCommand(`sudo -iu postgres psql <<EOSQL
		CREATE USER keycloak WITH PASSWORD 'mmpass';
		CREATE DATABASE keycloak OWNER keycloak;
		GRANT ALL PRIVILEGES ON DATABASE keycloak TO keycloak;
		EOSQL
		'`)
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

	// Check if we should use a custom dump, a custom realm file or the default one
	if t.config.ExternalAuthProviderSettings.KeycloakRealmFilePath != "" {
		_, err := sshc.UploadFile(t.config.ExternalAuthProviderSettings.KeycloakRealmFilePath, "/opt/keycloak/keycloak-"+t.config.ExternalAuthProviderSettings.KeycloakVersion+"/data/import/mattermost-realm.json", true)
		if err != nil {
			return fmt.Errorf("failed to upload keycloak realm file: %w", err)
		}
		extraArguments = append(extraArguments, "--import-realm")
	} else if t.config.ExternalAuthProviderSettings.KeycloakDBDumpURI == "" {
		mlog.Info("No realm file or database dump provided, using loadtest's default keycloak realm configuration")

		keycloakRealmFile, err := assets.AssetString("mattermost-realm.json")
		if err != nil {
			return fmt.Errorf("failed to read default keycloak realm file: %w", err)
		}

		_, err = sshc.Upload(strings.NewReader(keycloakRealmFile), "/opt/keycloak/keycloak-"+t.config.ExternalAuthProviderSettings.KeycloakVersion+"/data/import/mattermost-realm.json", true)
		if err != nil {
			return fmt.Errorf("failed to upload default keycloak realm file: %w", err)
		}
		extraArguments = append(extraArguments, "--import-realm")
	}

	// Values for the keycloak.env file
	keycloakEnvFileContents := []string{
		// Enable health endpoints
		"KC_HEALTH_ENABLED=true",
		// Setup admin user
		"KEYCLOAK_ADMIN=" + t.config.ExternalAuthProviderSettings.KeycloakAdminUser,
		"KEYCLOAK_ADMIN_PASSWORD=" + t.config.ExternalAuthProviderSettings.KeycloakAdminPassword,
		// Ensure Java JVM has enough memory for large imports
		"JAVA_OPTS=-Xms1024m -Xmx2048m",
		// Logging
		"KC_LOG_FILE=/opt/keycloak/keycloak-" + t.config.ExternalAuthProviderSettings.KeycloakVersion + "/data/log/keycloak.log",
		"KC_LOG_FILE_OUTPUT=json",
		// Database
		"KC_DB_POOL_MIN_SIZE=20",
		"KC_DB_POOL_INITIAL_SIZE=20",
		"KC_DB_POOL_MAX_SIZE=200",
		"KC_DB=postgres",
		"KC_DB_URL=jdbc:psql://localhost:5432/keycloak",
		"KC_DB_PASSWORD=mmpass",
		"KC_DB_USERNAME=keycloak",
		"KC_DATABASE=keycloak",
	}

	// Upload keycloak.env file
	_, err = sshc.Upload(strings.NewReader(strings.Join(keycloakEnvFileContents, "\n")), "/etc/systemd/system/keycloak.env", true)
	if err != nil {
		return fmt.Errorf("failed to upload keycloak env file: %w", err)
	}

	// Parse keycloak service file template
	tmpl, err := template.New("keycloakServiceFile").Parse(string(keycloakServiceFileContents))
	if err != nil {
		return fmt.Errorf("failed to parse keycloak service file template: %w", err)
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct {
		KeycloakVersion string
		Command         string
	}{
		KeycloakVersion: t.config.ExternalAuthProviderSettings.KeycloakVersion,
		Command:         command + " " + strings.Join(extraArguments, " "),
	})
	if err != nil {
		return fmt.Errorf("failed to execute keycloak service file template: %w", err)
	}

	// Install systemd service
	_, err = sshc.Upload(&buf, "/etc/systemd/system/keycloak.service", true)
	if err != nil {
		return fmt.Errorf("failed to upload keycloak service file: %w", err)
	}

	// Enable and start service
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
	url := fmt.Sprintf("http://%s:8080/health", t.output.KeycloakServer.PublicDNS)
	timeout := time.After(120 * time.Second) // yes, is **that** slow
	for {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		mlog.Info("Service not up yet, waiting...")
		select {
		case <-timeout:
			return errors.New("timeout: keycloak service is not responding")
		case <-time.After(5 * time.Second):
		}
	}

	_, err = sshc.RunCommand(fmt.Sprintf("%s/kcadm.sh config credentials --server http://127.0.0.1:8080 --user %s --password %s --realm master", keycloakBinPath, t.config.ExternalAuthProviderSettings.KeycloakAdminUser, t.config.ExternalAuthProviderSettings.KeycloakAdminPassword))
	if err != nil {
		return fmt.Errorf("failed to authenticate keycload admin: %w", err)
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

func (t *Terraform) populateKeycloakUsers(sshc *ssh.Client) error {
	keycloakBinPath := "/opt/keycloak/keycloak-" + t.config.ExternalAuthProviderSettings.KeycloakVersion + "/bin"
	usersTxtPath := t.getAsset("keycloak-users.txt")

	// Check if users file exists. Prevents from creating users multiple times.
	if _, err := os.Stat(usersTxtPath); err == nil || os.IsExist(err) {
		return nil
	}

	mlog.Info("Populating keycloak with users", mlog.String("users_file", usersTxtPath), mlog.Int("users_count", t.config.ExternalAuthProviderSettings.GenerateUsersCount))

	// Open users.txt file
	handler, err := os.OpenFile(usersTxtPath, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("failed to open keycloak users file: %w", err)
	}
	defer handler.Close()

	// Create users
	for i := 1; i <= t.config.ExternalAuthProviderSettings.GenerateUsersCount; i++ {
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

	client, err := extAgent.NewClient(output.KeycloakServer.PublicIP)
	if err != nil {
		return fmt.Errorf("error in getting ssh connection %w", err)
	}
	defer client.Close()

	// Ensure keycloak is stopped
	_, err = client.RunCommand("sudo systemctl stop keycloak")
	if err != nil {
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
		"sudo -iu postgres psql -d keycloak -f /tmp/" + strings.TrimSuffix(fileName, ".tgz"),
		"sudo systemctl start keycloak",
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
