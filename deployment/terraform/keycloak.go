package terraform

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
)

func (t *Terraform) setupKeycloak(extAgent *ssh.ExtAgent) error {
	command := "start-dev"

	if !t.config.ExternalAuthProviderSettings.DevelopmentMode {
		command = "start"
	}

	sshc, err := extAgent.NewClient(t.output.KeycloakServer.PublicIP)
	if err != nil {
		return err
	}

	// Install realm file
	if t.config.ExternalAuthProviderSettings.KeycloakRealmFilePath != "" {
		// Copy realm file to server
		_, err := sshc.UploadFile(t.config.ExternalAuthProviderSettings.KeycloakRealmFilePath, "/opt/keycloak/keycloak-"+t.config.ExternalAuthProviderSettings.KeycloakVersion+"/data/import/realm.json", true)
		if err != nil {
			return fmt.Errorf("failed to upload keycloak realm file: %w", err)
		}

		// tell keycloak to import the realm
		command = command + " --import-realm"
	}

	// Create keycloak.env file
	var keycloakEnvFileContents []string

	// Setup the admin user
	keycloakEnvFileContents = append(keycloakEnvFileContents, "KEYCLOAK_ADMIN=%s", t.config.ExternalAuthProviderSettings.KeycloakAdminUser)
	keycloakEnvFileContents = append(keycloakEnvFileContents, "KEYCLOAK_ADMIN_PASSWORD=%s", t.config.ExternalAuthProviderSettings.KeycloakAdminPassword)

	// Setup the database if not running in development mode
	if !t.config.ExternalAuthProviderSettings.DevelopmentMode {
		keycloakEnvFileContents = append(keycloakEnvFileContents, "%s\nKC_DB_URL=%s", t.output.KeycloakDatabaseCluster.Endpoints[0])
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
		Command:         command,
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
	_, err = sshc.RunCommand("sudo systemctl enable --now keycloak")
	if err != nil {
		return fmt.Errorf("failed to enable and start keycloak service: %w", err)
	}

	return nil
}
