package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAskForConfirmation(t *testing.T) {
	t.Run("happy paths", func(t *testing.T) {
		testCases := []struct {
			name           string
			input          string
			expectedResult bool
		}{
			{
				name:           "User enters 'y'",
				input:          "y\n",
				expectedResult: true,
			},
			{
				name:           "User enters 'yes'",
				input:          "yes\n",
				expectedResult: true,
			},
			{
				name:           "User enters 'Y' (uppercase)",
				input:          "Y\n",
				expectedResult: true,
			},
			{
				name:           "User enters 'YES' (uppercase)",
				input:          "YES\n",
				expectedResult: true,
			},
			{
				name:           "User enters 'n'",
				input:          "n\n",
				expectedResult: false,
			},
			{
				name:           "User enters 'no'",
				input:          "no\n",
				expectedResult: false,
			},
			{
				name:           "User enters empty string",
				input:          "\n",
				expectedResult: false,
			},
			{
				name:           "User enters other text",
				input:          "maybe\n",
				expectedResult: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Save original stdin and stdout
				oldStdin := os.Stdin
				t.Cleanup(func() {
					// Restore stdin and stdout
					os.Stdin = oldStdin
				})

				// Create a pipe to simulate user input
				newStdin, newStdinWriter, _ := os.Pipe()
				os.Stdin = newStdin

				// Write the test input to the newStdin
				_, err := newStdinWriter.Write([]byte(tc.input))
				require.NoError(t, err)
				newStdinWriter.Close()

				// Call the function
				result, err := askForConfirmation("Proceed?")

				// Check error
				require.NoError(t, err)

				// Check result
				require.Equal(t, tc.expectedResult, result)
			})
		}
	})
}
