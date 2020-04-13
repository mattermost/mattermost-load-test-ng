package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"
	"github.com/spf13/cobra"
)

var (
	cMap = map[string]interface{}{
		"./docs/loadtest_config.md":         loadtest.Config{},
		"./docs/coordinator_config.md":      coordinator.Config{},
		"./docs/deployer_config.md":         deployment.Config{},
		"./docs/simplecontroller_config.md": simplecontroller.Config{},
		"./docs/simulcontroller_config.md":  simulcontroller.Config{},
	}
)

func main() {
	rootCmd := &cobra.Command{
		Use:          "ltassist",
		SilenceUsage: true,
		Short:        "Tool for load test utilities.",
	}

	configCmd := &cobra.Command{
		Use:          "config [type]",
		SilenceUsage: true,
		RunE:         runConfigAssistCmdF,
		Short:        "Create config interactively",
		Long:         "Interactively create specified config type and save the file.",
		Example:      "ltassist config simplecontroller",
		Args:         cobra.ExactArgs(1),
	}
	rootCmd.AddCommand(configCmd)

	checkCmd := &cobra.Command{
		Use:          "check",
		SilenceUsage: true,
		RunE:         runCheckConfigsCmdF,
		Short:        "Verify configs with the docs",
		Long:         "Checks if the specific configs properly documented.",
		Example:      "ltassist check",
	}
	rootCmd.AddCommand(checkCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runConfigAssistCmdF(cmd *cobra.Command, args []string) error {
	for f, cfg := range cMap {
		p := reflect.ValueOf(cfg).Type().PkgPath()
		if strings.HasSuffix(p, args[0]) {
			t := reflect.ValueOf(cfg).Type()
			fmt.Printf("Creating %s:\n\n", t.Name())

			v, err := createStruct(reflect.ValueOf(cfg), f, false)
			if err != nil {
				return err
			}

			data, err := json.MarshalIndent(v.Addr().Interface(), "", "  ")
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", data)
			return nil
		}
	}
	return fmt.Errorf("couldn't find a config for %q", args[0])
}

func runCheckConfigsCmdF(cmd *cobra.Command, args []string) error {
	for f, cfg := range cMap {
		t := reflect.ValueOf(cfg).Type()
		p := t.PkgPath()
		_, err := createStruct(reflect.ValueOf(cfg), f, true)
		if err != nil {
			fmt.Printf("docs for %s.%s is not consistent: %s\n", p, t.Name(), err)
		}
	}
	return nil
}
