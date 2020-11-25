## Commonly tested setups

### Single App cluster

| Name    | Type        | Specs                | Amount |
|---------|-------------|----------------------|--------|
| App     | c5.xlarge   | 4vCPU - 8GB RAM      | 1      |
| DB      | db.r4.large | 2vCPU - 15.25GB RAM  | 1      |
| Agent   | t3.xlarge   | 4vCPU - 16GB RAM     | 2      |
| Metrics | t3.xlarge   | 4vCPU - 16GB RAM     | 1      |

### Multi App cluster

| Name    | Type        | Specs                | Amount                  |
|---------|-------------|----------------------|-------------------------|
| App     | c5.xlarge   | 4vCPU - 8GB RAM      | 2                       |
| DB      | db.r4.large | 2vCPU - 15.25GB RAM  | 2 (1 writer + 1 reader) |
| Proxy   | m4.xlarge   | 4vCPU - 16GB RAM     | 1                       |
| Agent   | t3.xlarge   | 4vCPU - 16GB RAM     | 4                       |
| Metrics | t3.xlarge   | 4vCPU - 16GB RAM     | 1                       |


### Results

| Cluster Type | Database Type/Version     | Supported users  |
|--------------|---------------------------|------------------|
| Single       | MySQL (aurora 5.7)        | ~2000            |
| Single       | PostgreSQL (aurora 9.6.8) | ~2000            |
| Multi        | MySQL (aurora 5.7)        | ~3000            |
| Multi        | PostgreSQL (aurora 9.6.8) | ~3000            |

### Notes

Supported users values are calculated using default config settings which in turn derive from real data collected on our community servers. They are an estimation of the maximum amount of concurrently active users the target instance comfortably supports. They are not to be intended as the maxium number of registered users an instance could have.

`c5.xlarge` were specifically chosen to host Mattermost server as they provide the most stable results between different deployments (fluctuation in performance is minimized).

`t3.xlarge` were chosen and tested to simulate a maximum of `2000` users on default settings (simulative controller with default rates). The average amount of memory for each user has been measured at around ~4MB.

By default, the first load-test agent instance will be also be running the [`coordinator`](coordinator.md). This is acceptable as its overhead is almost negligible compared to the agent itself.

Metrics instance hosts both Prometheus and Grafana. It also hosts the Inbucket service.
