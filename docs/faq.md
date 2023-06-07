## FAQ

### What login methods does the load-test agent support?

The load-test agent currently supports only the email/password authentication method.

### What is a bounded load-test?

We define a load-test to be bounded when the number of simulated users is fixed.
This is particularly useful when running performance comparisons between two clusters/builds.

### What is an unbounded load-test

We define a load-test to be unbounded when the number of simulated users can vary up to a pre-configured limit.
This type of load-test is used to determine the capacity of a system and will output an estimated number of users.
The rule of thumb is that when starting an unbounded load-test we should always shoot for more users than what we think an installation can support.

### How do I configure a test to be bounded or unbounded?

The nature of the test is controlled by how the [`coordinator`](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/coordinator.md) controls the [feedback loop](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/coordinator.md#the-feedback-loop). If the coordinator is configured to decrease the users when some metrics surpass a threshold (e.g. the P99 latency in the server is over 2 seconds), then the test will be unbounded. If the coordinator does not monitor these metrics and just let all users connect freely, then the test will be bounded.

To configure this, you need to take a look at the `MonitorConfig.Queries` configuration in the deployer:
- If the array of queries is empty, or all of them are disabled (by setting `Alert` to `false`), the test is **bounded**.
- If there is at least one query that is enabled, the test is **unbounded**.

Note that in both cases, `ClusterConfig.MaxActiveUsers` should be set to `AgentInstanceCount * UsersConfiguration.MaxActiveUsers`.

### Can I use a pre-existing Mattermost or database deployment?

Yes, you can use an existing Mattermost deployment, or just the database portion.

Note: You should **not** utilize an existing production setup to loadtest against because the loadtest agent will create users, posts, teams and channels and utilize most of your server resources. Best practice is to clone your production setup and loadtest against that.

### Can I export a Grafana dashboard for future reference?

Yes, you can do so by using the feature to [publish a snapshot to Raintank](https://grafana.com/docs/grafana/latest/dashboards/share-dashboards-panels/#publish-a-snapshot). An example of what such a snapshot looks like: [MySQL bounded test comparing `v7.9.1` vs `v7.10.0-rc2`](https://snapshots.raintank.io/dashboard/snapshot/h356ygrRZIUFWf5u5cctLjFavu97lFR2?orgId=2).

Two considerations:

- Due to [this issue](https://github.com/grafana/grafana/issues/32585), you need to be logged in to access the Snapshot option in the Share dialog. Although logging in is not usually needed in these temporal instances, you can still do so for this purpose with the credentials `admin`/`admin`.
- Note that a snapshot, although very useful for reference, is not a fully-functioning dashboard, so you will not be able to query new data using it. Take a look at the example above to understand how it works.

#### Using an existing Mattermost deployment

1. Set the `AppInstanceCount` and `TerraformDBSettings.InstanceCount` to `0` in the `config/deployer.json`. This will prevent the Mattermost and database cluster from being created.
2. Update the `ConnectionConfiguration` values in `config/config.json` to the correct information for your pre-existing Mattermost deployment.

    `ServerURL` and `WebSocketURL` can be found in your **Mattermost** config.json file under `ServiceSettings`. If `WebSocketURL` is blank, then replace `http` / `https` with the appropriate `ws` or `wss` value.

    ```json
    "ConnectionConfiguration": {
    	"ServerURL": "http://localhost:8065",
    	"WebSocketURL": "ws://localhost:8065",
    	"AdminEmail": "sysadmin@sample.mattermost.com",
    	"AdminPassword": "Sys@dmin-sample1"
    }
    ```

#### Using an existing database deployment

Before attempting to use an existing database, ensure your database will accept the connections from where you've configured your resources to deploy into on AWS.

1. Set the `TerraformDBSettings.InstanceCount` to `0` in the `config/deployer.json`. This will prevent the database cluster from being created.
2. Create a config patch file with the appropriate config settings to access your database. Your patch file should look like the below file, with the values representing your deployment.

    Note: In this example, we will create a patch file with the below called `configPatch.json` stored in the root loadtest folder.

    ```json
    {
        "SqlSettings": {
            "DriverName": "postgres",
            "DataSource": "postgres://mmuser:mostest@databaseURL:port/mattermost_test?sslmode=disable\u0026connect_timeout=10\u0026binary_parameters=yes",
            "DataSourceReplicas": [],
            "DataSourceSearchReplicas": [],
            "MaxIdleConns": 20,
            "ConnMaxLifetimeMilliseconds": 3600000,
            "ConnMaxIdleTimeMilliseconds": 300000,
            "MaxOpenConns": 300,
            "Trace": false,
            "AtRestEncryptKey": "",
            "QueryTimeout": 30,
            "DisableDatabaseSearch": false,
            "MigrationsStatementTimeoutSeconds": 100000,
            "ReplicaLagSettings": []
        }
    }
    ```

3. Modify `MattermostConfigPatchFile` within the `config/deployer.json` file to point to your patch file with an absolute path.

    Example:

    ```json
    "MattermostConfigPatchFile": "/home/ubuntu/mattermost-load-test-ng/configPatch.json",
    ```

4. Continue with the deployment create process.

## Troubleshooting

### Users are not connecting

Make sure that both `Enable Account Creation` and `Enable Open Server` settings are set to `true` in MM System Console.

### Agent logs show several `current team should be set` errors. As a result, users are not joining teams and channels

This can be caused by the app server not being initialized (at least one open team should be created). This can be done manually or through the `ltctl loadtest init` command.
If done manually, `Allow any user with an account on this server to join this team` under `Team Settings` should be set to `Yes`.
Also the `Max Users Per Team` setting in Mattermost System Console should be enough to account for the number of simulated users.

### Agent failing with `MaxActiveUsers is not compatible with max Rlimit value` error

This means the maximum number of file descriptors is lower than what the agent needs to operate. 

The following command can be run to raise the limit to the suggested value:

```sh
ulimit -n VALUE
```

#### Note
For terraform deployments, this value is hard coded in the `systemd` file for the loadtest api. If you need to change the value, you'll have to change the `LimitNOFILE` value in `/lib/systemd/system/ltapi.service` file to a higher value.

1. ssh into your loadtest agents. You can see the agents available by running `go run ./cmd/ltctl ssh`.
2. Modify the `/lib/systemd/system/ltapi.service` file with the new value.
3. Restart the related processes

```bash
sudo systemctl daemon-reload
sudo systemctl restart ltapi
```

You will have to run this for every loadtest agent you have. These will be appended by `agent-` when you run the `ltctl ssh` command above. 



