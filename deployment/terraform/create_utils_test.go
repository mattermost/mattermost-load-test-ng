package terraform

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseEthtoolOutputRXSizes(t *testing.T) {
	t.Run("valid parsing", func(t *testing.T) {
		output := `Ring parameters for ens3:
Pre-set maximums:
RX:             4096
RX Mini:        n/a
RX Jumbo:       n/a
TX:             4096
Current hardware settings:
RX:             1024
RX Mini:        n/a
RX Jumbo:       n/a
TX:             1024`

		parsedOutput, err := parseEthtoolOutputRXSizes(output)
		require.NoError(t, err)

		require.Equal(t, 4096, parsedOutput.maxRX)
		require.Equal(t, 1024, parsedOutput.actualRX)
	})

	t.Run("invalid parsing - empty output", func(t *testing.T) {
		output := ""
		_, err := parseEthtoolOutputRXSizes(output)
		require.Error(t, err)
	})

	t.Run("invalid parsing - too many RX lines", func(t *testing.T) {
		output := `Ring parameters for ens3:
Pre-set maximums:
RX:             4096
RX Mini:        n/a
RX Jumbo:       n/a
TX:             4096
Current hardware settings:
RX:             1024
RX Mini:        n/a
RX Jumbo:       n/a
TX:             1024
Another RX line:
RX:             1024`

		_, err := parseEthtoolOutputRXSizes(output)
		require.Error(t, err)
	})

	t.Run("invalid parsing - too few RX lines", func(t *testing.T) {
		output := `Ring parameters for ens3:
Pre-set maximums:
RX:             4096
RX Mini:        n/a
RX Jumbo:       n/a
TX:             4096
Current hardware settings:
RX Mini:        n/a
RX Jumbo:       n/a
TX:             1024`

		_, err := parseEthtoolOutputRXSizes(output)
		require.Error(t, err)
	})
}
