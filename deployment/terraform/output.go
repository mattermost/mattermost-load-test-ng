// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"encoding/json"
)

type output struct {
	Proxy struct {
		Value []Instance `json:"value"`
	} `json:"proxy"`
	Instances struct {
		Value []Instance `json:"value"`
	} `json:"instances"`
	DBCluster struct {
		Value []DBCluster `json:"value"`
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
	Proxy               Instance            `json:"proxy"`
	Instances           []Instance          `json:"instances"`
	DBCluster           DBCluster           `json:"dbCluster"`
	Agents              []Instance          `json:"agents"`
	MetricsServer       Instance            `json:"metricsServer"`
	ElasticSearchServer ElasticSearchDomain `json:"elasticServer"`
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
	ClusterEndpoint string `json:"endpoint"`
	ReaderEndpoint  string `json:"reader_endpoint"`
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

// Output reads the current terraform output
func (t *Terraform) Output() (*Output, error) {
	var buf bytes.Buffer

	if err := t.runCommand(&buf, "output", "-json", "-state="+t.getStatePath()); err != nil {
		return nil, err
	}
	var o output
	if err := json.Unmarshal(buf.Bytes(), &o); err != nil {
		return nil, err
	}

	outputv2 := &Output{
		Instances: o.Instances.Value,
		Agents:    o.Agents.Value,
	}

	if len(o.Proxy.Value) > 0 {
		outputv2.Proxy = o.Proxy.Value[0]
	}
	if len(o.DBCluster.Value) > 0 {
		outputv2.DBCluster = o.DBCluster.Value[0]
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
	return outputv2, nil
}

// HasProxy returns whether a deployment has proxy installed in it or not.
func (o *Output) HasProxy() bool {
	return o.Proxy.PrivateIP != ""
}

// HasDB returns whether a deployment has database installed in it or not.
func (o *Output) HasDB() bool {
	return o.DBCluster.ClusterEndpoint != ""
}

//HasElasticSearch returns whether a deployment has ElasticSaearch installed in it or not.
func (o *Output) HasElasticSearch() bool {
	return o.ElasticSearchServer.Endpoint != ""
}

// HasAppServers returns whether a deployment includes app server instances.
func (o *Output) HasAppServers() bool {
	return len(o.Instances) > 0
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
