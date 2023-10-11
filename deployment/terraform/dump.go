// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

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

	dumpURI := t.config.DBDumpURI
	fileName := filepath.Base(dumpURI)
	mlog.Info("Provisioning dump file", mlog.String("uri", dumpURI))
	if err := deployment.ProvisionURL(appClients[0], dumpURI, fileName); err != nil {
		return err
	}

	// stop
	// reset
	// load dump
	// start

	stopCmd := deployment.Cmd{
		Msg:     "Stopping app servers",
		Value:   "sudo systemctl stop mattermost",
		Clients: appClients,
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

	startCmd := deployment.Cmd{
		Msg:     "Restarting app server",
		Value:   "sudo systemctl start mattermost && until $(curl -sSf http://localhost:8065 --output /dev/null); do sleep 1; done;",
		Clients: appClients,
	}

	for _, c := range []deployment.Cmd{stopCmd, resetCmd, loadDBDumpCmd, startCmd} {
		mlog.Info(c.Msg)
		for _, client := range c.Clients {
			mlog.Debug("Running cmd", mlog.String("cmd", c.Value))
			if out, err := client.RunCommand(c.Value); err != nil {
				return fmt.Errorf("failed to run cmd %q: %w %s", c.Value, err, out)
			}
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
