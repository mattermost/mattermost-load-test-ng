// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gencontroller

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScheduledPostConfigDefaults(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "gencontroller.json")
	require.NoError(t, os.WriteFile(configPath, []byte("{}"), 0o600))

	cfg, err := ReadConfig(configPath)
	require.NoError(t, err)
	require.Zero(t, cfg.NumScheduledPosts)
	require.Equal(t, 0.1, cfg.PercentRecurringScheduledPosts)
}

func TestScheduledPostConfigValidation(t *testing.T) {
	tests := []struct {
		name       string
		target     int64
		percentage float64
		wantErr    bool
	}{
		{
			name:       "negative target",
			target:     -1,
			percentage: 0.1,
			wantErr:    true,
		},
		{
			name:       "percentage below zero",
			percentage: -0.01,
			wantErr:    true,
		},
		{
			name:       "percentage above one",
			percentage: 1.01,
			wantErr:    true,
		},
		{
			name:       "zero target",
			percentage: 0.1,
		},
		{
			name:       "zero percentage",
			target:     1,
			percentage: 0,
		},
		{
			name:       "one percentage",
			target:     1,
			percentage: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ReadConfig("../../../config/gencontroller.sample.json")
			require.NoError(t, err)
			cfg.NumScheduledPosts = tt.target
			cfg.PercentRecurringScheduledPosts = tt.percentage

			err = cfg.IsValid(10)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
