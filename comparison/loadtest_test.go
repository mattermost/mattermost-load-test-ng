package comparison

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildLoadDBDumpCmd(t *testing.T) {
	t.Run("without permalink IPs to replace", func(t *testing.T) {
		newIP := "192.168.1.1"
		oldIPs := []string{}

		cmds, err := buildLoadDBDumpCmds("dbfilename", newIP, oldIPs, dbSettings{
			UserName: "mmuser",
			Password: "mostest",
			DBName:   "mattermost",
			Host:     "test-db-0.c0gzspkyf2r5.us-east-1.rds.amazonaws.com",
			Engine:   "aurora-postgresql",
		})
		require.NoError(t, err)

		// Check the slice contains only two commands: zcat and psql
		require.Len(t, cmds, 2)
		require.True(t, strings.HasPrefix(cmds[0], "zcat"))
		require.True(t, strings.HasPrefix(cmds[1], "psql"))
	})

	t.Run("with permalink IPs to replace", func(t *testing.T) {
		newIP := "192.168.1.1"
		oldIPs := []string{"54.78.456.5", "56.78.98.1"}

		cmds, err := buildLoadDBDumpCmds("dbfilename", newIP, oldIPs, dbSettings{
			UserName: "mmuser",
			Password: "mostest",
			DBName:   "mattermost",
			Host:     "test-db-0.c0gzspkyf2r5.us-east-1.rds.amazonaws.com",
			Engine:   "aurora-postgresql",
		})
		require.NoError(t, err)

		// Pre-flight check: the slice should contain three commands: zcat, sed and psql
		require.Len(t, cmds, 3)
		require.True(t, strings.HasPrefix(cmds[0], "zcat"))
		require.True(t, strings.HasPrefix(cmds[1], "sed"))
		require.True(t, strings.HasPrefix(cmds[2], "psql"))

		// Check that the old IPs are indeed replaced and unrelated IPs are kept
		input := "54.78.456.5:8065/teamname/pl/id1 56.78.98.1/anotherteam/pl/id2 11.22.33.4/teamname/pl/id3"
		expectedOutput := "192.168.1.1:8065/teamname/pl/id1 192.168.1.1:8065/anotherteam/pl/id2 11.22.33.4/teamname/pl/id3"
		// We need to remove the single quotes here for the command to work with
		// os/exec (in the actual code, it is sent through ssh and they're needed).
		// We also need to split the binary name and the arguments to pass everything
		// to exec.Command
		binaryWithArguments := strings.Split(strings.ReplaceAll(cmds[1], "'", ""), " ")
		binaryName := binaryWithArguments[0]
		arguments := binaryWithArguments[1:]
		sedCmd := exec.Command(binaryName, arguments...)

		// Send the input string to the sed command and compare the output against
		// the expected one
		sedCmd.Stdin = strings.NewReader(input)
		stdout, err := sedCmd.Output()
		require.NoError(t, err)
		require.Equal(t, expectedOutput, string(stdout))
	})
}
