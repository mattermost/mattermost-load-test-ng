// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"context"
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
	go func() {
		if t.output.HasS3Bucket() {
			mlog.Info("emptying S3 bucket s3://" + t.output.S3Bucket.Id)
			emptyS3BucketArgs := []string{
				"s3",
				"rm",
				"s3://" + t.output.S3Bucket.Id,
				"--recursive",
			}
			// We intentionally ignore potential errors from this command,
			// since it introduces spurious failures when the bucket is
			// destroyed before the command finishes.
			// See https://mattermost.atlassian.net/browse/MM-62075
			_ = t.runAWSCommand(emptyBucketCtx, emptyS3BucketArgs, nil)
			mlog.Info("emptied S3 bucket s3://" + t.output.S3Bucket.Id)
		}
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
			"rds",
			"delete-db-cluster",
			"--db-cluster-identifier=" + t.config.TerraformDBSettings.ClusterIdentifier,
			"--region=" + t.config.AWSRegion,
			"--skip-final-snapshot",
		}
		// We have to ignore if the cluster was already deleted to make the command idempotent.
		if err := t.runAWSCommand(nil, args, nil); err != nil && !strings.Contains(err.Error(), "DBClusterNotFoundFault") {
			return err
		}
	}

	return t.loadOutput()
}
