# Coordinator Config

## ClusterConfig

*cluster.LoadAgentClusterConfig*

### Agents

*[]agent.LoadAgentConfig*

#### Id
*string*
The unique identifier of the load-agent to be created and run.

#### ApiURL
*string*
The URL to the load-test API server that will run the agent.

### MaxActiveUsers
*int*
The maximum number of concurrently active users to be run across the whole load-agent cluster.

## MonitorConfig

*performance.MonitorConfig*

### PrometheusURL
*string*
The URL to the [Prometheus](https://prometheus.io/docs/introduction/overview/) API server that will collect performance metrics for the target instance.

### UpdateIntervalMs
*int*
The amount of time (in milliseconds) to wait before each query update.

### Queries

*[]prometheus.Query*

#### Description
*string*
The description for the query.

#### Query
*string*
The [PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/) query to be run.

#### Threshold
*float64*
The value over which the performance monitor will fire an alert to the coordinator's feedback loop.

#### Alert
*bool*
The value indicating whether or not to fire an alert.
