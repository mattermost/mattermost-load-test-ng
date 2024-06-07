package main

import (
	"bytes"
	"testing"
)

func TestPrintResults(t *testing.T) {
	// Create a buffer to capture the output
	buf := &bytes.Buffer{}

	// Prepare test data
	results := []Result{
		{
			Report:       "Sample Report",
			DashboardURL: "http://example.com/dashboard",
			LoadTests: []LoadTestResult{
				{
					Label: "Test1",
					Config: LoadTestConfig{
						Type:     "bounded",
						DBEngine: "postgres",
						NumUsers: 100,
						Duration: "10m",
					},
					Status: LoadTestStatus{
						SupportedUsers: 80,
						NumErrors:      2,
					},
				},
			},
		},
	}
}
	// Call the function with the test data and the buffer
  printResults(results, buf)

	// Verify the output
	expectedOutput := `==================================================
Comparison result:
Report: Sample Report
Grafana Dashboard: http://example.com/dashboard
Test1:
  Type: bounded
  DB Engine: postgres
  Duration: 10m
  Users: 100
  Errors: 2
==================================================

`
	require.Equal(t, expectedOutput, buf.String())
}