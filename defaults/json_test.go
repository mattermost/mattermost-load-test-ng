package defaults

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFromJSON(t *testing.T) {
	t.Run("changed values", func(t *testing.T) {
		var cfg testCFG
		f, cleanup := getTestCFG(t, `{"setting": "hello", "another": 2}`, "json")
		defer cleanup()

		require.NoError(t, ReadFrom(f.Name(), "", &cfg))
		assert.Equal(t, "hello", cfg.Setting)
		assert.Equal(t, 2, cfg.Another)
	})

	t.Run("unknown field", func(t *testing.T) {
		var cfg testCFG
		f, cleanup := getTestCFG(t, `{"setting": "hello", "unknown": 2}`, "json")
		defer cleanup()

		err := ReadFrom(f.Name(), "", &cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown field")
	})

	t.Run("ensure default values", func(t *testing.T) {
		var cfg testCFG
		f, cleanup := getTestCFG(t, `{"setting": "hello"}`, "json")
		defer cleanup()

		require.NoError(t, ReadFrom(f.Name(), "", &cfg))
		assert.Equal(t, 1, cfg.Another)
		assert.Equal(t, "hello", cfg.Setting)
		assert.Equal(t, "nested", cfg.Nested.Setting)
	})
}
