package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"

	"github.com/spf13/cobra"
)

type config struct {
	docPath      string
	defaultPath  string
	defaultValue control.Config
}

func init() {
	for k, c := range configs {
		cfg, err := readDefaultConfig(k, c.defaultPath)
		checkError(err)

		c.defaultValue = cfg
		configs[k] = c
	}
}

var configs = map[string]config{
	"agent": {
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

func main() {
	rootCmd := &cobra.Command{
		Use:   "ltassist",
		Short: "Helper tool for load-test configuration and documentation.",
	}

	configCmd := &cobra.Command{
		Use:     "config",
		RunE:    runConfigAssistCmdF,
		Short:   "Create a config interactively",
		Long:    "Interactively create selected config type and save the file.\n" + validTypes(),
		Example: "ltassist config",
	}
	rootCmd.AddCommand(configCmd)

	checkCmd := &cobra.Command{
		Use:     "check",
		RunE:    runCheckConfigsCmdF,
		Short:   "Verify configs with the docs",
		Long:    "Checks if the specific configs are properly documented.",
		Example: "ltassist check",
	}
	rootCmd.AddCommand(checkCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runConfigAssistCmdF(_ *cobra.Command, args []string) error {
	if len(args) > 0 {
		if _, ok := configs[args[0]]; ok {
			return createConfig(args[0])
		}
	}

	var configNames []string
	fmt.Printf("Select the configuration type you want to create:\n")
	for name := range configs {
		configNames = append(configNames, name)
	}
	sort.Strings(configNames)
	for i := range configNames {
		fmt.Printf("%d. %s\n", i+1, configNames[i])
	}

	var i int
	var err error
	for {
		inp, er := readInput("int", "")
		if er != nil {
			return er
		}
		i, err = strconv.Atoi(strings.TrimSpace(inp))
		if err != nil {
			fmt.Println(color.RedString("invalid type. Retry:"))
			continue
		}
		break
	}
	if i < 1 || i > len(configNames) {
		checkError(fmt.Errorf("the selection must be in the range 1-%d", len(configNames)))
	}
	return createConfig(configNames[i-1])
}

func createConfig(name string) error {
	config, ok := configs[name]
	if !ok {
		return fmt.Errorf("couldn't find a config for %q", name)
	}

	fmt.Printf("Creating %s.Config:\n\n", name)
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

func runCheckConfigsCmdF(_ *cobra.Command, args []string) error {
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
	case "agent":
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
