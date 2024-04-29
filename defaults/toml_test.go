package defaults

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFromTOML(t *testing.T) {
	t.Run("changed values", func(t *testing.T) {
		cfg, f, cleanup := getTestCFG(t, `Setting = "hello"
[Nested]
Setting = "nested_custom"`)
		defer cleanup()

		require.NoError(t, ReadFromTOML(f.Name(), &cfg))
		assert.Equal(t, "hello", cfg.Setting)
		assert.Equal(t, "nested_custom", cfg.Nested)
	})

	t.Run("ensure default values", func(t *testing.T) {
		cfg, f, cleanup := getTestCFG(t, ``)
		defer cleanup()

		require.NoError(t, ReadFromTOML(f.Name(), &cfg))
		assert.Equal(t, "hi", cfg.Setting)
		assert.Equal(t, 1, cfg.Another)
		assert.Equal(t, "nested", cfg.Nested.Setting)
	})
}
