// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package report

import (
	"fmt"
	"os"

	"github.com/prometheus/common/model"
)

// diff contains the differences from a base measurement.
type diff struct {
	actual       float64
	delta        float64
	deltaPercent float64
}

// avgp99 is an array of 2 slices.
// The array is for avg and p99 measurements.
// The slice is used to contain measurements from multiple reports so that they can be
// displayed side-by-side.
type avgp99 [2][]diff

type comp struct {
	store map[model.LabelValue]avgp99
	api   map[model.LabelValue]avgp99
}

// labelValues is used to compare a single metric from different load tests.
type labelValues struct {
	label  string             // The label of a report.
	values []model.SamplePair // The series of values for a given metric.
}

// Compare compares the given set of reports.
// The first report is considered to be the base.
func Compare(target *os.File, genGraph bool, reports ...Report) error {
	base := reports[0]

	// Calculate the deltas.
	c := calculateDeltas(reports...)

	// Now display the comparison in markdown.
	displayMarkdown(c, target, base, len(reports[1:]))

	// TODO: generate a single image combining all the graphs.
	// Printing the graphs.
	if genGraph {
		// gPlots is a slice of structs to aggregate graphs of a single type
		// from multiple reports.
		var gPlots []struct {
			name   string // the name of the metric
			graphs []labelValues
		}
		// A single report has multiple graphs.
		// What we are doing here is changing the aggregation such that
		// a single gPlot contains graphs of a single type from multiple reports.
		for i, r := range reports[1:] {
			for _, g := range r.Graphs {
				if i == 0 {
					gPlots = append(gPlots, struct {
						name   string
						graphs []labelValues
					}{
						// Only set the name one time
						name: g.Name,
						graphs: []labelValues{
							{
								label:  r.Label,
								values: g.Values,
							},
						},
					})
				} else {
					// Graph of the same type from another report, append it.
					gPlots[i].graphs = append(gPlots[i].graphs, labelValues{
						label:  r.Label,
						values: g.Values,
					})
				}
			}
		}

		for i, plot := range gPlots {
			err := generateGraph(plot.name, base.Label, base.Graphs[i], plot.graphs)
			if err != nil {
				return fmt.Errorf("error while generating graph for %s: %w", plot.name, err)
			}
		}
	}
	return nil
}

// calculateDeltas returns a comparison from a given set of reports.
func calculateDeltas(reports ...Report) comp {
	base := reports[0]
	c := comp{
		store: make(map[model.LabelValue]avgp99),
		api:   make(map[model.LabelValue]avgp99),
	}
	for _, r := range reports[1:] {
		// XXX: This can be somewhat refactored but whether absolute metrics
		// are useful or not needs to be seen.
		for label, value := range base.AvgStoreTimes {
			actual := roundTo3DecimalPlaces(float64(r.AvgStoreTimes[label]))
			delta := actual - roundTo3DecimalPlaces(float64(value))
			deltaP := (delta / actual) * 100

			diffs := c.store[label]
			diffs[0] = append(diffs[0], diff{
				actual:       actual,
				delta:        delta,
				deltaPercent: deltaP,
			})
			c.store[label] = diffs
		}

		for label, value := range base.P99StoreTimes {
			actual := roundTo3DecimalPlaces(float64(r.P99StoreTimes[label]))
			delta := actual - roundTo3DecimalPlaces(float64(value))
			deltaP := (delta / actual) * 100

			diffs := c.store[label]
			diffs[1] = append(diffs[1], diff{
				actual:       actual,
				delta:        delta,
				deltaPercent: deltaP,
			})
			c.store[label] = diffs
		}

		for label, value := range base.AvgAPITimes {
			actual := roundTo3DecimalPlaces(float64(r.AvgAPITimes[label]))
			delta := actual - roundTo3DecimalPlaces(float64(value))
			deltaP := (delta / actual) * 100

			diffs := c.api[label]
			diffs[0] = append(diffs[0], diff{
				actual:       actual,
				delta:        delta,
				deltaPercent: deltaP,
			})
			c.api[label] = diffs
		}

		for label, value := range base.P99APITimes {
			actual := roundTo3DecimalPlaces(float64(r.P99APITimes[label]))
			delta := actual - roundTo3DecimalPlaces(float64(value))
			deltaP := (delta / actual) * 100

			diffs := c.api[label]
			diffs[1] = append(diffs[1], diff{
				actual:       actual,
				delta:        delta,
				deltaPercent: deltaP,
			})
			c.api[label] = diffs
		}
	}
	return c
}
