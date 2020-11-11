// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package comparison

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/report"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

type deploymentInfo struct {
	config    deployment.Config
	loadTests []LoadTestConfig
}

// Comparison holds the state needed to perform automated
// load-test comparisons.
type Comparison struct {
	config      *Config
	deployments map[string]*deploymentInfo
	cancelCh    chan struct{}
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

// New creates and initializes a new Comparison object to be used to run an
// automated load-test comparison. It returns an error in case of failure.
func New(cfg *Config, deployerCfg *deployment.Config) (*Comparison, error) {
	if err := defaults.Validate(cfg); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	cmp := &Comparison{
		config:      cfg,
		deployments: map[string]*deploymentInfo{},
		cancelCh:    make(chan struct{}),
	}

	var i int
	deployments := map[string]string{}
	for _, lt := range cfg.LoadTests {
		key := string(lt.Type) + string(lt.DBEngine)
		var dp *deploymentInfo
		if id, ok := deployments[key]; !ok {
			dp = &deploymentInfo{config: *deployerCfg}
			dp.config.DBInstanceEngine = "aurora-" + string(lt.DBEngine)
			dp.config.ClusterName = fmt.Sprintf("%s%d", dp.config.ClusterName, i)
			id = fmt.Sprintf("deployment%d", i)
			cmp.deployments[id] = dp
			deployments[key] = id
			i++
		} else {
			dp = cmp.deployments[id]
		}
		dp.loadTests = append(dp.loadTests, lt)
	}

	return cmp, nil
}

func (c *Comparison) deploymentAction(action func(t *terraform.Terraform) error) error {
	var wg sync.WaitGroup
	wg.Add(len(c.deployments))
	errsCh := make(chan error, len(c.deployments))
	for id, dp := range c.deployments {
		go func(id string, dp *deploymentInfo) {
			defer wg.Done()
			t := terraform.New(id, &dp.config)
			defer t.Cleanup()
			if err := action(t); err != nil {
				errsCh <- fmt.Errorf("deployment action failed: %w", err)
			}
		}(id, dp)
	}
	wg.Wait()
	close(errsCh)
	var err error
	for err = range errsCh {
		mlog.Error(err.Error())
	}
	return err
}

func (c *Comparison) getLoadTestsCount() int {
	var count int
	for _, dp := range c.deployments {
		for range dp.loadTests {
			count++
		}
	}
	return count
}

func runBoundedLoadTest(t *terraform.Terraform, coordConfig *coordinator.Config, d time.Duration, cancelCh <-chan struct{}) (coordinator.Status, error) {
	var err error
	var status coordinator.Status
	mlog.Info("starting bounded load-test")
	if err := t.StartCoordinator(coordConfig); err != nil {
		return status, err
	}

	var canceled bool
	select {
	case <-cancelCh:
		mlog.Info("cancelling load-test")
		canceled = true
	case <-time.After(d):
	}

	mlog.Info("stopping bounded load-test")
	status, err = t.StopCoordinator()
	if err != nil {
		return status, err
	}

	if canceled {
		return status, errors.New("canceled")
	}

	// TODO: remove this once MM-30326 has been merged and a new release
	// published.
	status.StopTime = time.Now()

	return status, nil
}

func runUnboundedLoadTest(t *terraform.Terraform, coordConfig *coordinator.Config, cancelCh <-chan struct{}) (coordinator.Status, error) {
	mlog.Info("starting unbounded load-test")
	if err := t.StartCoordinator(coordConfig); err != nil {
		return coordinator.Status{}, err
	}

	for {
		status, err := t.GetCoordinatorStatus()
		if err != nil {
			return status, err
		}

		if status.State == coordinator.Done {
			mlog.Info("unbounded load-test has completed")
			return status, nil
		}

		if status.State != coordinator.Running {
			return status, errors.New("coordinator should be running")
		}

		select {
		case <-cancelCh:
			mlog.Info("cancelling load-test")
			if status, err := t.StopCoordinator(); err != nil {
				return status, err
			}
			return coordinator.Status{}, errors.New("canceled")
		case <-time.After(1 * time.Minute):
		}
	}
}

func initLoadTest(t *terraform.Terraform, config *deployment.Config, buildCfg BuildConfig, cancelCh <-chan struct{}) error {
	output, err := t.Output()
	if err != nil {
		return err
	}

	if !output.HasAppServers() {
		return errors.New("no app servers in this deployment")
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}

	agentClient, err := extAgent.NewClient(output.Agents[0].PublicIP)
	if err != nil {
		return fmt.Errorf("error in getting ssh connection %w", err)
	}
	defer agentClient.Close()

	appClients := make([]*ssh.Client, len(output.Instances))
	for i, instance := range output.Instances {
		client, err := extAgent.NewClient(instance.PublicIP)
		if err != nil {
			return fmt.Errorf("error in getting ssh connection %w", err)
		}
		defer client.Close()
		appClients[i] = client
	}

	type cmd struct {
		msg     string
		value   string
		clients []*ssh.Client
	}

	stopCmd := cmd{
		msg:     "Stopping app servers",
		value:   "sudo systemctl stop mattermost",
		clients: appClients,
	}

	buildFileName := filepath.Base(buildCfg.URL)
	installCmd := cmd{
		msg:     "Installing app",
		value:   fmt.Sprintf("tar xzf %s && cp /opt/mattermost/config/config.json . && sudo rm -rf /opt/mattermost && sudo mv mattermost /opt/ && mv config.json /opt/mattermost/config/", buildFileName),
		clients: appClients,
	}

	binaryPath := "/opt/mattermost/bin/mattermost"
	resetCmd := cmd{
		msg:     "Resetting database",
		value:   fmt.Sprintf("%s reset --confirm", binaryPath),
		clients: []*ssh.Client{appClients[0]},
	}

	startCmd := cmd{
		msg:     "Restarting app server",
		value:   fmt.Sprintf("sudo systemctl start mattermost && until $(curl -sSf http://localhost:8065 --output /dev/null); do sleep 1; done;"),
		clients: appClients,
	}

	// do init process
	createAdminCmd := cmd{
		msg: "Creating sysadmin",
		value: fmt.Sprintf("%s user create --email %s --username %s --password '%s' --system_admin || true",
			binaryPath, config.AdminEmail, config.AdminUsername, config.AdminPassword),
		clients: []*ssh.Client{appClients[0]},
	}
	initDataCmd := cmd{
		msg:     "Initializing data",
		value:   fmt.Sprintf("cd mattermost-load-test-ng && ./bin/ltagent init --user-prefix '%s' > /dev/null 2>&1", output.Agents[0].Tags.Name),
		clients: []*ssh.Client{agentClient},
	}

	cmds := []cmd{stopCmd, installCmd, resetCmd, startCmd, createAdminCmd, initDataCmd}
	for _, c := range cmds {
		for _, client := range c.clients {
			select {
			case <-cancelCh:
				mlog.Info("cancelling load-test init")
				return errors.New("canceled")
			default:
			}
			if out, err := client.RunCommand(c.value); err != nil {
				return fmt.Errorf("failed to run cmd %q: %w %s", c.value, err, out)
			}
		}
	}

	return nil
}

func runLoadTest(t *terraform.Terraform, lt LoadTestConfig, cancelCh <-chan struct{}) (coordinator.Status, error) {
	var status coordinator.Status
	coordConfig, err := coordinator.ReadConfig("")
	if err != nil {
		return status, err
	}

	switch lt.Type {
	case LoadTestTypeBounded:
		coordConfig.ClusterConfig.MaxActiveUsers = lt.NumUsers
		// TODO: uncomment this line and remove the loop after a new release is
		// published.
		// coordConfig.MonitorConfig.Queries = nil
		for i := 0; i < len(coordConfig.MonitorConfig.Queries); i++ {
			coordConfig.MonitorConfig.Queries[i].Alert = false
		}
		duration, parseErr := time.ParseDuration(lt.Duration)
		if parseErr != nil {
			return status, parseErr
		}
		return runBoundedLoadTest(t, coordConfig, duration, cancelCh)
	case LoadTestTypeUnbounded:
		// TODO: cleverly set MaxActiveUsers to (numAgents * UsersConfiguration.MaxActiveUsers)
		return runUnboundedLoadTest(t, coordConfig, cancelCh)
	}

	return status, fmt.Errorf("unimplemented LoadTestType %s", lt.Type)
}

func (c *Comparison) getResults(resultsCh <-chan Result) ([]Result, error) {
	var results []Result
	for res := range resultsCh {
		dp := c.deployments[res.deploymentID]
		t := terraform.New(res.deploymentID, &dp.config)
		defer t.Cleanup()
		output, err := t.Output()
		if err != nil {
			return results, err
		}

		promURL := "http://" + output.MetricsServer.PublicIP + ":9090"
		helper, err := prometheus.NewHelper(promURL)
		if err != nil {
			return results, fmt.Errorf("failed to create prometheus.Helper: %w", err)
		}
		g := report.New(res.LoadTests[0].Label, helper, dp.config.Report)
		baseReport, err := g.Generate(res.LoadTests[0].Status.StartTime, res.LoadTests[0].Status.StopTime)
		if err != nil {
			return results, fmt.Errorf("error while generating report: %w", err)
		}
		g = report.New(res.LoadTests[1].Label, helper, dp.config.Report)
		newReport, err := g.Generate(res.LoadTests[1].Status.StartTime, res.LoadTests[1].Status.StopTime)
		if err != nil {
			return results, fmt.Errorf("error while generating report: %w", err)
		}

		if c.config.Output.GenerateReport {
			var buf bytes.Buffer
			// TODO: fix images getting overwritten cause of same name
			err := report.Compare(&buf, c.config.Output.GenerateGraphs, baseReport, newReport)
			if err != nil {
				return results, err
			}
			res.Report = buf.String()
		}

		if c.config.Output.UploadDashboard {
			var dashboardData bytes.Buffer
			title := fmt.Sprintf("Comparison - %d - %s - %s",
				res.LoadTests[0].loadTestID, res.LoadTests[0].Config.DBEngine, res.LoadTests[0].Config.Type)
			if err := report.GenerateDashboard(title, baseReport, newReport, &dashboardData); err != nil {
				return results, err
			}

			url, err := t.UploadDashboard(dashboardData.String())
			if err != nil {
				return results, err
			}
			res.DashboardURL = fmt.Sprintf("http://%s:3000%s", output.MetricsServer.PublicIP, url)
		}

		results = append(results, res)
	}

	return results, nil
}

func provisionBuilds(t *terraform.Terraform, baseBuildURL, newBuildURL string) error {
	output, err := t.Output()
	if err != nil {
		return err
	}
	if !output.HasAppServers() {
		return errors.New("no app servers in this deployment")
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}
	clients := make([]*ssh.Client, len(output.Instances))
	for i, instance := range output.Instances {
		client, err := extAgent.NewClient(instance.PublicIP)
		if err != nil {
			return fmt.Errorf("error in getting ssh connection %w", err)
		}
		defer client.Close()
		clients[i] = client
	}

	for _, client := range clients {
		for _, url := range []string{baseBuildURL, newBuildURL} {
			filePrefix := "file://"
			buildFileName := filepath.Base(url)
			if strings.HasPrefix(url, filePrefix) {
				// upload build file from local filesystem
				buildPath := strings.TrimPrefix(url, filePrefix)
				info, err := os.Stat(buildPath)
				if err != nil {
					return err
				}
				if !info.Mode().IsRegular() {
					return fmt.Errorf("build file %s has to be a regular file", buildPath)
				}
				if out, err := client.UploadFile(buildPath, "/home/ubuntu/"+buildFileName, false); err != nil {
					return fmt.Errorf("error uploading build: %w %s", err, out)
				}
			} else {
				// download build file from URL
				cmd := fmt.Sprintf("wget -O %s %s", buildFileName, url)
				if out, err := client.RunCommand(cmd); err != nil {
					return fmt.Errorf("failed to run cmd %q: %w %s", cmd, err, out)
				}
			}
		}
	}

	return nil
}

// Run performs a fully automated load-test comparison.
// It returns a list of results or an error in case of failure.
func (c *Comparison) Run() ([]Result, error) {
	// create deployments
	err := c.deploymentAction(func(t *terraform.Terraform) error {
		if err := t.Create(false); err != nil {
			return err
		}
		return provisionBuilds(t, c.config.BaseBuild.URL, c.config.NewBuild.URL)
	})
	if err != nil {
		return nil, err
	}

	// run load-tests
	var wg sync.WaitGroup
	wg.Add(len(c.deployments))
	nLoadTests := c.getLoadTestsCount()
	errsCh := make(chan error, nLoadTests)
	resultsCh := make(chan Result, nLoadTests)
	for dpID, dp := range c.deployments {
		go func(dpID string, dp *deploymentInfo) {
			defer wg.Done()
			t := terraform.New(dpID, &dp.config)
			defer t.Cleanup()

			for ltID, lt := range dp.loadTests {
				res := Result{deploymentID: dpID}
				for i, buildCfg := range []BuildConfig{c.config.BaseBuild, c.config.NewBuild} {
					mlog.Debug("initializing load-test")
					// initialize instance state
					if err := initLoadTest(t, &dp.config, buildCfg, c.cancelCh); err != nil {
						errsCh <- err
						return
					}
					mlog.Debug("load-test init done")

					status, err := runLoadTest(t, lt, c.cancelCh)
					if err != nil {
						errsCh <- err
						return
					}
					res.LoadTests[i] = LoadTestResult{loadTestID: ltID,
						Label: buildCfg.Label, Config: lt, Status: status}
				}

				resultsCh <- res
			}
		}(dpID, dp)
	}

	go func() {
		wg.Wait()
		close(resultsCh)
		close(errsCh)
	}()

	if err := <-errsCh; err != nil {
		mlog.Error(err.Error())
		mlog.Debug("an error has occurred, cancelling")
		close(c.cancelCh)
		wg.Wait()
		for err := range errsCh {
			mlog.Error(err.Error())
		}
		return nil, err
	}

	// do actual comparisons and generate some output
	return c.getResults(resultsCh)
}

// Destroy destroys all resources associated with the deployments in the
// current automated load-test comparison.
func (c *Comparison) Destroy() error {
	return c.deploymentAction(func(t *terraform.Terraform) error {
		return t.Destroy()
	})
}
