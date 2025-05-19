# Coordinator Config

## ClusterConfig

*cluster.LoadAgentClusterConfig*

### Agents

*[]cluster.LoadAgentConfig*

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

The delay (in milliseconds) between each query update.
This value also indirectly controls how often new users are added during the ramp-up phase, assuming there is no performance degradation (i.e., no query has exceeded its threshold).
If performance degradation is detected, `coordinator.Config.RestTimeSec` determines the rate at which users are added or removed.

**Note**: This value cannot exceed `coordinator.Config.RestTimeSec * 1000`.

### Queries

*[]prometheus.Query*

#### Description

*string*

The description for the query.

#### Legend

*string*

The legend shown in this query's panel in the Grafana dashboard that is generated with all enabled queries. If this string is empty, Grafana creates an automatic legend for the panel.

#### Query

*string*

The [PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/) query to be run.

#### Threshold

*float64*

The value over which the performance monitor will fire an alert to the coordinator's feedback loop.

#### Alert

*bool*

The value indicating whether or not to fire an alert.

## NumUsersInc

*int*

The number of active users to increment at each iteration of the feedback loop.  
It should be proportional to the maximum number of users expected to test.

## NumUsersDec

*int*

The number of active users to decrement at each iteration of the feedback loop.  
It should be proportional to the maximum number of users expected to test.

## RestTimeSec

*int*

The number of seconds to wait after a performance degradation event before starting to increment or decrement users again.

**Note**: The actual time waited before an increment or decrement action can be up to (`RestTimeSec + MonitorConfig.UpdateIntervalMs/1000`) seconds.

## LogSettings

### EnableConsole

*bool*

If true, the server outputs log messages to the console based on ConsoleLevel option.

### ConsoleLevel

*string*

Level of detail at which log events are written to the console.

### ConsoleJson

*bool*

When true, logged events are written in a machine readable JSON format. Otherwise they are printed as plain text.

### EnableFile

*bool*

When true, logged events are written to the file specified by the `FileLocation` setting.

### FileLevel

*string*

Level of detail at which log events are written to log files.

### FileJson

*bool*

When true, logged events are written in a machine readable JSON format. Otherwise they are printed as plain text.

### FileLocation

*string*

The location of the log file.

### EnableColor

*bool*

When true enables colored output.
