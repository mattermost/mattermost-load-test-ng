## FAQ

### What login methods does the load-test agent support?

The load-test agent currently supports only the email/password authentication method.

### What is a bounded load-test?

We define a load-test to be bounded when the number of simulated users is fixed.
This is particularly useful when running performance comparisons between two clusters/builds.

#### Note

To manually run a bounded load-test using the [`coordinator`](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/coordinator.md) the [feedback loop](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/coordinator.md#the-feedback-loop) should be disabled. This can be done by either removing `MonitorConfig.Queries` or by disabling each query (by setting `Alert` to `false`).

### What is an unbounded load-test?

We define a load-test to be unbounded when the number of simulated users can vary up to a pre-configured limit.
This type of load-test is used to determine the capacity of a system and will output an estimated number of users.
The rule of thumb is that when starting an unbounded load-test we should always shoot for more users than what we think an installation can support.
`ClusterConfig.MaxActiveUsers`  should be set to  `AgentInstanceCount * UsersConfiguration.MaxActiveUsers`.

## Troubleshooting

### Users are not connecting

Make sure that both `Enable Account Creation` and `Enable Open Server` settings are set to `true` in MM System Console.

### Agent logs show several `current team should be set` errors. As a result, users are not joining teams and channels

This can be caused by the app server not being initialized (at least one open team should be created). This can be done manually or through the `ltctl loadtest init` command.
If done manually, `Allow any user with an account on this server to join this team` under `Team Settings` should be set to `Yes`.
Also the `Max Users Per Team` setting in Mattermost System Console should be enough to account for the number of simulated users.
