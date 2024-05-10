// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	client "github.com/mattermost/mattermost-load-test-ng/api/client/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/cluster"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"

	"github.com/spf13/cobra"
)

func loadConfig(cmd *cobra.Command) (Config, error) {
	var config Config
	configFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		return config, err
	}
	if configFilePath == "" {
		return config, errors.New("config file is required")
	}
	if err := defaults.ReadFrom(configFilePath, "", &config); err != nil {
		return config, err
	}
	if err := defaults.Validate(config); err != nil {
		return config, fmt.Errorf("could not validate configuration: %w", err)
	}
	return config, nil
}

func loadLoadTestConfig(cmd *cobra.Command) (loadtest.Config, error) {
	var ltConfig loadtest.Config
	configFilePath, err := cmd.Flags().GetString("ltconfig")
	if err != nil {
		return ltConfig, err
	}
	cfg, err := loadtest.ReadConfig(configFilePath)
	if err != nil {
		return ltConfig, err
	}
	ltConfig = *cfg
	if err := defaults.Validate(ltConfig); err != nil {
		return ltConfig, fmt.Errorf("could not validate configuration: %w", err)
	}
	return ltConfig, nil
}

func loadCoordinatorConfig(cmd *cobra.Command) (coordinator.Config, error) {
	var coordConfig coordinator.Config
	configFilePath, err := cmd.Flags().GetString("coordconfig")
	if err != nil {
		return coordConfig, err
	}
	cfg, err := coordinator.ReadConfig(configFilePath)
	if err != nil {
		return coordConfig, err
	}
	coordConfig = *cfg
	if err := defaults.Validate(coordConfig); err != nil {
		return coordConfig, fmt.Errorf("could not validate configuration: %w", err)
	}
	return coordConfig, nil
}

func printCoordinatorStatus(id string, status coordinator.Status) {
	fmt.Println("==================================================")
	fmt.Println(id)
	fmt.Println("")
	fmt.Println("State:", status.State)
	fmt.Println("Start time:", status.StartTime.Format(time.UnixDate))
	if status.State == coordinator.Done {
		fmt.Println("Stop time:", status.StopTime.Format(time.UnixDate))
		fmt.Println("Duration:", status.StopTime.Sub(status.StartTime).Round(time.Second))
	}
	fmt.Println("Active users:", status.ActiveUsers)
	fmt.Println("Number of errors:", status.NumErrors)
	if status.State == coordinator.Done {
		fmt.Println("Supported users:", status.SupportedUsers)
	}
	fmt.Println("==================================================")
}

func runLoadTest(config Config, ltConfig loadtest.Config, coordConfig coordinator.Config) error {
	coordinators := make([]*client.Coordinator, len(config.AppInstances))

	for i, instance := range config.AppInstances {
		id := "coord-" + instance.Id
		coord, err := client.New(id, config.AgentInstances[0].ApiURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create coordinator client: %w", err)
		}
		coordinators[i] = coord
	}

	for i, coord := range coordinators {
		coordConfig.ClusterConfig.Agents = make([]cluster.LoadAgentConfig, len(config.AgentInstances))
		ltConfig.ConnectionConfiguration.ServerURL = config.AppInstances[i].ServerURL
		ltConfig.ConnectionConfiguration.WebSocketURL = config.AppInstances[i].WebSocketURL
		for j, agent := range config.AgentInstances {
			coordConfig.ClusterConfig.Agents[j].Id = fmt.Sprintf("coord-%s-agent-%s", coord.Id(), agent.Id)
			coordConfig.ClusterConfig.Agents[j].ApiURL = agent.ApiURL
		}
		if _, err := coord.Create(&coordConfig, &ltConfig); err != nil {
			return fmt.Errorf("failed to create coordinator: %w", err)
		}
		fmt.Printf("%s created\n", coord.Id())
		if _, err := coord.Run(); err != nil {
			return fmt.Errorf("failed to start coordinator: %w", err)
		}
		fmt.Printf("%s started\n", coord.Id())

		defer func(c *client.Coordinator) {
			if _, err := c.Destroy(); err != nil {
				fmt.Printf("failed to destroy %s: %s\n", c.Id(), err)
			}
			fmt.Printf("%s destroyed\n", c.Id())
		}(coord)
	}

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(10 * time.Second)

	showStatus := func() int {
		var done int
		for _, coord := range coordinators {
			status, err := coord.Status()
			if err != nil {
				fmt.Println(err)
				continue
			}
			if status.State == coordinator.Done {
				done++
			}
			printCoordinatorStatus(coord.Id(), status)
		}
		return done
	}

	showStatus()

	for {
		select {
		case <-interruptChannel:
			fmt.Println("")
			return nil
		case <-ticker.C:
			if showStatus() == len(coordinators) {
				fmt.Println("coordinators are done, exiting")
				return nil
			}
		}
	}

	return nil
}

func runCmdF(cmd *cobra.Command, args []string) error {
	config, err := loadConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	ltConfig, err := loadLoadTestConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	coordConfig, err := loadCoordinatorConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := runLoadTest(config, ltConfig, coordConfig); err != nil {
		return fmt.Errorf("failed to run load-test: %w", err)
	}

	return nil
}

func main() {
	cmd := &cobra.Command{
		Use:          "go run ./examples/api -c config.json",
		RunE:         runCmdF,
		SilenceUsage: true,
	}
	cmd.PersistentFlags().StringP("config", "c", "", "path to the configuration file to use")
	cmd.PersistentFlags().StringP("ltconfig", "", "", "path to the load-test agent configuration file to use")
	cmd.PersistentFlags().StringP("coordconfig", "", "", "path to the coordinator configuration file to use")
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
