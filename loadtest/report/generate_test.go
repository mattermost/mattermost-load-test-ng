// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package report

import (
	"context"
	"testing"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"

	apiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAPI struct {
	dataMap map[string]model.Matrix
}

func (m mockAPI) Query(ctx context.Context, query string, ts time.Time) (model.Value, apiv1.Warnings, error) {
	return model.Vector{}, apiv1.Warnings{}, nil
}

func (m mockAPI) QueryRange(ctx context.Context, query string, r apiv1.Range) (model.Value, apiv1.Warnings, error) {
	return m.dataMap[query], apiv1.Warnings{}, nil
}

func TestGenerate(t *testing.T) {
	var modelNow = model.Now()
	var values = []model.SamplePair{
		{
			Timestamp: modelNow,
			Value:     0.01,
		},
		{
			Timestamp: modelNow,
			Value:     0.02,
		},
	}
	var apiMap = map[model.LabelValue]model.SampleValue{
		"handler1": 0.01,
		"handler2": 0.02,
	}
	var storeMap = map[model.LabelValue]model.SampleValue{
		"method1": 0.01,
		"method2": 0.02,
	}

	var input = map[string]model.Matrix{
		"sum(rate(mattermost_db_store_time_sum[10s])) by (method) / sum(rate(mattermost_db_store_time_count[10s])) by (method)": {
			&model.SampleStream{
				Metric: model.Metric{
					"method": "method1",
				},
				Values: []model.SamplePair{
					{
						Timestamp: model.Time(time.Now().Unix()),
						Value:     0.01,
					},
				},
			},
			&model.SampleStream{
				Metric: model.Metric{
					"method": "method2",
				},
				Values: []model.SamplePair{
					{
						Timestamp: model.Time(time.Now().Unix()),
						Value:     0.02,
					},
				},
			},
		},
		"histogram_quantile(0.99, sum(rate(mattermost_db_store_time_bucket[10s])) by (le,method))": {
			&model.SampleStream{
				Metric: model.Metric{
					"method": "method1",
				},
				Values: []model.SamplePair{
					{
						Timestamp: model.Time(time.Now().Unix()),
						Value:     0.01,
					},
				},
			},
			&model.SampleStream{
				Metric: model.Metric{
					"method": "method2",
				},
				Values: []model.SamplePair{
					{
						Timestamp: model.Time(time.Now().Unix()),
						Value:     0.02,
					},
				},
			},
		},
		"sum(rate(mattermost_api_time_sum[10s])) by (handler) / sum(rate(mattermost_api_time_count[10s])) by (handler)": {
			&model.SampleStream{
				Metric: model.Metric{
					"handler": "handler1",
				},
				Values: []model.SamplePair{
					{
						Timestamp: model.Time(time.Now().Unix()),
						Value:     0.01,
					},
				},
			},
			&model.SampleStream{
				Metric: model.Metric{
					"handler": "handler2",
				},
				Values: []model.SamplePair{
					{
						Timestamp: model.Time(time.Now().Unix()),
						Value:     0.02,
					},
				},
			},
		},
		"histogram_quantile(0.99, sum(rate(mattermost_api_time_bucket[10s])) by (handler,le))": {
			&model.SampleStream{
				Metric: model.Metric{
					"handler": "handler1",
				},
				Values: []model.SamplePair{
					{
						Timestamp: model.Time(time.Now().Unix()),
						Value:     0.01,
					},
				},
			},
			&model.SampleStream{
				Metric: model.Metric{
					"handler": "handler2",
				},
				Values: []model.SamplePair{
					{
						Timestamp: model.Time(time.Now().Unix()),
						Value:     0.02,
					},
				},
			},
		},
		`avg(irate(mattermost_process_cpu_seconds_total{instance=~"app.*"}[1m])* 100)`: {
			&model.SampleStream{
				Metric: model.Metric{},
				Values: values,
			},
		},
		`avg(go_memstats_heap_inuse_bytes{instance=~"app.*:8067"})`: {
			&model.SampleStream{
				Metric: model.Metric{},
				Values: values,
			},
		},
		`avg(go_memstats_stack_inuse_bytes{instance=~"app.*:8067"})`: {
			&model.SampleStream{
				Metric: model.Metric{},
				Values: values,
			},
		},
		`sum(rate(mattermost_http_requests_total{instance=~"app.*:8067"}[1m]))`: {
			&model.SampleStream{
				Metric: model.Metric{},
				Values: values,
			},
		},
		`sum(rate(mattermost_db_store_time_sum{instance=~"app.*:8067"}[1m])) / sum(rate(mattermost_db_store_time_count{instance=~"app.*:8067"}[1m]))`: {
			&model.SampleStream{
				Metric: model.Metric{},
				Values: values,
			},
		},
	}

	cfg := Config{
		Label: "",
		GraphQueries: []GraphQuery{
			{
				Name:  "CPU Utilization",
				Query: "avg(irate(mattermost_process_cpu_seconds_total{instance=~\"app.*\"}[1m])* 100)",
			},
			{
				Name:  "Heap In Use",
				Query: "avg(go_memstats_heap_inuse_bytes{instance=~\"app.*:8067\"})",
			},
			{
				Name:  "Stack In Use",
				Query: "avg(go_memstats_stack_inuse_bytes{instance=~\"app.*:8067\"})",
			},
			{
				Name:  "RPS",
				Query: "sum(rate(mattermost_http_requests_total{instance=~\"app.*:8067\"}[1m]))",
			},
			{
				Name:  "Avg Store times",
				Query: "sum(rate(mattermost_db_store_time_sum{instance=~\"app.*:8067\"}[1m])) / sum(rate(mattermost_db_store_time_count{instance=~\"app.*:8067\"}[1m]))",
			},
		},
	}

	label := "base"
	var output = Report{
		Label:         label,
		AvgStoreTimes: storeMap,
		P99StoreTimes: storeMap,
		AvgAPITimes:   apiMap,
		P99APITimes:   apiMap,
		Graphs: []graph{
			{
				Name:   "CPU Utilization",
				Values: values,
			},
			{
				Name:   "Heap In Use",
				Values: values,
			},
			{
				Name:   "Stack In Use",
				Values: values,
			},
			{
				Name:   "RPS",
				Values: values,
			},
			{
				Name:   "Avg Store times",
				Values: values,
			},
		},
	}

	helper := &prometheus.Helper{}
	helper.SetAPI(mockAPI{
		dataMap: input,
	})

	g := New(label, helper, cfg)
	now := time.Now()
	r, err := g.Generate(now.Add(-10*time.Second), now)
	require.NoError(t, err)
	assert.Equal(t, output, r, "incorrect report generated")
}
