//go:generate go-bindata -nometadata -mode 0644 -pkg report -o ./bindata.go -prefix "assets/" assets/
// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package report

import (
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/prometheus/common/model"
)

type sortByType int

const (
	sortByLabel sortByType = iota
	sortByAvg
	sortByP99
)

func printSummary(c comp, target io.Writer, base Report, cols int) {
	printTimes := func(data map[model.LabelValue]avgp99, metric string, showImproved bool) {
		var keys []model.LabelValue
		var sortBy sortByType
		if metric == "avg" {
			sortBy = sortByAvg
		} else if metric == "p99" {
			sortBy = sortByP99
		}
		if showImproved {
			keys = sortKeys(data, sortBy, false)
		} else {
			keys = sortKeys(data, sortBy, true)
		}

		for _, label := range keys {
			measurement := data[label]
			var d []diff
			if metric == "avg" {
				d = measurement[0]
			} else if metric == "p99" {
				d = measurement[1]
			}
			if len(d) != 1 {
				break
			}
			if (showImproved && d[0].deltaPercent >= 0) || (!showImproved && d[0].deltaPercent <= 0) {
				break
			}
			fmt.Fprintf(target, "| %s", label)
			fmt.Fprintf(target, " |  %s", metric)
			fmt.Fprintf(target, "| %s", getDuration(float64(base.AvgStoreTimes[label])))
			fmt.Fprintf(target, "| %s | %s | %.3f", d[0].actual, d[0].delta, d[0].deltaPercent)
			fmt.Fprintln(target)
		}
	}

	fmt.Fprintln(target, "### Store times avg (worsened):")
	printHeader(target, cols)
	printTimes(c.store, "avg", false)
	fmt.Fprintln(target, "### Store times p99 (worsened):")
	printHeader(target, cols)
	printTimes(c.store, "p99", false)

	fmt.Fprintln(target, "### Store times avg (improved):")
	printHeader(target, cols)
	printTimes(c.store, "avg", true)
	fmt.Fprintln(target, "### Store times p99 (improved):")
	printHeader(target, cols)
	printTimes(c.store, "p99", true)

	fmt.Fprintln(target, "### API times avg (worsened):")
	printHeader(target, cols)
	printTimes(c.api, "avg", false)
	fmt.Fprintln(target, "### API times p99 (worsened):")
	printHeader(target, cols)
	printTimes(c.api, "p99", false)

	fmt.Fprintln(target, "### API times avg (improved):")
	printHeader(target, cols)
	printTimes(c.api, "avg", true)
	fmt.Fprintln(target, "### API times p99 (improved):")
	printHeader(target, cols)
	printTimes(c.api, "p99", true)
}

// displayMarkdown prints a given comparison in markdown to the given target.
func displayMarkdown(c comp, target io.Writer, base Report, cols int) {
	printSummary(c, target, base, cols)

	fmt.Fprintln(target, "### Store times:")
	printHeader(target, cols)

	keys := sortKeys(c.store, sortByLabel, false)
	for _, label := range keys {
		measurement := c.store[label]
		fmt.Fprint(target, "| "+label)
		avg := measurement[0]
		p99 := measurement[1]

		fmt.Fprint(target, " |  Avg")
		fmt.Fprintf(target, "| %s", getDuration(float64(base.AvgStoreTimes[label])))
		for i := 0; i < len(avg); i++ {
			fmt.Fprintf(target, "| %s | %s | %.3f", avg[i].actual, avg[i].delta, avg[i].deltaPercent)
		}
		fmt.Fprintln(target)

		fmt.Fprint(target, "| |  P99")
		fmt.Fprintf(target, "| %s", getDuration(float64(base.P99StoreTimes[label])))
		for i := 0; i < len(p99); i++ {
			fmt.Fprintf(target, "| %s | %s | %.3f", p99[i].actual, p99[i].delta, p99[i].deltaPercent)
		}
		fmt.Fprintln(target)
	}

	fmt.Fprintln(target, "### API times:")
	printHeader(target, cols)

	keys = sortKeys(c.api, sortByLabel, false)
	for _, label := range keys {
		measurement := c.api[label]
		fmt.Fprint(target, "| ", label)
		avg := measurement[0]
		p99 := measurement[1]

		fmt.Fprint(target, " | Avg")
		fmt.Fprintf(target, "| %s", getDuration(float64(base.AvgAPITimes[label])))
		for i := 0; i < len(avg); i++ {
			fmt.Fprintf(target, "| %s | %s | %.3f", avg[i].actual, avg[i].delta, avg[i].deltaPercent)
		}
		fmt.Fprintln(target)

		fmt.Fprint(target, "| | P99")
		fmt.Fprintf(target, "| %s", getDuration(float64(base.P99APITimes[label])))
		for i := 0; i < len(p99); i++ {
			fmt.Fprintf(target, "| %s | %s | %.3f", p99[i].actual, p99[i].delta, p99[i].deltaPercent)
		}
		fmt.Fprintln(target)
	}
}

// printHeader prints the header row of a markdown table.
func printHeader(target io.Writer, cols int) {
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
	fmt.Fprintf(f, "plot '%s' u 1:2 w l lw 2 t '%s'", fileName, baseLabel)
	for i := 0; i < len(others); i++ {
		fmt.Fprintf(f, ", '%s' u 1:%d w l lw 2 t '%s'", fileName, i+3, others[i].label)
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

func sortKeys(m map[model.LabelValue]avgp99, t sortByType, desc bool) []model.LabelValue {
	var labels []model.LabelValue
	for key := range m {
		labels = append(labels, key)
	}
	sort.Slice(labels, func(i, j int) bool {
		if desc {
			i, j = j, i
		}
		switch t {
		case sortByLabel:
			return labels[i] < labels[j]
		case sortByAvg:
			return m[labels[i]][0][0].deltaPercent < m[labels[j]][0][0].deltaPercent
		case sortByP99:
			return m[labels[i]][1][0].deltaPercent < m[labels[j]][1][0].deltaPercent
		default:
			return labels[i] < labels[j]
		}
	})
	return labels
}

func getDuration(f float64) time.Duration {
	ms := int64(math.Round(f * 1000))    // convert seconds to ms and round it off.
	d := time.Duration(ms * 1000 * 1000) // convert to ns for duration.
	return d
}

func GenerateDashboard(title string, baseReport, newReport Report, out io.Writer) error {
	baseLabel := baseReport.Label
	newLabel := newReport.Label
	from := newReport.StartTime
	to := newReport.EndTime
	offset := newReport.StartTime.Sub(baseReport.StartTime)

	// We swap everything if it happens that the load-tests were done in the
	// inverse than expected order (next then base).
	if baseReport.StartTime.After(newReport.StartTime) {
		from = baseReport.StartTime
		to = baseReport.EndTime
		offset = baseReport.StartTime.Sub(newReport.StartTime)
		baseLabel, newLabel = newLabel, baseLabel
	}

	tmpl, err := template.New("").Parse(MustAssetString("comparison.tmpl.json"))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}
	data := map[string]interface{}{
		"title":     title,
		"offset":    int(math.Round(offset.Seconds())),
		"baseLabel": baseLabel,
		"newLabel":  newLabel,
		"from":      from.Format(time.RFC3339),
		"to":        to.Format(time.RFC3339),
	}
	if err := tmpl.Execute(out, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
