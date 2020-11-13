// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package comparison

import (
	"fmt"
	"sync"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"

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

// New creates and initializes a new Comparison object to be used to run
// automated load-test comparisons. It returns an error in case of failure.
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

// Run performs fully automated load-test comparisons.
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
		mlog.Error("an error has occurred, cancelling", mlog.Err(err))
		close(c.cancelCh)
		wg.Wait()
		return nil, err
	}

	mlog.Info("load-tests have completed, going to generate some output")
	// do actual comparisons and generate some output
	return c.getResults(resultsCh)
}

// Destroy destroys all resources associated with the deployments for the
// current automated load-test comparisons.
func (c *Comparison) Destroy() error {
	return c.deploymentAction(func(t *terraform.Terraform) error {
		return t.Destroy()
	})
}
