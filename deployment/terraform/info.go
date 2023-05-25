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

	displayInfo(output)

	return nil
}

func displayInfo(output *Output) {
	if len(output.Agents) == 0 {
		fmt.Println("No active deployment found.")
		return
	}

	fmt.Println("==================================================")
	fmt.Println("Deployment information:")

	if output.HasAppServers() {
		if output.HasProxy() {
			fmt.Println("Mattermost URL: http://" + output.Proxy.PublicDNS)
		} else {
			fmt.Println("Mattermost URL: http://" + output.Instances[0].PublicDNS + ":8065")
		}
		fmt.Println("App Server(s):")
		for _, instance := range output.Instances {
			fmt.Println("- " + instance.Tags.Name + ": " + instance.PublicIP)
		}
	}

	if output.HasJobServer() {
		fmt.Println("Job Server(s):")
		for _, instance := range output.JobServers {
			fmt.Println("- " + instance.Tags.Name + ": " + instance.PublicIP)
		}
	}

	fmt.Println("Load Agent(s):")
	for _, agent := range output.Agents {
		fmt.Println("- " + agent.Tags.Name + ": " + agent.PublicIP)
	}
	fmt.Println("Coordinator: " + output.Agents[0].PublicIP)

	if output.HasMetrics() {
		fmt.Println("Grafana URL: http://" + output.MetricsServer.PublicIP + ":3000")
		fmt.Println("Prometheus URL: http://" + output.MetricsServer.PublicIP + ":9090")
		fmt.Println("Pyroscope URL: http://" + output.MetricsServer.PublicIP + ":4040")
	}
	if output.HasDB() {
		fmt.Println("DB writer endpoint: " + output.DBWriter())
		for _, rd := range output.DBReaders() {
			fmt.Println("DB reader endpoint: " + rd)
		}
	}

	if output.HasElasticSearch() {
		fmt.Println("ElasticSearch cluster endpoint: " + output.ElasticSearchServer.Endpoint)
	}
	fmt.Println("==================================================")
}
