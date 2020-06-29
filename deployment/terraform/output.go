// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

// Output contains the output variables which are
// created after a deployment.
type Output struct {
	Proxy struct {
		Value []struct {
			PrivateIP  string `json:"private_ip"`
			PublicIP   string `json:"public_ip"`
			PublicDNS  string `json:"public_dns"`
			PrivateDNS string `json:"private_dns"`
		} `json:"value"`
	} `json:"proxy"`
	Instances struct {
		Value []struct {
			PrivateIP  string `json:"private_ip"`
			PublicIP   string `json:"public_ip"`
			PublicDNS  string `json:"public_dns"`
			PrivateDNS string `json:"private_dns"`
			Tags       struct {
				Name string `json:"Name"`
			} `json:"tags"`
		} `json:"value"`
	} `json:"instances"`
	DBCluster struct {
		Value []struct {
			ClusterEndpoint string `json:"endpoint"`
			ReaderEndpoint  string `json:"reader_endpoint"`
		} `json:"value"`
	} `json:"dbCluster"`
	Agents struct {
		Value []struct {
			PrivateIP  string `json:"private_ip"`
			PublicIP   string `json:"public_ip"`
			PublicDNS  string `json:"public_dns"`
			PrivateDNS string `json:"private_dns"`
			Tags       struct {
				Name string `json:"Name"`
			} `json:"tags"`
		} `json:"value"`
	} `json:"agents"`
	MetricsServer struct {
		Value []struct {
			PrivateIP  string `json:"private_ip"`
			PublicIP   string `json:"public_ip"`
			PublicDNS  string `json:"public_dns"`
			PrivateDNS string `json:"private_dns"`
		} `json:"value"`
	} `json:"metricsServer"`
	ElasticServer struct {
		Value []struct {
			Endpoint string `json:"endpoint"`
			Tags     struct {
				Name string `json:"Name"`
			} `json:"tags"`
		} `json:"value"`
	} `json:"elasticServer"`
	S3Bucket struct {
		Value []struct {
			Id     string `json:"id"`
			Region string `json:"region"`
		}
	} `json:"s3Bucket"`
	S3Key struct {
		Value []struct {
			Id     string `json:"id"`
			Secret string `json:"secret"`
		}
	} `json:"s3Key"`
}

// HasProxy returns whether a deployment has proxy installed in it or not.
func (o *Output) HasProxy() bool {
	return len(o.Proxy.Value) > 0
}

//HasElasticSearch returns whether a deployment has ElasticSaearch installed in it or not.
func (o *Output) HasElasticSearch() bool {
	return len(o.ElasticServer.Value) > 0
}

// HasMetrics returns whether a deployment includes app server instances.
func (o *Output) HasAppServers() bool {
	return len(o.Instances.Value) > 0
}

// HasMetrics returns whether a deployment includes the metrics instance.
func (o *Output) HasMetrics() bool {
	return len(o.MetricsServer.Value) > 0
}
