// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package report

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"

	"github.com/prometheus/common/model"
)

// Config contains information needed to generate reports.
type Config struct {
	Label        string // Label to be used when querying Prometheus.
	GraphQueries []GraphQuery
}

// GraphQuery contains the query to be executed against a Prometheus instance
// to gather data for reports.
type GraphQuery struct {
	Name  string // A friendly name for the query.
	Query string // The actual Prometheus query to be executed.
}

// Generator is used to generate load test reports.
type Generator struct {
	label  string
	helper *prometheus.Helper
	cfg    Config
}

// Report contains the entire report data comprising of several metrics
// that are needed to compare load test runs.
type Report struct {
	Label         string // A friendly name of the report.
	StartTime     time.Time
	EndTime       time.Time
	AvgStoreTimes map[model.LabelValue]model.SampleValue
	P99StoreTimes map[model.LabelValue]model.SampleValue
	AvgAPITimes   map[model.LabelValue]model.SampleValue
	P99APITimes   map[model.LabelValue]model.SampleValue
	Graphs        []graph
}

// graph contains data for a single metric.
type graph struct {
	Name   string
	Values []model.SamplePair
}

// New returns a new instance of a generator.
func New(label string, helper *prometheus.Helper, cfg Config) *Generator {
	return &Generator{
		label:  label,
		helper: helper,
		cfg:    cfg,
	}
}

// Load loads a report from a given file path.
func Load(path string) (Report, error) {
	var r Report
	buf, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	err = json.Unmarshal(buf, &r)
	if err != nil {
		return r, err
	}
	return r, nil
}

// Generate returns a report from a given start time to end time.
func (g *Generator) Generate(startTime, endTime time.Time) (Report, error) {
	data := Report{
		Label:     g.label,
		StartTime: startTime,
		EndTime:   endTime,
	}

	var err error
	diff := endTime.Sub(startTime)
	sec := int(diff.Seconds())

	if sec <= 0 {
		return data, fmt.Errorf("duration should be greater than 0: %v %v", startTime, endTime)
	}

	// Avg store times.
	tmpl := `sum(rate(mattermost_db_store_time_sum%s[%ds])) by (method) / sum(rate(mattermost_db_store_time_count%s[%ds])) by (method)`
	query := fmt.Sprintf(tmpl, g.cfg.Label, sec, g.cfg.Label, sec)
	data.AvgStoreTimes, err = g.getValue(endTime, query, "method")
	if err != nil {
		return data, fmt.Errorf("error while getting avg store times: %w", err)
	}

	// P99 store times.
	tmpl = `histogram_quantile(0.99, sum(rate(mattermost_db_store_time_bucket%s[%ds])) by (le,method))`
	query = fmt.Sprintf(tmpl, g.cfg.Label, sec)
	data.P99StoreTimes, err = g.getValue(endTime, query, "method")
	if err != nil {
		return data, fmt.Errorf("error while getting p99 store times: %w", err)
	}

	// Avg API times.
	tmpl = `sum(rate(mattermost_api_time_sum%s[%ds])) by (handler) / sum(rate(mattermost_api_time_count%s[%ds])) by (handler)`
	query = fmt.Sprintf(tmpl, g.cfg.Label, sec, g.cfg.Label, sec)
	data.AvgAPITimes, err = g.getValue(endTime, query, "handler")
	if err != nil {
		return data, fmt.Errorf("error while getting avg API times: %w", err)
	}

	// P99 API times.
	tmpl = `histogram_quantile(0.99, sum(rate(mattermost_api_time_bucket%s[%ds])) by (handler,le))`
	query = fmt.Sprintf(tmpl, g.cfg.Label, sec)
	data.P99APITimes, err = g.getValue(endTime, query, "handler")
	if err != nil {
		return data, fmt.Errorf("error while getting p99 API times: %w", err)
	}

	for _, gq := range g.cfg.GraphQueries {
		res, err := g.helper.Matrix(gq.Query, startTime, endTime)
		if err != nil {
			return data, fmt.Errorf("error while getting %s: %w", gq.Name, err)
		}
		data.Graphs = append(data.Graphs, graph{
			Name:   gq.Name,
			Values: res[0].Values,
		})
	}

	return data, nil
}

// getValue returns a map of labels to the values for the last timestamp of a given query.
func (g *Generator) getValue(endTime time.Time, query, label string) (map[model.LabelValue]model.SampleValue, error) {
	// We just query from endTime-5s to endTime because the query already computes the values
	// from startTime to endTime.
	fmt.Println("AAA")
	res, err := g.helper.Matrix(query, endTime.Add(-5*time.Second), endTime)
	if err != nil {
		fmt.Println("BBB")
		return nil, err
	}

	// We only take the last value as that is the one that contains the value for the interval
	// we want.
	values := make(map[model.LabelValue]model.SampleValue)
	for _, sample := range res {
		metric := sample.Metric[model.LabelName(label)]
		val := sample.Values[len(sample.Values)-1]
		// We ignore metrics that don't have a value.
		if !math.IsNaN(float64(val.Value)) {
			values[metric] = val.Value
		}
	}
	return values, nil
}
