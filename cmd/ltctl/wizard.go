package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/spf13/cobra"
)

func RunWizardCreateConfigF(cmd *cobra.Command, args []string) error {
	activeUsers, err := cmd.Flags().GetInt("active-users")
	if err != nil {
		return fmt.Errorf("failed to get active users flag: %w", err)
	}

	arch, ok := deployment.Architectures[activeUsers]
	if !ok {
		var keys []string
		for k := range deployment.Architectures {
			keys = append(keys, fmt.Sprintf("%d", k))
		}
		return fmt.Errorf("reference architecture for %d users not found. Use one of: %s", activeUsers, keys)
	}

	createDeployerConfig, err := cmd.Flags().GetBool("create-deployer")
	if err != nil {
		return fmt.Errorf("failed to get create deployer flag: %w", err)
	}

	if createDeployerConfig {
		deployerConfig := deployment.Config{}
		defaults.Set(&deployerConfig)

		deployerConfig.AppInstanceCount = arch.AppServers.Count
		deployerConfig.AppInstanceType = arch.AppServers.InstanceType

		deployerConfig.TerraformDBSettings.InstanceCount = arch.DatabaseServer.Count
		deployerConfig.TerraformDBSettings.InstanceType = arch.DatabaseServer.InstanceType

		if err := writeToFile("./config/deployer.json", deployerConfig); err != nil {
			return fmt.Errorf("failed to write deployer config: %w", err)
		}
	}

	createConfig, err := cmd.Flags().GetBool("create-config")
	if err != nil {
		return fmt.Errorf("failed to get create deployer flag: %w", err)
	}

	if createConfig {
		config := loadtest.Config{}
		defaults.Set(&config)

		if err := writeToFile("./config/config.json", config); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
	}

	return nil
}

func writeToFile(filename string, cfg any) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", filename, err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write %s: %w", filename, err)
	}

	return nil
}
