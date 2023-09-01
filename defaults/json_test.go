package defaults

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCFG struct {
	Setting string `default:"hi"`
	Another int `default:"1"`
}

func TestReadFromJSON(t *testing.T) {
	cfg := testCFG{}
	f, err := os.CreateTemp("", "loadtest")
	require.NoError(t, err)
	defer os.Remove(f.Name()) // clean up

	// Ensuring that a bad config throws an error
	_, err = f.Write([]byte(`{"setting": "hello" "another": 1}`))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	require.Error(t, ReadFromJSON("", f.Name(), &cfg))

	cfg = testCFG{}
	f1, err := os.CreateTemp("", "loadtest")
	require.NoError(t, err)
	defer os.Remove(f1.Name()) // clean up

	// Ensuring default values get correctly overridden
	_, err = f1.Write([]byte(`{"setting": "hello"}`))
	require.NoError(t, err)
	require.NoError(t, f1.Close())

	require.NoError(t, ReadFromJSON("", f1.Name(), &cfg))
	assert.Equal(t, 1, cfg.Another)
	assert.Equal(t, "hello", cfg.Setting)
}
