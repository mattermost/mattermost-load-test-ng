package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"

	"github.com/spf13/cobra"
)

func init() {
	for k, c := range configs {
		cfg, err := readDefaultConfig(k, c.defaultPath)
		checkError(err)

		c.defaultValue = cfg
		configs[k] = c
	}
}

var configs = map[string]config{
	"loadtest": {
		docPath:     "./docs/loadtest_config.md",
		defaultPath: "./config/config.default.json",
	},
	"coordinator": {
		docPath:     "./docs/coordinator_config.md",
		defaultPath: "./config/coordinator.default.json",
	},
	"deployer": {
		docPath:     "./docs/deployer_config.md",
		defaultPath: "./config/deployer.default.json",
	},
	"simplecontroller": {
		docPath:     "./docs/simplecontroller_config.md",
		defaultPath: "./config/simplecontroller.default.json",
	},
	"simulcontroller": {
		docPath:     "./docs/simulcontroller_config.md",
		defaultPath: "./config/simulcontroller.default.json",
	},
}

type config struct {
	docPath      string
	defaultPath  string
	defaultValue control.Config
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "ltassist",
		Short: "Tool for load test utilities.",
	}

	configCmd := &cobra.Command{
		Use:     "config [type]",
		RunE:    runConfigAssistCmdF,
		Short:   "Create config interactively",
		Long:    "Interactively create specified config type and save the file.\n" + validTypes(),
		Example: "ltassist config simplecontroller",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cobra.ExactArgs(1)(cmd, args)
			}
			if _, ok := configs[args[0]]; ok {
				return nil
			}
			fmt.Println(validTypes())
			return cobra.OnlyValidArgs(cmd, args)
		},
	}
	rootCmd.AddCommand(configCmd)

	checkCmd := &cobra.Command{
		Use:     "check",
		RunE:    runCheckConfigsCmdF,
		Short:   "Verify configs with the docs",
		Long:    "Checks if the specific configs properly documented.",
		Example: "ltassist check",
	}
	rootCmd.AddCommand(checkCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runConfigAssistCmdF(cmd *cobra.Command, args []string) error {
	config, ok := configs[args[0]]
	if !ok {
		return fmt.Errorf("couldn't find a config for %q", args[0])
	}

	fmt.Printf("Creating %s.Config:\n\n", args[0])
	f := config.docPath
	v, err := createStruct(config.defaultValue, f, false)

	checkError(err)

	err = v.Addr().Interface().(control.Config).IsValid()
	checkError(err)

	data, err := json.MarshalIndent(v.Addr().Interface(), "", "  ")
	checkError(err)

	fmt.Printf("%s\n", data)
	return nil
}

func runCheckConfigsCmdF(cmd *cobra.Command, args []string) error {
	for name, config := range configs {
		f := config.docPath
		_, err := createStruct(config.defaultValue, f, true)
		if err != nil {
			fmt.Printf("docs for %s.Config is not consistent: %s\n", name, err)
		}
	}
	return nil
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func validTypes() string {
	s := "Valid types are:"
	for name := range configs {
		s += "\n - " + name
	}
	return s
}

func readDefaultConfig(configType, defaultPath string) (control.Config, error) {
	switch configType {
	case "loadtest":
		return loadtest.ReadConfig(defaultPath)
	case "coordinator":
		return coordinator.ReadConfig(defaultPath)
	case "deployer":
		return deployment.ReadConfig(defaultPath)
	case "simplecontroller":
		return simplecontroller.ReadConfig(defaultPath)
	case "simulcontroller":
		return simulcontroller.ReadConfig(defaultPath)
	}
	return nil, fmt.Errorf("could not find: %q", configType)
}
