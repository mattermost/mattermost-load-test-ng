package defaults

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFromTOML(t *testing.T) {
	t.Run("changed values", func(t *testing.T) {
		var cfg testCFG

		f, cleanup := getTestCFG(t, `Setting = "hello"
[Nested]
Setting = "nested_custom"`, "toml")
		defer cleanup()

		require.NoError(t, ReadFrom(f.Name(), "", &cfg))
		assert.Equal(t, "hello", cfg.Setting)
		assert.Equal(t, "nested_custom", cfg.Nested.Setting)
	})

	t.Run("unknown field", func(t *testing.T) {
		var cfg testCFG
		f, cleanup := getTestCFG(t, `Unknown = "hello"`, "toml")
		defer cleanup()

		err := ReadFrom(f.Name(), "", &cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fields in the document are missing in the target struct")
	})

	t.Run("ensure default values", func(t *testing.T) {
		var cfg testCFG

		f, cleanup := getTestCFG(t, ``, "toml")
		defer cleanup()

		require.NoError(t, ReadFrom(f.Name(), "", &cfg))
		assert.Equal(t, "hi", cfg.Setting)
		assert.Equal(t, 1, cfg.Another)
		assert.Equal(t, "nested", cfg.Nested.Setting)
	})
}
