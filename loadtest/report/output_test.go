// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package report

import (
	"os"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
)

func TestSortKeys(t *testing.T) {
	require.NotPanics(t, func() {
		sortKeys(map[model.LabelValue]avgp99{
			"store_metric": [2][]diff{
				{
				},
				{
					{
						base:         time.Second,
						actual:       2 * time.Second,
						delta:        1 * time.Second,
						deltaPercent: 100,
					},
				},
			},
			"another_metric": [2][]diff{
				{
					{
						base:         time.Millisecond,
						actual:       2 * time.Millisecond,
						delta:        1 * time.Millisecond,
						deltaPercent: 100,
					},
				},
				{
					{
						base:         time.Second,
						actual:       2 * time.Second,
						delta:        1 * time.Second,
						deltaPercent: 100,
					},
				},
			},
		}, sortByAvg, true)
	})

	require.NotPanics(t, func() {
		sortKeys(map[model.LabelValue]avgp99{
			"store_metric": [2][]diff{
				{
					{
						base:         time.Millisecond,
						actual:       2 * time.Millisecond,
						delta:        1 * time.Millisecond,
						deltaPercent: 100,
					},
				},
				{
				},
			},
			"another_metric": [2][]diff{
				{
					{
						base:         time.Millisecond,
						actual:       2 * time.Millisecond,
						delta:        1 * time.Millisecond,
						deltaPercent: 100,
					},
				},
				{
					{
						base:         time.Second,
						actual:       2 * time.Second,
						delta:        1 * time.Second,
						deltaPercent: 100,
					},
				},
			},
		}, sortByP99, true)
	})
}

func TestPrintSummary(t *testing.T) {
	f, err := os.CreateTemp("", "output")
	require.Nil(t, err)
	defer os.Remove(f.Name())

	cases := []struct {
		name           string
		cmp            comp
		expectedOutput string
	}{
		{
			"empty comp",
			comp{},
			`### Store times avg (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### Store times p99 (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### Store times avg (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### Store times p99 (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### API times avg (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### API times p99 (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### API times avg (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### API times p99 (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
`,
		},
		{
			"skip small deltas",
			comp{
				store: map[model.LabelValue]avgp99{
					"store_metric": [2][]diff{
						{
							{
								base:         time.Millisecond,
								actual:       2 * time.Millisecond,
								delta:        1 * time.Millisecond,
								deltaPercent: 100,
							},
						},
						{
							{
								base:         time.Second,
								actual:       2 * time.Second,
								delta:        1 * time.Second,
								deltaPercent: 100,
							},
						},
					},
				},
				api: map[model.LabelValue]avgp99{
					"api_metric": [2][]diff{
						{
							{
								base:         time.Millisecond,
								actual:       3 * time.Millisecond,
								delta:        2 * time.Millisecond,
								deltaPercent: 200,
							},
						},
						{
							{
								base:         time.Second,
								actual:       3 * time.Second,
								delta:        2 * time.Second,
								deltaPercent: 200,
							},
						},
					},
				},
			},
			`### Store times avg (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### Store times p99 (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
| store_metric | p99 | 1s | 2s | 1s | 100.00
### Store times avg (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### Store times p99 (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### API times avg (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
| api_metric | avg | 1ms | 3ms | 2ms | 200.00
### API times p99 (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
| api_metric | p99 | 1s | 3s | 2s | 200.00
### API times avg (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### API times p99 (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
`,
		},
		{
			"skip small delta percentages",
			comp{
				store: map[model.LabelValue]avgp99{
					"store_metric": [2][]diff{
						{
							{
								base:         100 * time.Millisecond,
								actual:       95 * time.Millisecond,
								delta:        5 * time.Millisecond,
								deltaPercent: 0.5,
							},
						},
						{
							{
								base:         time.Second,
								actual:       2 * time.Second,
								delta:        1 * time.Second,
								deltaPercent: 100,
							},
						},
					},
				},
				api: map[model.LabelValue]avgp99{
					"api_metric": [2][]diff{
						{
							{
								base:         100 * time.Millisecond,
								actual:       95 * time.Millisecond,
								delta:        5 * time.Millisecond,
								deltaPercent: 0.5,
							},
						},
						{
							{
								base:         time.Second,
								actual:       3 * time.Second,
								delta:        2 * time.Second,
								deltaPercent: 200,
							},
						},
					},
				},
			},
			`### Store times avg (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### Store times p99 (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
| store_metric | p99 | 1s | 2s | 1s | 100.00
### Store times avg (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### Store times p99 (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### API times avg (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### API times p99 (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
| api_metric | p99 | 1s | 3s | 2s | 200.00
### API times avg (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### API times p99 (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
`,
		},
		{
			"worsened",
			comp{
				store: map[model.LabelValue]avgp99{
					"store_metric": [2][]diff{
						{
							{
								base:         time.Millisecond,
								actual:       3 * time.Millisecond,
								delta:        2 * time.Millisecond,
								deltaPercent: 200,
							},
						},
						{
							{
								base:         time.Second,
								actual:       3 * time.Second,
								delta:        2 * time.Second,
								deltaPercent: 200,
							},
						},
					},
				},
				api: map[model.LabelValue]avgp99{
					"api_metric": [2][]diff{
						{
							{
								base:         time.Millisecond,
								actual:       3 * time.Millisecond,
								delta:        2 * time.Millisecond,
								deltaPercent: 200,
							},
						},
						{
							{
								base:         time.Second,
								actual:       3 * time.Second,
								delta:        2 * time.Second,
								deltaPercent: 200,
							},
						},
					},
				},
			},
			`### Store times avg (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
| store_metric | avg | 1ms | 3ms | 2ms | 200.00
### Store times p99 (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
| store_metric | p99 | 1s | 3s | 2s | 200.00
### Store times avg (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### Store times p99 (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### API times avg (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
| api_metric | avg | 1ms | 3ms | 2ms | 200.00
### API times p99 (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
| api_metric | p99 | 1s | 3s | 2s | 200.00
### API times avg (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### API times p99 (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
`,
		},
		{
			"improved and worsened",
			comp{
				store: map[model.LabelValue]avgp99{
					"store_metric": [2][]diff{
						{
							{
								base:         3 * time.Millisecond,
								actual:       time.Millisecond,
								delta:        -2 * time.Millisecond,
								deltaPercent: -200,
							},
						},
						{
							{
								base:         3 * time.Millisecond,
								actual:       time.Millisecond,
								delta:        -2 * time.Millisecond,
								deltaPercent: -200,
							},
						},
					},
				},
				api: map[model.LabelValue]avgp99{
					"api_metric": [2][]diff{
						{
							{
								base:         time.Millisecond,
								actual:       3 * time.Millisecond,
								delta:        2 * time.Millisecond,
								deltaPercent: 200,
							},
						},
						{
							{
								base:         time.Second,
								actual:       3 * time.Second,
								delta:        2 * time.Second,
								deltaPercent: 200,
							},
						},
					},
				},
			},
			`### Store times avg (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### Store times p99 (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### Store times avg (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
| store_metric | avg | 3ms | 1ms | -2ms | -200.00
### Store times p99 (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
| store_metric | p99 | 3ms | 1ms | -2ms | -200.00
### API times avg (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
| api_metric | avg | 1ms | 3ms | 2ms | 200.00
### API times p99 (worsened):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
| api_metric | p99 | 1s | 3s | 2s | 200.00
### API times avg (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
### API times p99 (improved):
| | | Base | Actual | Delta | Delta % |
| --- | --- | --- | --- | --- | --- |
`,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			err := os.Truncate(f.Name(), 0)
			require.Nil(t, err)
			_, err = f.Seek(0, os.SEEK_SET)
			require.Nil(t, err)
			printSummary(c.cmp, f, 1)
			output, err := os.ReadFile(f.Name())
			require.Nil(t, err)
			require.Equal(t, c.expectedOutput, string(output))
		})
	}
}
