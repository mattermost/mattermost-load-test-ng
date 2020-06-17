// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package report

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"
	"github.com/prometheus/common/model"
)

// Generator is used to generate load test reports.
type Generator struct {
	label  string
	helper *prometheus.Helper
}

// Report contains the entire report data comprising of several metrics
// that are needed to compare load test runs.
type Report struct {
	Label         string // A friendly name of the report.
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
func New(label string, helper *prometheus.Helper) *Generator {
	return &Generator{
		label:  label,
		helper: helper,
	}
}

// Load loads a report from a given file path.
func Load(path string) (Report, error) {
	var r Report
	buf, err := ioutil.ReadFile(path)
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
		Label: g.label,
	}

	var err error
	diff := endTime.Sub(startTime)
	sec := int(diff.Seconds())

	// TODO: ability to filter the instances by a given label for cases
	// when a single Prometheus instance runs multiple clusters.

	// Avg store times.
	tmpl := `sum(increase(mattermost_db_store_time_sum[%ds])) by (method) / sum(increase(mattermost_db_store_time_count[%ds])) by (method)`
	query := fmt.Sprintf(tmpl, sec, sec)
	data.AvgStoreTimes, err = g.getValue(endTime, query, "method")
	if err != nil {
		return data, fmt.Errorf("error while getting avg store times: %w", err)
	}

	// P99 store times.
	tmpl = `histogram_quantile(0.99, sum(rate(mattermost_db_store_time_bucket[%ds])) by (le,method))`
	query = fmt.Sprintf(tmpl, sec)
	data.P99StoreTimes, err = g.getValue(endTime, query, "method")
	if err != nil {
		return data, fmt.Errorf("error while getting p99 store times: %w", err)
	}

	// Avg API times.
	tmpl = `sum(increase(mattermost_api_time_sum[%ds])) by (handler) / sum(increase(mattermost_api_time_count[%ds])) by (handler)`
	query = fmt.Sprintf(tmpl, sec, sec)
	data.AvgAPITimes, err = g.getValue(endTime, query, "handler")
	if err != nil {
		return data, fmt.Errorf("error while getting avg API times: %w", err)
	}

	// P99 API times.
	tmpl = `histogram_quantile(0.99, sum(rate(mattermost_api_time_bucket[%ds])) by (handler,le))`
	query = fmt.Sprintf(tmpl, sec)
	data.P99APITimes, err = g.getValue(endTime, query, "handler")
	if err != nil {
		return data, fmt.Errorf("error while getting p99 API times: %w", err)
	}

	// TODO: move them to be loadable from config so that more queries can be easily added later.
	graphQueries := []struct {
		Name  string
		Query string
	}{
		{
			Name:  "CPU Utilization",
			Query: `avg(irate(mattermost_process_cpu_seconds_total{instance=~"app.*"}[1m])* 100)`,
		},
		{
			Name:  "Heap In Use",
			Query: `avg(go_memstats_heap_inuse_bytes{instance=~"app.*:8067"})`,
		},
		{
			Name:  "Stack In Use",
			Query: `avg(go_memstats_stack_inuse_bytes{instance=~"app.*:8067"})`,
		},
		{
			Name:  "RPS",
			Query: `sum(rate(mattermost_http_requests_total{instance=~"app.*:8067"}[1m]))`,
		},
		{
			Name:  "Avg Store times",
			Query: `sum(increase(mattermost_db_store_time_sum{instance=~"app.*:8067"}[1m])) / sum(increase(mattermost_db_store_time_count{instance=~"app.*:8067"}[1m]))`,
		},
	}

	for _, gq := range graphQueries {
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
	res, err := g.helper.Matrix(query, endTime.Add(-5*time.Second), endTime)
	if err != nil {
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
