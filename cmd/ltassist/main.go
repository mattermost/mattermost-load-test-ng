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
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"

	"github.com/spf13/cobra"
)

var docs = map[string]string{
	"agent":            "./docs/config/config.md",
	"coordinator":      "./docs/config/coordinator.md",
	"deployer":         "./docs/config/deployer.md",
	"simplecontroller": "./docs/config/simplecontroller.md",
	"simulcontroller":  "./docs/config/simulcontroller.md",
}

func main() {
	rootCmd := &cobra.Command{
		Use:          "ltassist",
		SilenceUsage: true,
		Short:        "Helper tool for load-test configuration and documentation.",
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
		if _, ok := docs[args[0]]; ok {
			return createConfig(args[0])
		}
	}

	var configNames []string
	fmt.Printf("Select the configuration type you want to create:\n")
	for name := range docs {
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
		return fmt.Errorf("the selection must be in the range 1-%d", len(configNames))
	}
	return createConfig(configNames[i-1])
}

func createConfig(name string) error {
	doc, ok := docs[name]
	if !ok {
		return fmt.Errorf("couldn't find a config for %q", name)
	}

	fmt.Printf("Creating %s.Config:\n\n", name)

	cfg, err := getDefaultConfig(name)
	if err != nil {
		return fmt.Errorf("could not get default config: %w", err)
	}

	v, err := createStruct(cfg, doc, false)
	if err != nil {
		return fmt.Errorf("could not create struct: %w", err)
	}

	if err = defaults.Validate(v.Addr().Interface()); err != nil {
		return fmt.Errorf("could not validate configuration: %w", err)
	}

	data, err := json.MarshalIndent(v.Addr().Interface(), "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal config: %w", err)
	}

	fmt.Printf("%s\n", data)
	return nil
}

func runCheckConfigsCmdF(_ *cobra.Command, args []string) error {
	for name, doc := range docs {
		cfg, err := getDefaultConfig(name)
		if err != nil {
			return fmt.Errorf("could not get default config: %w", err)
		}

		_, err = createStruct(cfg, doc, true)
		if err != nil {
			fmt.Printf("docs for %s.Config is not consistent: %s\n", name, err)
		}
	}
	return nil
}

func validTypes() string {
	s := "Valid types are:"
	for name := range docs {
		s += "\n - " + name
	}
	return s
}

func getDefaultConfig(configType string) (interface{}, error) {
	var cfg interface{}
	switch configType {
	case "agent":
		cfg = &loadtest.Config{}
	case "coordinator":
		cfg = &coordinator.Config{}
	case "deployer":
		cfg = &deployment.Config{}
	case "simplecontroller":
		cfg = &simplecontroller.Config{}
	case "simulcontroller":
		cfg = &simulcontroller.Config{}
	default:
		return nil, fmt.Errorf("could not find: %q", configType)
	}
	if err := defaults.Set(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
