// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package comparison

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/report"

	"github.com/mattermost/mattermost-server/v6/shared/mlog"
)

// DeploymentInfo holds information regarding a deployment.
type DeploymentInfo struct {
	// Number of application instances.
	AppInstanceCount int
	// Type of EC2 application instances.
	AppInstanceType string
	// Number of load-test agent instances.
	AgentInstanceCount int
	// Type of EC2 load-test agent instances.
	AgentInstanceType string
	// Type of EC2 proxy instance.
	ProxyInstanceType string
	// Number of RDS nodes.
	DBInstanceCount int
	// Type of RDS instance.
	DBInstanceType string
}

// LoadTestResult holds information regarding a load-test
// performed during a comparison.
type LoadTestResult struct {
	Label  string             // A label for the load-test.
	Config LoadTestConfig     // The config object associated with the load-test.
	Status coordinator.Status // The final status of the load-test.

	loadTestID int
}

// Results holds information regarding the results of an
// automated load-test comparison.
type Result struct {
	// An array of load-test results where the first element is the base run
	// and the second element is the new run.
	LoadTests [2]LoadTestResult
	// The Markdown report for the comparison.
	Report string
	// The URL to a comparative Grafana dashboard.
	DashboardURL string

	deploymentID string
}

type Output struct {
	// Information about the deployment in which the comparison ran.
	DeploymentInfo DeploymentInfo
	// A list of results.
	Results []Result
}

func getDeploymentInfo(config *deployment.Config) DeploymentInfo {
	return DeploymentInfo{
		AppInstanceCount:   config.AppInstanceCount,
		AppInstanceType:    config.AppInstanceType,
		AgentInstanceCount: config.AgentInstanceCount,
		AgentInstanceType:  config.AgentInstanceType,
		ProxyInstanceType:  config.ProxyInstanceType,
		DBInstanceCount:    config.TerraformDBSettings.InstanceCount,
		DBInstanceType:     config.TerraformDBSettings.InstanceType,
	}
}

func (c *Comparison) getResults(resultsCh <-chan Result) []Result {
	var wg sync.WaitGroup
	var results []Result
	resCh := make(chan Result, len(resultsCh))

	wg.Add(len(resultsCh))
	for res := range resultsCh {
		go func(res Result) {
			defer wg.Done()

			dp := c.deployments[res.deploymentID]
			t := terraform.New(res.deploymentID, &dp.config)
			output, err := t.Output()
			if err != nil {
				mlog.Error("Failed to get terraform output", mlog.Err(err))
				return
			}

			promURL := "http://" + output.MetricsServer.PublicIP + ":9090"
			helper, err := prometheus.NewHelper(promURL)
			if err != nil {
				mlog.Error("Failed to create prometheus.Helper", mlog.Err(err))
				return
			}
			g := report.New(res.LoadTests[0].Label, helper, dp.config.Report)
			baseReport, err := g.Generate(res.LoadTests[0].Status.StartTime, res.LoadTests[0].Status.StopTime)
			if err != nil {
				mlog.Error("Error while generating report", mlog.Err(err))
				return
			}
			g = report.New(res.LoadTests[1].Label, helper, dp.config.Report)
			newReport, err := g.Generate(res.LoadTests[1].Status.StartTime, res.LoadTests[1].Status.StopTime)
			if err != nil {
				mlog.Error("Error while generating report", mlog.Err(err))
				return
			}

			if c.config.Output.GenerateReport {
				var buf bytes.Buffer
				graphsPrefix := fmt.Sprintf("%s_%s_%d_", res.LoadTests[0].Config.DBEngine,
					res.LoadTests[0].Config.Type, res.LoadTests[0].loadTestID)
				opts := report.CompareOpts{
					GenGraph:     c.config.Output.GenerateGraphs,
					GraphsPrefix: graphsPrefix,
				}
				opts.GraphsPrefix = filepath.Join(c.config.Output.GraphsPath, opts.GraphsPrefix)

				err := report.Compare(&buf, opts, baseReport, newReport)
				if err != nil {
					mlog.Error("Failed to compare reports", mlog.Err(err))
				}
				res.Report = buf.String()
			}

			if c.config.Output.UploadDashboard {
				var dashboardData bytes.Buffer
				title := fmt.Sprintf("Comparison - %d - %s - %s",
					res.LoadTests[0].loadTestID, res.LoadTests[0].Config.DBEngine, res.LoadTests[0].Config.Type)
				if err := report.GenerateDashboard(title, baseReport, newReport, &dashboardData); err != nil {
					mlog.Error("Failed to generate dashboard", mlog.Err(err))
					return
				}

				url, err := t.UploadDashboard(dashboardData.String())
				if err != nil {
					mlog.Error("Failed to upload dashboard", mlog.Err(err))
					return
				}
				res.DashboardURL = fmt.Sprintf("http://%s:3000%s", output.MetricsServer.PublicIP, url)
			}

			resCh <- res
		}(res)
	}

	wg.Wait()
	close(resCh)
	for res := range resCh {
		results = append(results, res)
	}

	return results
}
