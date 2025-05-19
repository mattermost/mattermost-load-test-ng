package coordinator

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance"
	"github.com/stretchr/testify/require"
)

func TestConfigIsValid(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := Config{
			NumUsersInc:   8,
			NumUsersDec:   8,
			RestTimeSec:   2,
			MonitorConfig: performance.MonitorConfig{UpdateIntervalMs: 1000},
		}
		require.NoError(t, cfg.IsValid())
	})

	t.Run("invalid NumUsersInc", func(t *testing.T) {
		cfg := Config{
			NumUsersInc:   0, // Invalid: should be > 0
			NumUsersDec:   8,
			RestTimeSec:   2,
			MonitorConfig: performance.MonitorConfig{UpdateIntervalMs: 1000},
		}
		err := cfg.IsValid()
		require.Error(t, err)
		require.Equal(t, "NumUsersInc should be greater than 0", err.Error())
	})

	t.Run("invalid NumUsersDec", func(t *testing.T) {
		cfg := Config{
			NumUsersInc:   8,
			NumUsersDec:   0, // Invalid: should be > 0
			RestTimeSec:   2,
			MonitorConfig: performance.MonitorConfig{UpdateIntervalMs: 1000},
		}
		err := cfg.IsValid()
		require.Error(t, err)
		require.Equal(t, "NumUsersDec should be greater than 0", err.Error())
	})

	t.Run("invalid RestTimeSec", func(t *testing.T) {
		cfg := Config{
			NumUsersInc:   8,
			NumUsersDec:   8,
			RestTimeSec:   0, // Invalid: should be > 0
			MonitorConfig: performance.MonitorConfig{UpdateIntervalMs: 1000},
		}
		err := cfg.IsValid()
		require.Error(t, err)
		require.Equal(t, "RestTimeSec should be greater than 0", err.Error())
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
		require.Equal(t, "RestTimerSec should greater than MonitorConfig.UpdateIntervalMs/1000 (2)", err.Error())
	})
}
