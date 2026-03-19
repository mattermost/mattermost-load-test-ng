package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// askForConfirmation prints the prompt to the standard output, followed by the
// string " [y/N] ". Then it reads from the standard input and returns true if
// and only if the read input is either "y" or "yes". It is case-insensitive.
func askForConfirmation(prompt string) (bool, error) {
	// Print prompt to stdout
	fmt.Print(prompt + " [y/N] ")

	// Read input from stdin
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("unable to read answer from user: %w", err)
	}
	answer := strings.ToLower(scanner.Text())

	return answer == "y" || answer == "yes", nil
}

// checkDoNotDestroyMetricsInstanceFlag asks for confirmation if the
// --do-not-destroy-metrics-instance flag is passed
func checkDoNotDestroyMetricsInstanceFlag(cmd *cobra.Command, args []string) error {
	maintainMetrics, err := cmd.Flags().GetBool("do-not-destroy-metrics-instance")
	if err != nil {
		return fmt.Errorf("failed getting the --do-not-destroy-metrics-instance flag: %w", err)
	}

	if maintainMetrics {
		msg := "CAUTION!\n"
		msg += "The --do-not-destroy-metrics-instance flag will keep the metrics instance and associated resources alive, which means that:\n"
		msg += "  1. If you recreate your deployment after this command finishes, the metrics instance will be reused instead of creating a new one.\n"
		msg += "  2. You will need to run `destroy` *without* this flag afterwards to clean everything up.\n"
		msg += "Do you want to continue?"
		confirmed, err := askForConfirmation(msg)
		if err != nil {
			return err
		}

		if !confirmed {
			return fmt.Errorf("Aborting this destroy. Remove the --do-not-destroy-metrics-instance flag if you want to destroy everything.")
		}
	}

	return nil
}
