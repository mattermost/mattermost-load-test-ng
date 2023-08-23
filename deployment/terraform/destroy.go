// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// Destroy destroys the created load-test environment.
func (t *Terraform) Destroy() error {
	if err := t.preFlightCheck(); err != nil {
		return err
	}

	// We need to load the output to check whether the deployment has an S3
	// bucket to destroy
	if err := t.loadOutput(); err != nil {
		return err
	}

	// Empty the S3 bucket concurrently with the main terraform destroy command
	// to ensure that it is properly destroyed
	// See https://mattermost.atlassian.net/browse/MM-47263
	emptyBucketCtx, emptyBucketCancel := context.WithCancel(context.Background())
	defer emptyBucketCancel()
	emptyBucketErrCh := make(chan error, 1)
	go func() {
		if t.output.HasS3Bucket() {
			mlog.Info("emptying S3 bucket s3://" + t.output.S3Bucket.Id)
			emptyS3BucketArgs := []string{"--profile", t.Config().AWSProfile,
				"s3",
				"rm",
				"s3://" + t.output.S3Bucket.Id,
				"--recursive",
			}
			if err := t.runAWSCommand(emptyBucketCtx, emptyS3BucketArgs); err != nil {
				emptyBucketErrCh <- fmt.Errorf("failed to run local cmd \"aws %s\": %w", strings.Join(emptyS3BucketArgs, " "), err)
				return
			}
			mlog.Info("emptied S3 bucket s3://" + t.output.S3Bucket.Id)
		}
		emptyBucketErrCh <- nil
	}()

	var params []string
	params = append(params, "destroy")
	params = append(params, t.getParams()...)
	params = append(params, "-auto-approve",
		"-input=false",
		"-state="+t.getStatePath())

	if err := t.runCommand(nil, params...); err != nil {
		return err
	}

	// If we have restored from a DB backup, we need to manually delete the cluster.
	if t.config.TerraformDBSettings.ClusterIdentifier != "" {
		args := []string{
			"--profile=" + t.config.AWSProfile,
			"rds",
			"delete-db-cluster",
			"--db-cluster-identifier=" + t.config.TerraformDBSettings.ClusterIdentifier,
			"--region=" + t.config.AWSRegion,
			"--skip-final-snapshot",
		}
		// We have to ignore if the cluster was already deleted to make the command idempotent.
		if err := t.runAWSCommand(nil, args); err != nil && !strings.Contains(err.Error(), "DBClusterNotFoundFault") {
			return err
		}
	}

	// Make sure that the empty bucket command has finished and check for any
	// possible errors. The check may be redundant, since if we're already
	// here, it means that the terraform destroy has finished successfullly, so
	// the S3 command should have finished as well. Better safe than sorry, though.
	err := <-emptyBucketErrCh
	if err != nil {
		return fmt.Errorf("failed to empty s3://%s: %w", t.output.S3Bucket.Id, err)
	}

	return t.loadOutput()
}
