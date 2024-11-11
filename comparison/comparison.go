// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package comparison

import (
	"fmt"
	"sort"
	"sync"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

type deploymentConfig struct {
	config    deployment.Config
	loadTests []LoadTestConfig
}

// Comparison holds the state needed to perform automated
// load-test comparisons.
type Comparison struct {
	config         *Config
	coordConfig    *coordinator.Config
	ltConfig       *loadtest.Config
	deploymentInfo DeploymentInfo
	deployments    map[string]*deploymentConfig
}

// New creates and initializes a new Comparison object to be used to run
// automated load-test comparisons. It returns an error in case of failure.
func New(cfg *Config, deployerCfg *deployment.Config, coordConfig *coordinator.Config, ltConfig *loadtest.Config) (*Comparison, error) {
	if err := defaults.Validate(cfg); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	cmp := &Comparison{
		config:         cfg,
		coordConfig:    coordConfig,
		ltConfig:       ltConfig,
		deploymentInfo: getDeploymentInfo(deployerCfg),
		deployments:    map[string]*deploymentConfig{},
	}

	var i int
	deployments := map[string]string{}
	for _, lt := range cfg.LoadTests {
		key := string(lt.Type) + string(lt.DBEngine)
		var dp *deploymentConfig
		if id, ok := deployments[key]; !ok {
			dp = &deploymentConfig{config: *deployerCfg}
			dp.config.TerraformDBSettings.InstanceEngine = "aurora-" + string(lt.DBEngine)
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
func (c *Comparison) Run() (Output, error) {
	var output Output
	// create deployments
	err := c.deploymentAction(func(t *terraform.Terraform, dpConfig *deploymentConfig) error {
		if err := t.Create(false, c.ltConfig); err != nil {
			return err
		}
		return provisionFiles(t, dpConfig, c.config.BaseBuild.URL, c.config.NewBuild.URL)
	})
	if err != nil {
		return output, err
	}

	// run load-tests
	var wg sync.WaitGroup
	wg.Add(len(c.deployments))
	nLoadTests := c.getLoadTestsCount()
	errsCh := make(chan error, nLoadTests)
	resultsCh := make(chan Result, nLoadTests)
	for dpID, dp := range c.deployments {
		go func(dpID string, dp *deploymentConfig) {
			defer wg.Done()
			t, err := terraform.New(dpID, dp.config)
			if err != nil {
				errsCh <- fmt.Errorf("failed to create terraform engine: %w", err)
				return
			}

			for ltID, lt := range dp.loadTests {
				res := Result{deploymentID: dpID}
				dumpFilename := lt.getDumpFilename(ltID)
				s3BucketURI := lt.S3BucketDumpURI
				for i, buildCfg := range []BuildConfig{c.config.BaseBuild, c.config.NewBuild} {
					mlog.Debug("initializing load-test")
					// initialize instance state
					if err := initLoadTest(t, buildCfg, dumpFilename, s3BucketURI); err != nil {
						res.LoadTests[i] = LoadTestResult{Failed: true}
						errsCh <- err
						break
					}
					mlog.Debug("load-test init done")

					status, err := runLoadTest(t, lt, c.coordConfig, c.ltConfig)
					if err != nil {
						res.LoadTests[i] = LoadTestResult{Failed: true}
						errsCh <- err
						break
					}
					res.LoadTests[i] = LoadTestResult{
						loadTestID: ltID,
						Label:      buildCfg.Label,
						Config:     lt,
						Status:     status,
					}
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

	for err := range errsCh {
		if err != nil {
			mlog.Error("an error has occurred", mlog.Err(err))
		}
	}

	mlog.Info("load-tests have completed, generating results")

	output.DeploymentInfo = c.deploymentInfo
	// do actual comparisons and generate some results
	output.Results = c.getResults(resultsCh)

	return output, nil
}

// Destroy destroys all resources associated with the deployments for the
// current automated load-test comparisons.
func (c *Comparison) Destroy() error {
	return c.deploymentAction(func(t *terraform.Terraform, _ *deploymentConfig) error {
		if err := t.Sync(); err != nil {
			return err
		}
		return t.Destroy()
	})
}

func (c *Comparison) GetDeploymentIds() []string {
	ids := []string{}
	for k := range c.deployments {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	return ids
}
