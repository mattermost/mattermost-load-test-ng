# metricswatcher

`metricswatcher` connects to a Prometheus instance of Mattermost and watches some metrics.

## Configuration

Check the `PrometheusConfiguration` section of your config file:

```json
    "PrometheusConfiguration": {
        "PrometheusURL": "http://localhost:9090",
        "UpdateIntervalInMS": 1000
    }
```

## Running

```
metricswatcher --config config/config.default.json --queries config/prometheusqueries.sample.json
```

