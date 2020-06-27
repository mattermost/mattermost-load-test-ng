// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package report

import (
	"testing"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPlots(t *testing.T) {
	modelNow := model.Now()
	r1 := Report{
		Label: "report1",
		Graphs: []graph{
			{
				Name: "CPU",
				Values: []model.SamplePair{
					{
						Timestamp: modelNow,
						Value:     0.01,
					},
					{
						Timestamp: modelNow,
						Value:     0.02,
					},
				},
			},
			{
				Name: "Memory",
				Values: []model.SamplePair{
					{
						Timestamp: modelNow,
						Value:     0.01,
					},
					{
						Timestamp: modelNow,
						Value:     0.02,
					},
				},
			},
			{
				Name: "Store",
				Values: []model.SamplePair{
					{
						Timestamp: modelNow,
						Value:     0.01,
					},
					{
						Timestamp: modelNow,
						Value:     0.02,
					},
				},
			},
		},
	}

	r2 := Report{
		Label: "report2",
		Graphs: []graph{
			{
				Name: "CPU",
				Values: []model.SamplePair{
					{
						Timestamp: modelNow,
						Value:     1.01,
					},
					{
						Timestamp: modelNow,
						Value:     1.02,
					},
				},
			},
			{
				Name: "Memory",
				Values: []model.SamplePair{
					{
						Timestamp: modelNow,
						Value:     1.01,
					},
					{
						Timestamp: modelNow,
						Value:     1.02,
					},
				},
			},
			{
				Name: "Store",
				Values: []model.SamplePair{
					{
						Timestamp: modelNow,
						Value:     1.01,
					},
					{
						Timestamp: modelNow,
						Value:     1.02,
					},
				},
			},
		},
	}

	r3 := Report{
		Label: "report3",
		Graphs: []graph{
			{
				Name: "CPU",
				Values: []model.SamplePair{
					{
						Timestamp: modelNow,
						Value:     2.01,
					},
					{
						Timestamp: modelNow,
						Value:     2.02,
					},
				},
			},
			{
				Name: "Memory",
				Values: []model.SamplePair{
					{
						Timestamp: modelNow,
						Value:     2.01,
					},
					{
						Timestamp: modelNow,
						Value:     2.02,
					},
				},
			},
			{
				Name: "Store",
				Values: []model.SamplePair{
					{
						Timestamp: modelNow,
						Value:     2.01,
					},
					{
						Timestamp: modelNow,
						Value:     2.02,
					},
				},
			},
		},
	}

	gPlots := getPlots(r1, r2, r3)
	require.Len(t, gPlots, 3)
	for i, plot := range gPlots {
		switch i {
		case 0:
			assert.Equal(t, "CPU", plot.name)
			require.Len(t, plot.graphs, 2)
			assert.Equal(t, "report2", plot.graphs[0].label)
			assert.Equal(t, "report3", plot.graphs[1].label)
		case 1:
			assert.Equal(t, "Memory", plot.name)
			require.Len(t, plot.graphs, 2)
			assert.Equal(t, "report2", plot.graphs[0].label)
			assert.Equal(t, "report3", plot.graphs[1].label)
		case 2:
			assert.Equal(t, "Store", plot.name)
			require.Len(t, plot.graphs, 2)
			assert.Equal(t, "report2", plot.graphs[0].label)
			assert.Equal(t, "report3", plot.graphs[1].label)
		}
	}
}
