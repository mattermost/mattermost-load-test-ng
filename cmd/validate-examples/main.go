package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mattermost/mattermost-load-test-ng/comparison"
	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
)

func main() {
	configType := flag.String("type", "", "config type: config, comparison, coordinator, deployer")
	flag.Parse()

	if *configType == "" {
		fmt.Fprintf(os.Stderr, "error: --type is required\n")
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "error: exactly one file path argument is required\n")
		os.Exit(1)
	}
	filePath := args[0]

	var cfg any
	var err error

	switch *configType {
	case "config":
		cfg, err = loadtest.ReadConfig(filePath)
	case "comparison":
		cfg, err = comparison.ReadConfig(filePath)
	case "coordinator":
		cfg, err = coordinator.ReadConfig(filePath)
	case "deployer":
		cfg, err = deployment.ReadConfig(filePath)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown config type %q\n", *configType)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: ReadConfig failed for %s: %v\n", filePath, err)
		os.Exit(1)
	}

	if err := defaults.Validate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: validation failed for %s: %v\n", filePath, err)
		os.Exit(1)
	}

	fmt.Printf("ok: %s (%s)\n", filePath, *configType)
}
