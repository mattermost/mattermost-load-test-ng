# Comparing load test results

## Generating reports

After a load test is complete, a report can be generated and saved to a file which can be used to compare different load test runs. This can be done using

```sh
go run ./cmd/ltctl report generate --output=base.out --label=base "2020-06-23 07:23:35" "2020-06-23 07:33:35"
```

The timestamps _must_ be in UTC, and they indicate the range within which the data is to be collected for the test. It is recommended to keep the timestamps within a range during which the loadtest is running a stable number of users, and not in a ramp-up phase or an unstable state of adding/removing users. This is to get consistent results between different runs.

The timestamp ranges for different load tests can be different. They will be compared with the base report. If a report has more data points than the base report, the extra ones will be ignored.

There is no compression of timestamp ranges to normalize them. That is left to Prometheus queries. Data points are just plotted serially on a graph and compared.

## Comparing reports

To compare two load test reports, run:

```sh
go run ./cmd/ltctl report compare base.out new.out --output=results.txt --graph
```

The results.txt will be a markdown formatted table comparing the average and p99 times of the store and API metrics. Additionally, a `--graph` parameter can also be passed which can be used to generate graphs comparing different metrics like CPU, Memory etc. This also requires the `gnuplot` command to be installed on the system for it to plot graphs.

## Best practices while comparing load-tests

- Always use the same cluster setup to compare different tests.
- Calculate the number of users required to reach an average time of at least 10ms. And then compare the tests using that number of users. Anything less than that means the DB is not being stressed well enough and can introduce a lot of noise in the results.
- While starting another test, it is recommended to reset the DB and initialize it again to start with the exact same state as the previous test. Follow these steps to accomplish that.

On the app server's instance, run:
- `/opt/mattermost$ echo YES | ./bin/mattermost reset`
- `/opt/mattermost$ ./bin/mattermost user create --email sysadmin@sample.mattermost.com --username sysadmin --password Sys@dmin-sample1 --system_admin`
- `sudo service mattermost restart`

Then on the coordinator's instance, run:
- `/home/ubuntu/mattermost-load-test-ng$ ./bin/ltagent init`
