package comparison

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValid(t *testing.T) {
	t.Run("default config with different label values is valid", func(t *testing.T) {
		cfg := Config{}
		cfg.BaseBuild.Label = "base"
		cfg.NewBuild.Label = "new"

		require.NoError(t, cfg.IsValid())
	})

	t.Run("config with same labels for both build is not valid", func(t *testing.T) {
		cfg := Config{}
		cfg.BaseBuild.Label = "label"
		cfg.NewBuild.Label = "label"

		require.Error(t, cfg.IsValid())
	})
}
