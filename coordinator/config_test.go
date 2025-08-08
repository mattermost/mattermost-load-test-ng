package coordinator

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/stretchr/testify/require"
)

func TestConfigIsValid(t *testing.T) {
	t.Run("valid default config", func(t *testing.T) {
		var cfg Config
		require.NoError(t, defaults.Set(&cfg))
		require.NoError(t, cfg.IsValid())
	})

	t.Run("invalid NumUsersInc", func(t *testing.T) {
		var cfg Config
		require.NoError(t, defaults.Set(&cfg))

		cfg.NumUsersInc = 0 // Invalid: should be > 0

		err := defaults.Validate(cfg)
		require.Error(t, err)
		require.Equal(t, "NumUsersInc is not in the range of range:(0,]: value 0 is lesser or equal than 0", err.Error())
	})

	t.Run("invalid NumUsersDec", func(t *testing.T) {
		var cfg Config
		require.NoError(t, defaults.Set(&cfg))

		cfg.NumUsersDec = 0 // Invalid: should be > 0

		err := defaults.Validate(cfg)
		require.Error(t, err)
		require.Equal(t, "NumUsersDec is not in the range of range:(0,]: value 0 is lesser or equal than 0", err.Error())
	})

	t.Run("RestTimeSec less than UpdateIntervalMs/1000", func(t *testing.T) {
		cfg := Config{
			NumUsersInc:   8,
			NumUsersDec:   8,
			RestTimeSec:   1, // Invalid: less than UpdateIntervalMs/1000
			MonitorConfig: performance.MonitorConfig{UpdateIntervalMs: 2000},
		}
		err := cfg.IsValid()
		require.Error(t, err)
		require.Equal(t, "RestTimeSec (1) should greater than MonitorConfig.UpdateIntervalMs/1000 (2)", err.Error())
	})
}
