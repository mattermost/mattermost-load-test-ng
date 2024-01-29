// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
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
		"--profile=" + t.config.AWSProfile,
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
		"--profile=" + t.config.AWSProfile,
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

	if output.DBCluster.ClusterIdentifier == "" {
		return "", errors.New("DB cluster identifier not found")
	}

	var buf bytes.Buffer
	args := []string{
		"--profile=" + t.config.AWSProfile,
		"rds",
		"describe-db-clusters",
		"--db-cluster-identifier=" + output.DBCluster.ClusterIdentifier,
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
