package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func askForConfirmation(prompt string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("unable to read answer from user: %w", err)
	}

	return strings.ToLower(answer) != "y", nil
}

// checkDoNotDestroyMetricsInstanceFlag asks for confirmation if the
// --do-not-destroy-metrics-instance flag is passed
func checkDoNotDestroyMetricsInstanceFlag(cmd *cobra.Command, args []string) error {
	maintainMetrics, err := cmd.Flags().GetBool("do-not-destroy-metrics-instance")
	if err != nil {
		return fmt.Errorf("failed getting the --do-not-destroy-metrics-instance flag: %w", err)
	}

	if maintainMetrics {
		confirmed, err := askForConfirmation("CAUTION! The --do-not-destroy-metrics-instance flag will keep the metrics instance alive by removing it from Terraform state. This means that you will need to manually clean it up when you are done with it. Do you want to continue? [y/n] ")
		if err != nil {
			return err
		}

		if !confirmed {
			return fmt.Errorf("Aborting this destroy. Remove the --do-not-destroy-metrics-instance flag if you want to destroy everything.")
		}
	}

	return nil
}
