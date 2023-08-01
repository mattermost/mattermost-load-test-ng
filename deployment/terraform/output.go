// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

type output struct {
	Proxy struct {
		Value []Instance `json:"value"`
	} `json:"proxy"`
	Instances struct {
		Value []Instance `json:"value"`
	} `json:"instances"`
	DBCluster struct {
		Value []struct {
			Endpoint string `json:"endpoint"`
		} `json:"value"`
	} `json:"dbCluster"`
	Agents struct {
		Value []Instance `json:"value"`
	} `json:"agents"`
	MetricsServer struct {
		Value []Instance `json:"value"`
	} `json:"metricsServer"`
	ElasticServer struct {
		Value []ElasticSearchDomain `json:"value"`
	} `json:"elasticServer"`
	JobServers struct {
		Value []Instance `json:"value"`
	} `json:"jobServers"`
	S3Bucket struct {
		Value []S3Bucket `json:"value"`
	} `json:"s3Bucket"`
	S3Key struct {
		Value []IAMAccess `json:"value"`
	} `json:"s3Key"`
}

// Output contains the output variables which are
// created after a deployment.
type Output struct {
	ClusterName         string
	Proxy               Instance            `json:"proxy"`
	Instances           []Instance          `json:"instances"`
	DBCluster           DBCluster           `json:"dbCluster"`
	Agents              []Instance          `json:"agents"`
	MetricsServer       Instance            `json:"metricsServer"`
	ElasticSearchServer ElasticSearchDomain `json:"elasticServer"`
	JobServers          []Instance          `json:"jobServers"`
	S3Bucket            S3Bucket            `json:"s3Bucket"`
	S3Key               IAMAccess           `json:"s3Key"`
}

// Instance is an AWS EC2 instance resource.
type Instance struct {
	PrivateIP  string `json:"private_ip"`
	PublicIP   string `json:"public_ip"`
	PublicDNS  string `json:"public_dns"`
	PrivateDNS string `json:"private_dns"`
	Tags       Tags   `json:"tags"`
}

// ElasticSearchDomain is an AWS Elasticsearch domain.
type ElasticSearchDomain struct {
	Endpoint string `json:"endpoint"`
	Tags     Tags   `json:"tags"`
}

// Tags are the values attached to resource.
type Tags struct {
	Name string `json:"Name"`
}

// DBCluster defines a RDS cluster instance resource.
type DBCluster struct {
	Endpoints []string `json:"endpoint"`
}

// IAMAccess is a set of credentials that allow API requests to be made as an IAM user.
type IAMAccess struct {
	Id     string `json:"id"`
	Secret string `json:"secret"`
}

// S3Bucket defines a specific S3 bucket.
type S3Bucket struct {
	Id     string `json:"id"`
	Region string `json:"region"`
}

func (t *Terraform) loadOutput() error {
	var buf bytes.Buffer

	if err := t.runCommand(&buf, "output", "-json", "-state="+t.getStatePath()); err != nil {
		return err
	}
	var o output
	if err := json.Unmarshal(buf.Bytes(), &o); err != nil {
		return err
	}

	var clusterName string
	if t.config != nil {
		clusterName = t.config.ClusterName
	}
	outputv2 := &Output{
		ClusterName: clusterName,
		Instances:   o.Instances.Value,
		Agents:      o.Agents.Value,
		JobServers:  o.JobServers.Value,
	}

	if len(o.Proxy.Value) > 0 {
		outputv2.Proxy = o.Proxy.Value[0]
	}
	if len(o.DBCluster.Value) > 0 {
		for _, ep := range o.DBCluster.Value {
			outputv2.DBCluster.Endpoints = append(outputv2.DBCluster.Endpoints, ep.Endpoint)
		}
	}
	if len(o.MetricsServer.Value) > 0 {
		outputv2.MetricsServer = o.MetricsServer.Value[0]
	}
	if len(o.ElasticServer.Value) > 0 {
		outputv2.ElasticSearchServer = o.ElasticServer.Value[0]
	}
	if len(o.S3Bucket.Value) > 0 {
		outputv2.S3Bucket = o.S3Bucket.Value[0]
	}
	if len(o.S3Key.Value) > 0 {
		outputv2.S3Key = o.S3Key.Value[0]
	}

	t.output = outputv2

	return nil
}

func (t *Terraform) setOutput() error {
	if t.output == nil {
		return t.loadOutput()
	}
	return nil
}

// Output reads the current terraform output and caches it internally for future use.
// The output is guaranteed to be up to date after calls to Create and Destroy.
func (t *Terraform) Output() (*Output, error) {
	if err := t.setOutput(); err != nil {
		return nil, err
	}
	return t.output, nil
}

// HasProxy returns whether a deployment has proxy installed in it or not.
func (o *Output) HasProxy() bool {
	return o.Proxy.PrivateIP != ""
}

// HasDB returns whether a deployment has database installed in it or not.
func (o *Output) HasDB() bool {
	return len(o.DBCluster.Endpoints) > 0
}

// HasElasticSearch returns whether a deployment has ElasticSaearch installed in it or not.
func (o *Output) HasElasticSearch() bool {
	return o.ElasticSearchServer.Endpoint != ""
}

// HasAppServers returns whether a deployment includes app server instances.
func (o *Output) HasAppServers() bool {
	return len(o.Instances) > 0
}

// HasAgents returns whether a deployment includes agent instances.
func (o *Output) HasAgents() bool {
	return len(o.Agents) > 0
}

// HasMetrics returns whether a deployment includes the metrics instance.
func (o *Output) HasMetrics() bool {
	return o.MetricsServer.PrivateIP != ""
}

// HasS3Bucket returns whether a deployment includes the S3 Bucket.
func (o *Output) HasS3Bucket() bool {
	return o.S3Bucket.Region != ""
}

// HasS3Key returns whether a deployment includes the S3 Key.
func (o *Output) HasS3Key() bool {
	return o.S3Key.Secret != ""
}

// HasJobServer returns whether a deployment has a dedicated job server.
func (o *Output) HasJobServer() bool {
	return len(o.JobServers) > 0
}

// DBReaders returns the list of db reader endpoints.
func (o *Output) DBReaders() []string {
	var rds []string
	prefix := o.ClusterName + "-rd"
	for _, ep := range o.DBCluster.Endpoints {
		if strings.HasPrefix(ep, prefix) {
			rds = append(rds, ep)
		}
	}
	return rds
}

// DBWriter returns the db writer endpoint.
func (o *Output) DBWriter() string {
	var wr string
	prefix := o.ClusterName + "-wr"
	for _, ep := range o.DBCluster.Endpoints {
		if strings.HasPrefix(ep, prefix) {
			wr = ep
			break
		}
	}
	return wr
}

// PermalinksIPsSubstCommand returns the substitution command to replace
// permalinks in the DB dump for the current deployment.
func (o *Output) PermalinksIPsSubstCommand(permalinkIPsToReplace []string) string {
	if len(permalinkIPsToReplace) == 0 {
		return ""
	}

	if o.HasProxy() {
		return deployment.GenCmdForPermalinksIPsSubstitution(o.Proxy.PublicIP, permalinkIPsToReplace, true)
	}

	if len(o.Instances) > 0 {
		mlog.Warn("detected multiple app instances with no proxy, using only first instance's IP for permalink substitution")
	}

	return deployment.GenCmdForPermalinksIPsSubstitution(o.Instances[0].PublicIP, permalinkIPsToReplace, false)
}
