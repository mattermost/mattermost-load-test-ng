# Comparing load test results

## Generating reports

After a load test is complete, a report can be generated and saved to a file which can be used to compare different load test runs. This can be done using

```sh
go run ./cmd/ltctl report generate --start-time=1591858735 --end-time=1591858960 --output=base.out --label=base
```

The `start-time` and `end-time` are the timestamps in epoch within which the data is to be collected for the test. It is recommended to keep the timestamps within a range during which the loadtest is running a stable number of users, and not in a ramp-up phase or an unstable state of adding/removing users. This is to get consistent results between different runs.

The timestamp ranges for different load tests can be different. They will be compared with the base report. If a report has more data points than the base report, the extra ones will be ignored.

There is no compression of timestamp ranges to normalize them. That is left to Prometheus queries. Data points are just plotted serially on a graph and compared.

## Comparing reports

To compare two load test reports, run:

```sh
go run ./cmd/ltctl report compare base.out new.out --output=results.txt --graph
```

The results.txt will be a markdown formatted table comparing the average and p99 times of the store and API metrics. Additionally, a `--graph` parameter can also be passed which can be used to generate graphs comparing different metrics like CPU, Memory etc. This also requires the `gnuplot` command to be installed on the system for it to plot graphs.