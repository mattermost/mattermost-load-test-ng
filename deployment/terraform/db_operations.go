// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// StopDB stops the DB cluster and syncs the changes.
func (t *Terraform) StopDB() error {
	if err := t.preFlightCheck(); err != nil {
		return err
	}

	output, err := t.Output()
	if err != nil {
		return err
	}

	args := []string{
		"rds",
		"stop-db-cluster",
		"--db-cluster-identifier=" + output.DBCluster.ClusterIdentifier,
		"--region=" + t.config.AWSRegion,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := t.runAWSCommand(ctx, args, nil); err != nil {
		return err
	}

	return t.Sync()
}

// StartDB starts the DB cluster and syncs the changes.
func (t *Terraform) StartDB() error {
	if err := t.preFlightCheck(); err != nil {
		return err
	}

	output, err := t.Output()
	if err != nil {
		return err
	}

	args := []string{
		"rds",
		"start-db-cluster",
		"--db-cluster-identifier=" + output.DBCluster.ClusterIdentifier,
		"--region=" + t.config.AWSRegion,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := t.runAWSCommand(ctx, args, nil); err != nil {
		return err
	}

	return t.Sync()
}

type rdsOutput struct {
	DBCluster []struct {
		DatabaseName        string `json:"DatabaseName"`
		DBClusterIdentifier string `json:"DBClusterIdentifier"`
		Status              string `json:"Status"`
		Engine              string `json:"Engine"`
		EngineVersion       string `json:"EngineVersion"`
	} `json:"DBClusters"`
}

// DBStatus returns the status of the DB cluster.
func (t *Terraform) DBStatus() (string, error) {
	if err := t.preFlightCheck(); err != nil {
		return "", err
	}

	output, err := t.Output()
	if err != nil {
		return "", err
	}

	// Neither Terraform DB exists, nor any External DB is used.
	if output.DBCluster.ClusterIdentifier == "" && t.config.ExternalDBSettings.DataSource == "" {
		return "", errors.New("DB cluster identifier not found or no external DB is used.")
	}

	// If an external non-AWS DB is used.
	if t.config.ExternalDBSettings.DataSource != "" && t.config.ExternalDBSettings.ClusterIdentifier == "" {
		return "available", nil
	}

	var identifier string
	if t.config.TerraformDBSettings.ClusterIdentifier != "" {
		identifier = t.config.TerraformDBSettings.ClusterIdentifier
	} else {
		identifier = t.config.ExternalDBSettings.ClusterIdentifier
	}
	var buf bytes.Buffer
	args := []string{
		"rds",
		"describe-db-clusters",
		"--db-cluster-identifier=" + identifier,
		"--region=" + t.config.AWSRegion,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := t.runAWSCommand(ctx, args, &buf); err != nil {
		return "", err
	}

	var out rdsOutput
	err = json.Unmarshal(buf.Bytes(), &out)
	if err != nil {
		return "", err
	}

	if len(out.DBCluster) == 0 {
		return "", fmt.Errorf("No DB Clusters found for cluster identifier: %s", output.DBCluster.ClusterIdentifier)
	}

	return out.DBCluster[0].Status, nil
}

// HasPendingRebootDBParams queries the deployed DB cluster and checks whether
// there is at least a DB instance whose status is "pending-reboot"
func (t *Terraform) HasPendingRebootDBParams() (bool, error) {
	// Build the RDS client
	cfg, err := t.GetAWSConfig()
	if err != nil {
		return false, fmt.Errorf("failed to get AWS config: %w", err)
	}
	rdsClient := rds.NewFromConfig(cfg)

	// Check in parallel whether each DB instance needs to be rebooted
	type retValue struct {
		needsReboot bool
		err         error
	}
	retChan := make(chan retValue, len(t.output.DBCluster.Instances))
	var wg sync.WaitGroup
	for _, instance := range t.output.DBCluster.Instances {
		wg.Add(1)
		go func(dbId string) {
			defer wg.Done()
			needsReboot, err := hasPendingRebootDBParams(rdsClient, dbId)
			retChan <- retValue{needsReboot, err}
		}(instance.DBIdentifier)
	}

	wg.Wait()
	close(retChan)

	needsReboot := false
	var finalErr error
	for b := range retChan {
		needsReboot = needsReboot || b.needsReboot
		finalErr = errors.Join(finalErr, b.err)
	}

	return needsReboot, finalErr
}

// hasPendingRebootDBParams queries the specified DB instance and checks whether
// its status is "pending-reboot"
func hasPendingRebootDBParams(rdsClient *rds.Client, dbId string) (bool, error) {
	describeParams := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: model.NewPointer(dbId),
	}
	describeOut, err := rdsClient.DescribeDBInstances(context.Background(), describeParams)
	if err != nil {
		return false, fmt.Errorf("error describing DB instance %q: %w", dbId, err)
	}

	if len(describeOut.DBInstances) < 1 {
		return false, fmt.Errorf("describe instances returned no instances")
	}

	for _, group := range describeOut.DBInstances[0].DBParameterGroups {
		if group.ParameterApplyStatus == nil {
			return false, fmt.Errorf("parameter group has no ParameterApplyStatus")
		}

		if *group.ParameterApplyStatus == "pending-reboot" {
			return true, nil
		}
	}

	return false, nil
}

// RebootDBInstances reboots all deployed database instances, blocking the call
// until the status of each of them is back to "available"
func (t *Terraform) RebootDBInstances(extAgent *ssh.ExtAgent) error {
	// Build the RDS client
	cfg, err := t.GetAWSConfig()
	if err != nil {
		return fmt.Errorf("failed to get AWS config: %w", err)
	}
	rdsClient := rds.NewFromConfig(cfg)

	// Reboot each DB instance in parallel
	errChan := make(chan error, len(t.output.DBCluster.Instances))
	var wg sync.WaitGroup
	for _, instance := range t.output.DBCluster.Instances {
		wg.Add(1)
		go func(dbId string) {
			defer wg.Done()
			errChan <- rebootDBInstance(rdsClient, dbId)
		}(instance.DBIdentifier)
	}

	wg.Wait()
	close(errChan)

	var finalErr error
	for err := range errChan {
		finalErr = errors.Join(finalErr, err)
	}

	return finalErr
}

// rebootDBInstance reboots the specified database instance, blocking the call
// until its status is back to "available"
func rebootDBInstance(rdsClient *rds.Client, dbId string) error {
	params := &rds.RebootDBInstanceInput{
		DBInstanceIdentifier: model.NewPointer(dbId),
	}

	out, err := rdsClient.RebootDBInstance(context.Background(), params)
	if err != nil {
		return fmt.Errorf("failed to reboot DB instance: %w", err)
	}

	mlog.Info("DB instance reboot has started",
		mlog.String("id", dbId),
		mlog.String("status", *out.DBInstance.DBInstanceStatus))

	// Wait for the DB instance to become available, or fail after 15 minutes
	timeout := time.After(15 * time.Minute)
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout reached, instance is not available yet")
		case <-time.After(30 * time.Second):
			describeParams := &rds.DescribeDBInstancesInput{
				DBInstanceIdentifier: model.NewPointer(dbId),
			}
			describeOut, err := rdsClient.DescribeDBInstances(context.Background(), describeParams)
			if err != nil {
				return fmt.Errorf("error describing DB instance %q: %w", dbId, err)
			}

			if len(describeOut.DBInstances) < 1 {
				return fmt.Errorf("describe instances returned no instances")
			}

			if describeOut.DBInstances[0].DBInstanceStatus == nil {
				return fmt.Errorf("describe instances returned no status")
			}

			status := *describeOut.DBInstances[0].DBInstanceStatus

			// Finish when the DB is completely rebooted
			if status == "available" {
				mlog.Info("DB instance is now available.",
					mlog.String("id", dbId),
					mlog.String("status", status))
				return nil
			}

			mlog.Info("DB instance is not available yet. Waiting 30 seconds...",
				mlog.String("id", dbId),
				mlog.String("status", status))
		}
	}
}
