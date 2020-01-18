# metricswatcher

`metricswatcher` connects to a Prometheus instance of Mattermost and watches some metrics.

## Configuration

The file `config.metricswatcher.json` contains all the needed configuration, such as Prometheus server URL, interval to update the metrics, logging and Prometheus queries.

## Running

```
metricswatcher --config config/config.metricswatcher.json
```

