// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package report

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/mattermost/mattermost-server/v5/mlog"
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
	c := comp{
		store: make(map[model.LabelValue]avgp99),
		api:   make(map[model.LabelValue]avgp99),
	}

	// Calculate the deltas
	for _, r := range reports[1:] {
		// XXX: This can be somewhat refactored but whether absolute metrics
		// are useful or not needs to be seen.
		for label, value := range base.AvgStoreTimes {
			actual := float64(r.AvgStoreTimes[label])
			delta := actual - float64(value)
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
			actual := float64(r.P99StoreTimes[label])
			delta := actual - float64(value)
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
			actual := float64(r.AvgAPITimes[label])
			delta := actual - float64(value)
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
			actual := float64(r.P99APITimes[label])
			delta := actual - float64(value)
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

	// Now print them in markdown format.
	fmt.Fprintln(target, "### Store times in seconds:")
	printHeader(target, len(reports[1:]))

	keys := sortKeys(c.store)
	for _, label := range keys {
		measurement := c.store[label]
		fmt.Fprint(target, "| "+label)
		avg := measurement[0]
		p99 := measurement[1]

		fmt.Fprint(target, " |  Avg")
		fmt.Fprintf(target, "| %.3f", base.AvgStoreTimes[label])
		for i := 0; i < len(avg); i++ {
			fmt.Fprintf(target, "| %.3f | %.3f | %.3f", avg[i].actual, avg[i].delta, avg[i].deltaPercent)
		}
		fmt.Fprintln(target)

		fmt.Fprint(target, "| |  P99")
		fmt.Fprintf(target, "| %.3f", base.P99StoreTimes[label])
		for i := 0; i < len(p99); i++ {
			fmt.Fprintf(target, "| %.3f | %.3f | %.3f", p99[i].actual, p99[i].delta, p99[i].deltaPercent)
		}
		fmt.Fprintln(target)
	}

	fmt.Fprintln(target, "### API times in seconds:")
	printHeader(target, len(reports[1:]))

	keys = sortKeys(c.api)
	for _, label := range keys {
		measurement := c.api[label]
		fmt.Fprint(target, "| ", label)
		avg := measurement[0]
		p99 := measurement[1]

		fmt.Fprint(target, " | Avg")
		fmt.Fprintf(target, "| %.3f", base.AvgAPITimes[label])
		for i := 0; i < len(avg); i++ {
			fmt.Fprintf(target, "| %.3f | %.3f | %.3f", avg[i].actual, avg[i].delta, avg[i].deltaPercent)
		}
		fmt.Fprintln(target)

		fmt.Fprint(target, "| | P99")
		fmt.Fprintf(target, "| %.3f", base.P99APITimes[label])
		for i := 0; i < len(p99); i++ {
			fmt.Fprintf(target, "| %.3f | %.3f | %.3f", p99[i].actual, p99[i].delta, p99[i].deltaPercent)
		}
		fmt.Fprintln(target)
	}

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
			generateGraph(plot.name, base.Label, base.Graphs[i], plot.graphs)
		}
	}
	return nil
}

// generateGraph creates an input file for GNUplot to create a plot from.
func generateGraph(name, baseLabel string, base graph, others []labelValues) error {
	f, err := ioutil.TempFile("", "tmp.out")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	// Write the header
	fmt.Fprint(f, "# time, base")
	for i := range others {
		fmt.Fprintf(f, ", col%d", i)
	}
	fmt.Fprintln(f)
	// Do a row-wise traversal of the data.
	for i, pt := range base.Values {
		// Write the line of values separated by comma.
		fmt.Fprintf(f, "%d, %f", i, pt.Value)
		for _, r := range others {
			if i < len(r.values) {
				fmt.Fprintf(f, ", %f", r.values[i].Value)
			}
		}
		// Print a newline to go to the next timestamp.
		fmt.Fprintln(f)
	}
	err = f.Close()
	if err != nil {
		return err
	}

	err = plot(name, f.Name(), others, baseLabel)
	if err != nil {
		return fmt.Errorf("error while plotting %s graph: %s", name, err)
	}
	return nil
}

// printHeader prints the header row of a markdown table.
func printHeader(target *os.File, cols int) {
	fmt.Fprint(target, "| | | Base | ")
	header := ""
	for i := 0; i < cols; i++ {
		header += "Actual | Delta | Delta % |"
	}
	fmt.Fprintln(target, header)

	fmt.Fprint(target, "| --- | --- | --- | ")
	header = ""
	for i := 0; i < cols; i++ {
		header += "--- | --- | --- |"
	}
	fmt.Fprintln(target, header)
}

// plot creates a gnu plot file and then creates a png output file from it.
func plot(metric, fileName string, others []labelValues, baseLabel string) error {
	f, err := ioutil.TempFile("", "tmp.plt")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())

	imageFile := strings.Replace(strings.ToLower(metric), " ", "-", -1)
	fmt.Fprintln(f, "set grid")
	fmt.Fprintln(f, "set terminal png")
	fmt.Fprintf(f, "set output '%s.png'\n", imageFile)
	fmt.Fprintf(f, "set title '%s'\n", metric)
	fmt.Fprintln(f, "set xlabel 'time (normalized)'")

	// Just some gnuplot jargon to specify plot characteristics and the columns to compare.
	fmt.Fprintf(f, "plot '%s' u 1:2 w lp t '%s'", fileName, baseLabel)
	for i := 0; i < len(others); i++ {
		fmt.Fprintf(f, ", '%s' u 1:%d w lp t '%s'", fileName, i+3, others[i].label)
	}

	err = f.Close()
	if err != nil {
		return err
	}

	cmd := exec.Command("gnuplot", f.Name())
	_, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error while running gnuplot: %w", err)
	}
	mlog.Info("Wrote " + imageFile + ".png")
	return nil
}

func sortKeys(m map[model.LabelValue]avgp99) []model.LabelValue {
	var labels []model.LabelValue
	for key := range m {
		labels = append(labels, key)
	}
	sort.Slice(labels, func(i, j int) bool {
		return labels[i] < labels[j]
	})
	return labels
}
