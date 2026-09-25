package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/mattermost/mattermost-load-test-ng/comparison"
	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"

	"github.com/wiggin77/merror"
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

	if err := validate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: validation failed for %s: %v\n", filePath, err)
		os.Exit(1)
	}

	fmt.Printf("ok: %s (%s)\n", filePath, *configType)
}

// validate runs defaults.Validate and inspects the returned error for false positives
func validate(cfg any) error {
	err := defaults.Validate(cfg)
	if err == nil {
		return nil
	}

	var merr *merror.MError
	if !errors.As(err, &merr) {
		return fmt.Errorf("failed to convert error to merror")
	}

	// defaults.Validate returns an os.ErrNotExist on files that do not exist
	// locally; this is correct on production, but not for this script aiming to
	// validate template config files, which may contain paths that exist in the
	// target deployment, but not where the script is executed, so we need to
	// filter out such error
	filteredMerr := merror.New()
	for _, err := range merr.Errors() {
		if !errors.Is(err, os.ErrNotExist) {
			filteredMerr.Append(err)
		}
	}

	return filteredMerr.ErrorOrNil()
}
