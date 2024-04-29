package defaults

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFromJSON(t *testing.T) {
	t.Run("changed values", func(t *testing.T) {
		cfg, f, cleanup := getTestCFG(t, `{"setting": "hello", "another": 2}`)
		defer cleanup()

		require.NoError(t, ReadFromJSON(f.Name(), &cfg))
		assert.Equal(t, "hello", cfg.Setting)
		assert.Equal(t, 2, cfg.Another)
	})

	// TODO: Error json

	t.Run("ensure default values", func(t *testing.T) {
		cfg := testCFG{}
		f1, err := os.CreateTemp("", "loadtest")
		require.NoError(t, err)
		defer os.Remove(f1.Name()) // clean up

		// Ensuring default values get correctly overridden
		_, err = f1.Write([]byte(`{"setting": "hello"}`))
		require.NoError(t, err)
		require.NoError(t, f1.Close())

		require.NoError(t, ReadFromJSON(f1.Name(), &cfg))
		assert.Equal(t, 1, cfg.Another)
		assert.Equal(t, "hello", cfg.Setting)
		assert.Equal(t, "nested", cfg.Nested.Setting)
	})
}
