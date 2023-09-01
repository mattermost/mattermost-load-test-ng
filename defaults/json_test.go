package defaults

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type testCFG struct {
	Setting string
	Another int
}

func TestReadFromJSON(t *testing.T) {
	var cfg testCFG
	f, err := os.CreateTemp("", "loadtest")
	require.NoError(t, err)
	defer os.Remove(f.Name()) // clean up

	// Ensuring that a bad config throws an error
	_, err = f.Write([]byte(`{"setting": "hello" "another": 1}`))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	require.Error(t, ReadFromJSON("", f.Name(), &cfg))
}
