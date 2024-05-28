package defaults

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type testCFG struct {
	Setting string `default:"hi"`
	Another int    `default:"1"`
	Nested  struct {
		Setting string `default:"nested"`
	}
}

func getTestCFG(t *testing.T, fileContents, extension string) (*os.File, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "loadtest-*."+extension)
	require.NoError(t, err)

	_, err = f.Write([]byte(fileContents))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	return f, func() {
		os.Remove(f.Name()) // clean up
	}
}
