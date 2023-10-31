package terraform

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPyroscopeConfig(t *testing.T) {
	mmTarget := "app-0:8067"
	ltTarget := "agent-0:4000"

	testCases := []struct {
		Name                  string
		MMTargets             []string
		LTTargets             []string
		ExpectedStaticConfigs []StaticConfig
	}{
		{"Both empty", []string{}, []string{}, []StaticConfig{}},
		{"MMTargets empty", []string{}, []string{ltTarget}, []StaticConfig{{"agents", "gospy", []string{ltTarget}}}},
		{"LTTargets empty", []string{mmTarget}, []string{}, []StaticConfig{{"mattermost", "gospy", []string{mmTarget}}}},
		{"Both populated", []string{mmTarget}, []string{ltTarget}, []StaticConfig{{"mattermost", "gospy", []string{mmTarget}}, {"agents", "gospy", []string{ltTarget}}}},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			config := NewPyroscopeConfig(tc.MMTargets, tc.LTTargets)
			require.Len(t, config.ScrapeConfigs, 1)
			require.ElementsMatch(t, config.ScrapeConfigs[0].StaticConfigs, tc.ExpectedStaticConfigs)
		})
	}
}
