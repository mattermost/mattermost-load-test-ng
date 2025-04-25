// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"fmt"
	"net"
	"strconv"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
)

// Info displays information about the current load-test deployment.
func (t *Terraform) Info() error {
	output, err := t.Output()
	if err != nil {
		return err
	}

	displayInfo(t.config, output)

	return nil
}

func displayInfo(cfg *deployment.Config, output *Output) {
	fmt.Println("==================================================")
	fmt.Println("Deployment information:")

	if output.HasAppServers() {
		if output.HasProxy() {
			fmt.Println("Mattermost URL: http://" + output.Proxies[0].GetConnectionDNS())
		} else {
			fmt.Println("Mattermost URL: http://" + output.Instances[0].GetConnectionDNS() + ":8065")
		}
		fmt.Println("App Server(s):")
		for _, instance := range output.Instances {
			fmt.Println("- " + instance.Tags.Name + ": " + instance.GetConnectionIP())
		}
	}

	if output.HasJobServer() {
		fmt.Println("Job Server(s):")
		for _, instance := range output.JobServers {
			fmt.Println("- " + instance.Tags.Name + ": " + instance.GetConnectionIP())
		}
	}

	if output.HasAgents() {
		fmt.Println("Load Agent(s):")
		for _, agent := range output.Agents {
			fmt.Println("- " + agent.Tags.Name + ": " + agent.GetConnectionIP())
		}

		fmt.Println("Coordinator: " + output.Agents[0].GetConnectionIP())
	}

	if output.HasProxy() {
		if len(output.Proxies) > 1 {
			fmt.Println("Proxies:")
		} else {
			fmt.Println("Proxy:")
		}
		for _, inst := range output.Proxies {
			fmt.Println("- " + inst.Tags.Name + ": " + inst.GetConnectionIP())
		}
	}

	if output.HasMetrics() {
		fmt.Println("Grafana URL: http://" + output.MetricsServer.GetConnectionIP() + ":3000")
		fmt.Println("    Credentials: admin / " + cfg.GeneratedValues.GrafanaAdminPassword)
		fmt.Println("Prometheus URL: http://" + output.MetricsServer.GetConnectionIP() + ":9090")
		fmt.Println("Pyroscope URL: http://" + output.MetricsServer.GetConnectionIP() + ":4040")
	}
	if output.HasKeycloak() {
		fmt.Println("Keycloak server IP: " + output.KeycloakServer.GetConnectionIP())
		fmt.Println("Keycloak URL: http://" + output.KeycloakServer.GetConnectionDNS() + ":8080/")
		if len(output.KeycloakDatabaseCluster.Instances) > 0 {
			fmt.Printf("Keycloak DB Cluster: %v\n", output.KeycloakDatabaseCluster.Instances[0].Endpoint)
		}
	}
	if output.HasDB() {
		fmt.Println("DB Cluster Identifier: ", output.DBCluster.ClusterIdentifier)
		fmt.Println("DB writer endpoint: " + output.DBWriter())
		for _, rd := range output.DBReaders() {
			fmt.Println("DB reader endpoint: " + rd)
		}
	}

	if output.HasElasticSearch() {
		fmt.Println("ElasticSearch cluster endpoint: " + output.ElasticSearchServer.Endpoint)
	}

	if output.HasRedis() {
		fmt.Println("Redis endpoint: ", net.JoinHostPort(output.RedisServer.Address, strconv.Itoa(output.RedisServer.Port)))
	}
	fmt.Println("==================================================")
}
