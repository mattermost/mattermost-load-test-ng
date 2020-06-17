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

// displayMarkdown prints a given comparison in markdown to the given target.
func displayMarkdown(c comp, target *os.File, base Report, cols int) {
	fmt.Fprintln(target, "### Store times in seconds:")
	printHeader(target, cols)

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
	printHeader(target, cols)

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
