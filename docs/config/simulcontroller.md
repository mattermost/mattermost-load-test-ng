# SimulController Configuration

## MinIdleTimeMs

*int*

The minium amount of time (in milliseconds) the controlled users will wait between actions.

## AvgIdleTimeMs

*int*

The average amount of time (in milliseconds) the controlled users will wait between actions.

## EnabledPlugins

*[]string*

List of Mattermost plugin manifest IDs whose standalone simulator actions are registered when the controller starts. Actions from enabled plugins appear in logs, coordinator metrics, and rate injection keyed as `<pluginId>.<ActionName>` (see `mattermost-ai` below).

Including a plugin ID here only affects **what the load-test simulator runs**—the Mattermost deployment must still have that plugin installed and enabled (see deployer [`MattermostPlugins`](deployer.md#mattermostplugins) in Terraform / deployer docs).

### `mattermost-ai` (Agents)

When `mattermost-ai` is listed, load-test-ng adds the Agents plugin’s simulative actions: `mattermost-ai.AskAgentChannelMention` and `mattermost-ai.AskAgentDM`. Names are prefixed by the framework; action `Name` values defined in Agents do **not** include the plugin ID.

**Frequency semantics**

Each plugin action’s `frequency` is a **relative weight** in the same weighted action picker as built-in simulator actions—not a standalone global probability. For example, the core action `CreatePost` uses frequency `1.0`, so Agents defaults of `0.001`, `0.005`, or `0.01` correspond to roughly one-thousandth, five-thousandths, or one-hundredth of `CreatePost`’s weight before considering every other action in the list.

Use [coverage-frequency.md](../coverage-frequency.md) to refine frequencies after collecting roughly a week of telemetry (Prometheus / Grafana ratios).

**Agents-specific configuration**

Agents load-test behavior is configured by the Agents `loadtest` package, not `simulcontroller` JSON:

- Defaults apply when `./config/mattermost-ai-loadtest.json` does not exist and `MM_AGENTS_LOADTEST_CONFIG` is unset.
- Set `MM_AGENTS_LOADTEST_CONFIG` to an absolute path to override the JSON file used for Agents trigger frequencies, `triggerMode`, `agentUsername`, and related fields.

## RecapsConfiguration

Configuration for the built-in AI Recaps load-test actions.

### Enabled

*bool*

Enables the `CreateRecap`, `ViewRecaps`, and `CreateScheduledRecap` actions. Defaults to `false`. The target Mattermost server must also have the `EnableAIRecaps` feature flag enabled.

### AgentUsername

*string*

Username of the Agents bot used to process recaps. The controller resolves the username to a user ID once per simulated user. Defaults to `ai`.

### MaxChannelsPerRecap

*int*

Maximum number of randomly selected public or private member channels in an on-demand or `specific` scheduled recap. Each action selects between one and this value. Defaults to `3`.

### PollIntervalMs

*int*

Interval in milliseconds between status requests after creating an on-demand recap. Defaults to `5000`.

### PollTimeoutMs

*int*

Maximum time in milliseconds to poll an on-demand recap. A timeout is reported as action information rather than a fatal controller error so saturation remains measurable. Defaults to `120000`.

### MaxScheduledRecapsPerUser

*int*

Maximum number of schedules the load test creates for each user. Set to `0` to disable schedule creation while retaining the other recap actions. Defaults to `1`.

### ScheduledRecapChannelMode

*string*

Channel selection mode for scheduled recaps: `specific` selects member channels up to `MaxChannelsPerRecap`, while `all_unreads` lets the server choose channels with unread messages. Defaults to `specific`.

### ScheduledRecapDueTime

*string*

Optional scheduled recap due time in `HH:MM` UTC format. When empty, schedule times are spread randomly across the day. When set, every created schedule uses the same minute.

## AI Recaps load testing

AI Recaps actions model two complementary scenarios:

- Steady-state on-demand load: users create recaps from random channels, poll asynchronous processing to a terminal status, and later list, view, and read completed recaps.
- Scheduled burst load: users create recurring daily schedules. Set `ScheduledRecapDueTime` to one UTC minute to align all schedules and generate a morning-storm burst; leave it empty to spread scheduled work throughout the day.
