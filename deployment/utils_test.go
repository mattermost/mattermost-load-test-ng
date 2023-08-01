package deployment

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenCmdForPermalinksIPsSubstitution(t *testing.T) {
	tcs := []struct {
		name                  string
		newIP                 string
		permalinkIPsToReplace []string
		replacePort           bool
		input                 string
		output                string
	}{
		{
			name:   "empty permalinkIPsToReplace",
			newIP:  "127.0.0.1",
			input:  ``,
			output: ``,
		},
		{
			name:                  "empty newIP",
			permalinkIPsToReplace: []string{"10.1.1.1"},
			input:                 ``,
			output:                ``,
		},
		{
			name:                  "single IP",
			newIP:                 "127.0.0.1",
			permalinkIPsToReplace: []string{"10.1.1.1"},
			input: `
https://10.1.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.0.1.1/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.1.1.1:8065/private-core/xx/5njz9f9y6jfhxe1o7ec76mjjow
https://10.1.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.1.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.0.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.0.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
			`,
			output: `
https://127.0.0.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.0.1.1/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.1.1.1:8065/private-core/xx/5njz9f9y6jfhxe1o7ec76mjjow
https://127.0.0.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://127.0.0.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.0.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.0.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
			`,
		},
		{
			name:                  "multiple IPs",
			newIP:                 "127.0.0.1",
			permalinkIPsToReplace: []string{"10.1.1.1", "10.1.1.2"},
			input: `
https://10.1.1.2:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.0.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.1.1.1:8065/private-core/xx/5njz9f9y6jfhxe1o7ec76mjjow
https://10.1.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.1.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.0.1.2/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.0.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
			`,
			output: `
https://127.0.0.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.0.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.1.1.1:8065/private-core/xx/5njz9f9y6jfhxe1o7ec76mjjow
https://127.0.0.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://127.0.0.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.0.1.2/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.0.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
			`,
		},
		{
			name:                  "with port replacement",
			newIP:                 "127.0.0.1",
			permalinkIPsToReplace: []string{"10.1.1.1", "10.1.1.2"},
			input: `
https://10.1.1.2:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.0.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.1.1.1:8065/private-core/xx/5njz9f9y6jfhxe1o7ec76mjjow
https://10.1.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.1.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.0.1.2:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.1.1.1/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
			`,
			output: `
https://127.0.0.1/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.0.1.1:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.1.1.1:8065/private-core/xx/5njz9f9y6jfhxe1o7ec76mjjow
https://127.0.0.1/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://127.0.0.1/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://10.0.1.2:8065/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
https://127.0.0.1/private-core/pl/5njz9f9y6jfhxe1o7ec76mjjow
			`,
			replacePort: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			outputBuf := bytes.NewBuffer(nil)
			errorBuf := bytes.NewBuffer(nil)

			cmdStr := GenCmdForPermalinksIPsSubstitution(tc.newIP, tc.permalinkIPsToReplace, tc.replacePort)
			if cmdStr != "" {
				cmd := exec.Command("bash", "-c", cmdStr)
				cmd.Stdin = bytes.NewBufferString(tc.input)
				cmd.Stdout = outputBuf
				cmd.Stderr = errorBuf

				err := cmd.Run()
				require.Empty(t, errorBuf.String())
				require.NoError(t, err)
			}

			require.Equal(t, tc.output, outputBuf.String())
		})
	}
}
