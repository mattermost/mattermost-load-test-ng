# Comparing load test results

## Prerequisites

Test reports contain two types of data:
- API and store times averaged over the duration of the whole test.
- Granular data of user-specified queries, configured through [the Report subsection in `deployer.json`](config/deployer.md#report).

All those data points are retrieved from Prometheus at the moment of generating the report, but the Prometheus service should have been scraping the metrics *during the test*. Terraform deployments automatically provide a metrics instance with a Prometheus service running, but locally, things may be different:
- If you're running a dev environment locally through the docker-compose file in the `mattermost` repository (i.e., using `make run` or `make run-server`), a Prometheus server should be already up and running at `localhost:9090`.
- If you're running Mattermost locally using any other method, you may need to [manually install and run a Prometheus server](https://prometheus.io/docs/introduction/first_steps/). Make sure to configure it to scrape both your local Mattermost metrics (usually at `localhost:8067`) and [the node exporter](https://prometheus.io/docs/guides/node-exporter/#monitoring-linux-host-metrics-with-the-node-exporter) (usually at `localhost:9100`).

## Generating reports

After a load test is complete, a report can be generated and saved to a file which can be used to compare different load test runs. This can be done using

```sh
go run ./cmd/ltctl report generate --output=base.out --label=base "2020-06-23 07:23:35" "2020-06-23 07:33:35"
```

Note that in the case of a local deployment, you'll need to set the `--prometheus-url ` flag to the URL of the Prometheus server containing your metrics: usually `http://localhost:9090`. If the flag is not set, the tool will assume a Terraform deployment is up and will try to connect to its Prometheus.

The timestamps _must_ be in UTC, and they indicate the range within which the data is to be collected for the test. It is recommended to keep the timestamps within a range during which the loadtest is running a stable number of users, and not in a ramp-up phase or an unstable state of adding/removing users. This is to get consistent results between different runs.

The timestamp ranges for different load tests can be different. They will be compared with the base report. If a report has more data points than the base report, the extra ones will be ignored.

There is no compression of timestamp ranges to normalize them. That is left to Prometheus queries. Data points are just plotted serially on a graph and compared.

## Comparing reports

To compare two load test reports, run:

```sh
go run ./cmd/ltctl report compare base.out new.out --output=results.txt --graph
```

The results.txt will be a Markdown formatted table comparing the average and p99 times of the store and API metrics. Additionally, a `--graph` parameter can also be passed which can be used to generate graphs comparing different metrics like CPU, Memory etc. This also requires the `gnuplot` command to be installed on the system for it to plot graphs.

#### Note

The Markdown output contains an initial section with a sorted summary of worsened/improved calls. This list does automatically exclude calls with (absolute) delta values smaller than 2ms and (absolute) delta percentage values smaller than 1%.

## Best practices while comparing load-tests

- Always use the same cluster setup to compare different tests.
- Calculate the number of users required to reach an average time of at least 10ms. And then compare the tests using that number of users. Anything less than that means the DB is not being stressed well enough and can introduce a lot of noise in the results.
- While starting another test, it is recommended to reset the DB and initialize it again to start with the exact same state as the previous test. This can be done by running the following command:

```sh
go run ./cmd/ltctl loadtest reset
```

