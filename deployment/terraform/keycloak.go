package terraform

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/assets"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func (t *Terraform) setupKeycloak(extAgent *ssh.ExtAgent) error {
	keycloakBinPath := "/opt/keycloak/keycloak-" + t.config.ExternalAuthProviderSettings.KeycloakVersion + "/bin"

	mlog.Info("Configuring keycloak", mlog.String("host", t.output.KeycloakServer.PublicIP))

	command := "start-dev"

	if !t.config.ExternalAuthProviderSettings.DevelopmentMode {
		command = "start"
	}

	sshc, err := extAgent.NewClient(t.output.KeycloakServer.PublicIP)
	if err != nil {
		return err
	}

	// Create keycloak.env file
	var keycloakEnvFileContents []string

	// Install realm file
	if t.config.ExternalAuthProviderSettings.KeycloakRealmFilePath != "" {
		// Copy realm file to server
		_, err := sshc.UploadFile(t.config.ExternalAuthProviderSettings.KeycloakRealmFilePath, "/opt/keycloak/keycloak-"+t.config.ExternalAuthProviderSettings.KeycloakVersion+"/data/import/realm.json", true)
		if err != nil {
			return fmt.Errorf("failed to upload keycloak realm file: %w", err)
		}

	} else {
		mlog.Info("No realm file provided, using loadtest's default realm configuration")

		keycloakRealmFile, err := assets.AssetString("keycloak-realm.json")
		if err != nil {
			return fmt.Errorf("failed to read keycloak realm file: %w", err)
		}

		_, err = sshc.Upload(strings.NewReader(keycloakRealmFile), "/opt/keycloak/keycloak-"+t.config.ExternalAuthProviderSettings.KeycloakVersion+"/data/import/realm.json", true)
		if err != nil {
			return fmt.Errorf("failed to upload keycloak realm file: %w", err)
		}
	}

	// Setup admin user
	keycloakEnvFileContents = append(keycloakEnvFileContents, "KEYCLOAK_ADMIN="+t.config.ExternalAuthProviderSettings.KeycloakAdminUser)
	keycloakEnvFileContents = append(keycloakEnvFileContents, "KEYCLOAK_ADMIN_PASSWORD="+t.config.ExternalAuthProviderSettings.KeycloakAdminPassword)

	// Enable health endpoints
	keycloakEnvFileContents = append(keycloakEnvFileContents, "KC_HEALTH_ENABLED=true")

	// Production configuration
	if !t.config.ExternalAuthProviderSettings.DevelopmentMode {
		keycloakEnvFileContents = append(keycloakEnvFileContents, "KC_HOSTNAME="+t.output.KeycloakServer.PublicDNS+":8080")
	}

	// Setup the database if not running in development mode
	if !t.config.ExternalAuthProviderSettings.DevelopmentMode {
		dsn := "postgres://" + t.config.ExternalAuthProviderSettings.DatabaseUsername + ":" + t.config.ExternalAuthProviderSettings.DatabasePassword + "@" + t.output.KeycloakDatabaseCluster.Endpoints[0] + "/" + t.output.KeycloakDatabaseCluster.ClusterIdentifier + "keycloakdb?sslmode=disable"
		keycloakEnvFileContents = append(keycloakEnvFileContents, "KC_DB_URL="+dsn)
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
		Command:         command + " --import-realm",
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

	// Disable SSL requirement on the realm set up by the loadtest
	if t.config.ExternalAuthProviderSettings.KeycloakRealmName != "" && t.config.ExternalAuthProviderSettings.KeycloakRealmName != "master" {
		_, err = sshc.RunCommand(fmt.Sprintf(keycloakBinPath+"/kcadm.sh update realms/%s -s sslRequired=NONE", t.config.ExternalAuthProviderSettings.KeycloakRealmName))
		if err != nil {
			return fmt.Errorf("failed to disable ssl requirement: %w", err)
		}
	}

	mlog.Info("Keycloak configured")

	return nil
}
