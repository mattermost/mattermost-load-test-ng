package main

import (
	"bytes"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/comparison"
	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/stretchr/testify/require"
)

func TestPrintResults(t *testing.T) {
	// Create a buffer to capture the output
	buf := &bytes.Buffer{}

	// Prepare test data
	results := []comparison.Result{
		{
			Report:       "Sample Report",
			DashboardURL: "http://example.com/dashboard",
			LoadTests: [2]comparison.LoadTestResult{
				{
					Label: "Test1",
					Config: comparison.LoadTestConfig{
						Type:     "bounded",
						DBEngine: "postgres",
						NumUsers: 100,
						Duration: "10m",
					},
					Status: coordinator.Status{
						SupportedUsers: 80,
						NumErrors:      2,
					},
				},
			},
		},
	}
	// Call the function with the test data and the buffer
	printResults(results, buf)

	// Verify the output
	expectedOutput := `==================================================Comparison result:Report: report_0_postgres_bounded_100.md
Grafana Dashboard: http://example.com/dashboard\nTest:1\nType: bounded
  DB Engine: postgres
  Duration: 10m
  Users: 100
  Errors: 2
==================================================

`
	require.Equal(t, expectedOutput, buf.String())
}
