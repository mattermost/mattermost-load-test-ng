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
		Value struct {
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
		Value struct {
			PrivateIP  string `json:"private_ip"`
			PublicIP   string `json:"public_ip"`
			PublicDNS  string `json:"public_dns"`
			PrivateDNS string `json:"private_dns"`
		} `json:"value"`
	} `json:"metricsServer"`
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

// IsEmpty returns whether a deployment has some data or not.
// This is useful to check if info is being checked after a cluster is destroyed.
func (o *Output) IsEmpty() bool {
	return len(o.Instances.Value) == 0
}
