package terraform

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/stretchr/testify/require"
)

var reListElems = regexp.MustCompile(`\[(.*)\]`)

func TestPyroscopeSettingsGenString(t *testing.T) {
	mmTarget := "app-0:8067"
	ltTarget := "agent-0:4000"

	// The Name represents the state of each setting, with
	// 1 meaning true for bool and non-empty for []string,
	// and 0 meaning false for bool and empty for []string
	testCases := []struct {
		Name                 string
		EnableAppProfiling   bool
		EnableAgentProfiling bool
		MMTargets            []string
		LTTargets            []string
	}{
		{"0000", false, false, []string{}, []string{}},
		{"0001", false, false, []string{}, []string{ltTarget}},
		{"0010", false, false, []string{mmTarget}, []string{}},
		{"0011", false, false, []string{mmTarget}, []string{ltTarget}},
		{"0100", false, true, []string{}, []string{}},
		{"0101", false, true, []string{}, []string{ltTarget}},
		{"0110", false, true, []string{mmTarget}, []string{}},
		{"0111", false, true, []string{mmTarget}, []string{ltTarget}},
		{"1000", true, false, []string{}, []string{}},
		{"1001", true, false, []string{}, []string{ltTarget}},
		{"1010", true, false, []string{mmTarget}, []string{}},
		{"1011", true, false, []string{mmTarget}, []string{ltTarget}},
		{"1100", true, true, []string{}, []string{}},
		{"1101", true, true, []string{}, []string{ltTarget}},
		{"1110", true, true, []string{mmTarget}, []string{}},
		{"1111", true, true, []string{mmTarget}, []string{ltTarget}},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			settings := deployment.PyroscopeSettings{
				EnableAppProfiling:   tc.EnableAppProfiling,
				EnableAgentProfiling: tc.EnableAgentProfiling,
			}

			generatedYaml := settings.GenString(pyroscopeConfig, tc.MMTargets, tc.LTTargets)

			// We need to do some parsing here to get the part we're interested in, which is the targets
			// for the app nodes (the first 'targets: [...]' line), and the targets for the agent nodes,
			// (the second 'targets: [...]' line), and then we extract everything that is inside the
			// square brackets
			targets := []string{}
			for _, line := range strings.Split(generatedYaml, "\n") {
				if strings.Contains(line, "targets") {
					matches := reListElems.FindStringSubmatch(line)
					fmt.Println(matches)
					require.Len(t, matches, 2)
					targets = append(targets, matches[1])
				}
			}
			require.Len(t, targets, 2)

			// Now we need to reconstruct the slice of targets for the app nodes
			actualMMTargets := []string{}
			mmTargets := targets[0]
			if mmTargets != "" {
				actualMMTargets = strings.Split(mmTargets, ",")
			}

			// And the same for the agent nodes
			actualLTTargets := []string{}
			ltTargets := targets[1]
			if ltTargets != "" {
				actualLTTargets = strings.Split(ltTargets, ",")
			}

			// Now it's time to check the targets for the app nodes
			if tc.EnableAppProfiling {
				require.ElementsMatch(t, tc.MMTargets, actualMMTargets)
			} else {
				require.Empty(t, actualMMTargets)
			}

			// And again for the agent nodes
			if tc.EnableAgentProfiling {
				require.ElementsMatch(t, tc.LTTargets, actualLTTargets)
			} else {
				require.Empty(t, actualLTTargets)
			}
		})
	}

}
