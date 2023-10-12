package comparison

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDeploymentIds(t *testing.T) {
	testCases := []struct {
		Name string
		Ids  []string
	}{
		{
			Name: "Empty comparison",
			Ids:  []string{},
		},
		{
			Name: "One element",
			Ids:  []string{"a"},
		},
		{
			Name: "Elements already ordered",
			Ids:  []string{"a", "b"},
		},
		{
			Name: "Unordered elements",
			Ids:  []string{"b", "a"},
		},
	}

	emptyCfg := deploymentConfig{}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			cmp := Comparison{}
			cmp.deployments = make(map[string]*deploymentConfig)
			for _, id := range tc.Ids {
				cmp.deployments[id] = &emptyCfg
			}

			expectedIds := make([]string, len(tc.Ids))
			copy(expectedIds, tc.Ids)
			sort.Strings(expectedIds)

			actualIds := cmp.GetDeploymentIds()
			require.EqualValues(t, expectedIds, actualIds)
		})
	}
}
