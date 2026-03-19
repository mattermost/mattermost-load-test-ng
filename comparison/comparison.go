// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package comparison

import (
	"fmt"
	"sort"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

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
	deploymentInfo DeploymentInfo
	deployments    map[string]*deploymentConfig
}

// New creates and initializes a new Comparison object to be used to run
// automated load-test comparisons. It returns an error in case of failure.
func New(cfg *Config, deployerCfg *deployment.Config) (*Comparison, error) {
	if err := defaults.Validate(cfg); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	cmp := &Comparison{
		config:         cfg,
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

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return output, fmt.Errorf("failed to create ssh agent: %w", err)
	}

	nLoadTests := c.getLoadTestsCount()
	resultsCh := make(chan *Result, nLoadTests)

	// Run tests concurrently
	err = c.deploymentAction(func(t *terraform.Terraform, dpID string, dpConfig *deploymentConfig) error {
		// Create deployments
		if err := t.Create(extAgent, false); err != nil {
			return err
		}
		if err := provisionFiles(t, dpConfig, c.config.BaseBuild, c.config.NewBuild); err != nil {
			return err
		}

		// Run tests for each deployment
		for ltID, lt := range dpConfig.loadTests {
			res := &Result{deploymentID: dpID}
			dumpFilename := lt.getDumpFilename(ltID)
			s3BucketURI := lt.S3BucketDumpURI
			for i, buildCfg := range []BuildConfig{c.config.BaseBuild, c.config.NewBuild} {
				mlog.Debug("initializing load-test")
				// initialize instance state
				if err := initLoadTest(t, buildCfg, dumpFilename, s3BucketURI); err != nil {
					res.LoadTests[i] = LoadTestResult{Failed: true}
					return err
				}
				mlog.Debug("load-test init done")

				status, err := runLoadTest(t, lt)
				if err != nil {
					res.LoadTests[i] = LoadTestResult{Failed: true}
					return err
				}
				res.LoadTests[i] = LoadTestResult{
					loadTestID: ltID,
					Label:      buildCfg.Label,
					Config:     lt,
					Status:     status,
				}
			}

			// For each pair of base/new builds, compare the results and generate the report
			res, err := c.getResults(t, dpConfig, res)
			if err != nil {
				return err
			}
			resultsCh <- res
		}

		return nil
	})
	if err != nil {
		return output, err
	}

	close(resultsCh)

	mlog.Info("load-tests have completed, generating results")

	output.DeploymentInfo = c.deploymentInfo

	for res := range resultsCh {
		output.Results = append(output.Results, *res)
	}

	return output, nil
}

// Destroy destroys all resources associated with the deployments for the
// current automated load-test comparisons.
func (c *Comparison) Destroy(maintainMetrics bool) error {
	if maintainMetrics {
		for _, dp := range c.deployments {
			dp.config.MarkForDestroyAllButMetrics()
		}

		extAgent, err := ssh.NewAgent()
		if err != nil {
			return fmt.Errorf("failed to create ssh agent: %w", err)
		}

		return c.deploymentAction(func(t *terraform.Terraform, _ string, _ *deploymentConfig) error {
			return t.Create(extAgent, false)
		})
	}

	return c.deploymentAction(func(t *terraform.Terraform, _ string, _ *deploymentConfig) error {
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
