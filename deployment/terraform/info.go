// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"fmt"
)

// Info displays information about the current load-test deployment.
func (t *Terraform) Info() error {
	output, err := t.Output()
	if err != nil {
		return err
	}

	t.displayInfo(output)

	return nil
}

func (t *Terraform) displayInfo(output *Output) {
	if len(output.Agents.Value) == 0 {
		fmt.Println("No active deployment found.")
		return
	}

	fmt.Println("==================================================")
	fmt.Println("Deployment information:")

	if output.HasAppServers() {
		if output.HasProxy() {
			fmt.Println("Mattermost URL: http://" + output.Proxy.Value[0].PublicDNS)
		} else {
			fmt.Println("Mattermost URL: http://" + output.Instances.Value[0].PublicDNS + ":8065")
		}
		fmt.Println("App Server(s):")
		for _, instance := range output.Instances.Value {
			fmt.Println("- " + instance.Tags.Name + ": " + instance.PublicIP)
		}
	}

	fmt.Println("Load Agent(s):")
	for _, agent := range output.Agents.Value {
		fmt.Println("- " + agent.Tags.Name + ": " + agent.PublicIP)
	}
	fmt.Println("Coordinator: " + output.Agents.Value[0].PublicIP)

	if output.HasMetrics() {
		fmt.Println("Grafana URL: http://" + output.MetricsServer.Value[0].PublicIP + ":3000")
		fmt.Println("Prometheus URL: http://" + output.MetricsServer.Value[0].PublicIP + ":9090")
	}
	if output.HasAppServers() {
		fmt.Println("DB reader endpoint: " + output.DBCluster.Value[0].ReaderEndpoint)
		fmt.Println("DB cluster endpoint: " + output.DBCluster.Value[0].ClusterEndpoint)
	}

	if output.HasElasticSearch() {
		fmt.Println("ElasticSearch cluster endpoint: " + output.ElasticServer.Value[0].Endpoint)
	}
	fmt.Println("==================================================")
}
