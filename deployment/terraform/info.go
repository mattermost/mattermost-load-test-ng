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
	fmt.Println("==================================================")
	fmt.Println("Deployment information:")
	if output.HasProxy() {
		fmt.Println("Mattermost URL: http://" + output.Proxy.Value[0].PublicDNS)
	} else {
		fmt.Println("Mattermost URL: http://" + output.Instances.Value[0].PublicDNS + ":8065")
	}
	fmt.Println("App Server(s):")
	for _, instance := range output.Instances.Value {
		fmt.Println("- " + instance.Tags.Name + ": " + instance.PublicIP)
	}

	fmt.Println("Load Agent(s):")
	for _, agent := range output.Agents.Value {
		fmt.Println("- " + agent.Tags.Name + ": " + agent.PublicIP)
	}
	if len(output.Agents.Value) > 0 {
		fmt.Println("Coordinator: " + output.Agents.Value[0].PublicIP)
	}
	fmt.Println("Grafana URL: http://" + output.MetricsServer.Value.PublicIP + ":3000")
	fmt.Println("Prometheus URL: http://" + output.MetricsServer.Value.PublicIP + ":9090")
	fmt.Println("DB reader endpoint: " + output.DBCluster.Value.ReaderEndpoint)
	fmt.Println("DB cluster endpoint: " + output.DBCluster.Value.ClusterEndpoint)
	fmt.Println("==================================================")
}
