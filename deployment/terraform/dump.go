// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// ClearLicensesData runs a SQL query to delete all data from old licenses in
// the database. It does it by first stopping the server, then running the
// query, then restarting the server again.
func (t *Terraform) ClearLicensesData() error {
	output, err := t.Output()
	if err != nil {
		return err
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}

	if len(output.Instances) < 1 {
		return fmt.Errorf("no app instances deployed")
	}

	appClients := make([]*ssh.Client, len(output.Instances))
	for i, instance := range output.Instances {
		client, err := extAgent.NewClient(instance.PublicIP)
		if err != nil {
			return fmt.Errorf("error in getting ssh connection %w", err)
		}
		defer client.Close()
		appClients[i] = client
	}

	stopCmd := deployment.Cmd{
		Msg:     "Stopping app servers",
		Value:   "sudo systemctl stop mattermost",
		Clients: appClients,
	}

	clearCmdValue, err := deployment.ClearLicensesCmd(deployment.DBSettings{
		UserName: t.config.TerraformDBSettings.UserName,
		Password: t.config.TerraformDBSettings.Password,
		DBName:   t.config.DBName(),
		Host:     output.DBWriter(),
		Engine:   t.config.TerraformDBSettings.InstanceEngine,
	})
	if err != nil {
		return fmt.Errorf("error building command to clear licenses data: %w", err)
	}

	clearCmd := deployment.Cmd{
		Msg:     "Clearing old licenses data",
		Value:   clearCmdValue,
		Clients: appClients[0:1],
	}

	startCmd := deployment.Cmd{
		Msg:     "Restarting app server",
		Value:   "sudo systemctl start mattermost && until $(curl -sSf http://localhost:8065 --output /dev/null); do sleep 1; done;",
		Clients: appClients,
	}

	if err := t.executeCommands([]deployment.Cmd{stopCmd, clearCmd, startCmd}); err != nil {
		return fmt.Errorf("error executing database commands: %w", err)
	}

	return nil
}

// IngestDump works on an already deployed terraform setup and restores
// the DB dump file to the Mattermost server. It uses the deployment config
// for the dump URI.
func (t *Terraform) IngestDump() error {
	output, err := t.Output()
	if err != nil {
		return err
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}

	if len(t.output.Instances) < 1 {
		return fmt.Errorf("no app instances deployed")
	}

	appClients := make([]*ssh.Client, len(output.Instances))
	for i, instance := range output.Instances {
		client, err := extAgent.NewClient(instance.PublicIP)
		if err != nil {
			return fmt.Errorf("error in getting ssh connection %w", err)
		}
		defer client.Close()
		appClients[i] = client
	}

	dumpURI := t.config.DBDumpURI
	fileName := filepath.Base(dumpURI)
	mlog.Info("Provisioning dump file", mlog.String("uri", dumpURI))
	if err := deployment.ProvisionURL(appClients[0], dumpURI, fileName); err != nil {
		return err
	}

	resetCmd, err := getResetCmd(t.config, output, appClients)
	if err != nil {
		return fmt.Errorf("error building reset cmd: %w", err)
	}

	loadDBDumpCmd := deployment.Cmd{
		Msg:     "Loading DB dump",
		Clients: []*ssh.Client{appClients[0]},
	}

	dbCmd, err := deployment.BuildLoadDBDumpCmd(fileName, deployment.DBSettings{
		UserName: t.config.TerraformDBSettings.UserName,
		Password: t.config.TerraformDBSettings.Password,
		DBName:   t.config.DBName(),
		Host:     output.DBWriter(),
		Engine:   t.config.TerraformDBSettings.InstanceEngine,
	})
	if err != nil {
		return fmt.Errorf("error building command for loading DB dump: %w", err)
	}
	loadDBDumpCmd.Value = dbCmd

	if err := t.executeDatabaseCommands([]deployment.Cmd{resetCmd, loadDBDumpCmd}); err != nil {
		return fmt.Errorf("error ingesting db dump: %w", err)
	}

	return nil
}

// ExecuteCustomSQL executes provided custom SQL files in the app instances.
func (t *Terraform) ExecuteCustomSQL() error {
	output, err := t.Output()
	if err != nil {
		return err
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}

	if len(output.Instances) < 1 {
		return fmt.Errorf("no app instances deployed")
	}

	client, err := extAgent.NewClient(output.Instances[0].PublicIP)
	if err != nil {
		return fmt.Errorf("error in getting ssh connection %w", err)
	}
	defer client.Close()

	var commands []deployment.Cmd

	for _, sqlURI := range t.config.DBExtraSQL {
		fileName := filepath.Base(sqlURI)
		mlog.Info("Provisioning SQL file", mlog.String("uri", sqlURI))
		if err := deployment.ProvisionURL(client, sqlURI, fileName); err != nil {
			return err
		}

		loadSQLFileCmd := deployment.Cmd{
			Msg:     "Loading SQL file: " + fileName,
			Clients: []*ssh.Client{client},
		}

		dbCmd, err := deployment.BuildLoadDBDumpCmd(fileName, deployment.DBSettings{
			UserName: t.config.TerraformDBSettings.UserName,
			Password: t.config.TerraformDBSettings.Password,
			DBName:   t.config.DBName(),
			Host:     output.DBWriter(),
			Engine:   t.config.TerraformDBSettings.InstanceEngine,
		})
		if err != nil {
			return fmt.Errorf("error building command for loading DB dump: %w", err)
		}
		loadSQLFileCmd.Value = dbCmd

		commands = append(commands, loadSQLFileCmd)
	}

	if err := t.executeDatabaseCommands(commands); err != nil {
		return fmt.Errorf("error executing custom SQL files: %w", err)
	}

	return nil
}

// executeDatabaseCommands executes a series of commands stopping the mattermost service in the
// app instances then executing the provided commands and finally starting the mattermost service.
func (t *Terraform) executeDatabaseCommands(extraCommands []deployment.Cmd) error {
	output, err := t.Output()
	if err != nil {
		return err
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}

	if len(output.Instances) < 1 {
		return fmt.Errorf("no app instances deployed")
	}

	appClients := make([]*ssh.Client, len(output.Instances))
	for i, instance := range output.Instances {
		client, err := extAgent.NewClient(instance.PublicIP)
		if err != nil {
			return fmt.Errorf("error in getting ssh connection %w", err)
		}
		defer client.Close()
		appClients[i] = client
	}

	commands := []deployment.Cmd{{
		Msg:     "Stopping app servers",
		Value:   "sudo systemctl stop mattermost",
		Clients: appClients,
	}}

	// Append provided commands
	commands = append(commands, extraCommands...)

	commands = append(commands, deployment.Cmd{
		Msg:     "Restarting app server",
		Value:   "sudo systemctl start mattermost && until $(curl -sSf http://localhost:8065 --output /dev/null); do sleep 1; done;",
		Clients: appClients,
	})

	if err := t.executeCommands(commands); err != nil {
		return fmt.Errorf("error executing database commands: %w", err)
	}

	return nil
}

func (t *Terraform) executeCommands(commands []deployment.Cmd) error {
	for _, c := range commands {
		mlog.Info(c.Msg)

		errors := make(chan error, len(c.Clients))
		wg := sync.WaitGroup{}

		for _, client := range c.Clients {
			wg.Add(1)
			go func() {
				defer wg.Done()
				mlog.Debug("Running cmd", mlog.String("cmd", c.Value), mlog.String("ip", client.IP))
				if out, err := client.RunCommand(c.Value); err != nil {
					errors <- fmt.Errorf("failed to run cmd %q on %s: %w %s", c.Value, client.IP, err, out)
				}
			}()
		}

		wg.Wait()
		go func() {
			close(errors)
		}()

		errorsFound := false
		for e := range errors {
			errorsFound = true
			mlog.Error(e.Error())
		}

		if errorsFound {
			return fmt.Errorf("errors found during command execution")
		}

	}

	return nil
}

func getResetCmd(config *deployment.Config, output *Output, appClients []*ssh.Client) (deployment.Cmd, error) {
	dbName := config.DBName()
	resetCmd := deployment.Cmd{
		Msg:     "Resetting database",
		Clients: []*ssh.Client{appClients[0]},
	}
	switch config.TerraformDBSettings.InstanceEngine {
	case "aurora-postgresql":
		sqlConnParams := fmt.Sprintf("-U %s -h %s %s", config.TerraformDBSettings.UserName, output.DBWriter(), dbName)
		resetCmd.Value = strings.Join([]string{
			fmt.Sprintf("export PGPASSWORD='%s'", config.TerraformDBSettings.Password),
			fmt.Sprintf("dropdb %s", sqlConnParams),
			fmt.Sprintf("createdb %s", sqlConnParams),
			fmt.Sprintf("psql %s -c 'ALTER DATABASE %s SET default_text_search_config TO \"pg_catalog.english\"'", sqlConnParams, dbName),
		}, " && ")
	case "aurora-mysql":
		subCmd := fmt.Sprintf("mysqladmin -h %s -u %s -p%s -f", output.DBWriter(), config.TerraformDBSettings.UserName, config.TerraformDBSettings.Password)
		resetCmd.Value = fmt.Sprintf("%s drop %s && %s create %s", subCmd, dbName, subCmd, dbName)
	default:
		return deployment.Cmd{}, fmt.Errorf("invalid db engine %s", config.TerraformDBSettings.InstanceEngine)
	}

	return resetCmd, nil
}
