# SimulController Configuration

## MinIdleTimeMs

*int*

The minium amount of time (in milliseconds) the controlled users will wait between actions.

## AvgIdleTimeMs

*int*

The average amount of time (in milliseconds) the controlled users will wait between actions.

## PercentBatchAlignedScheduledPosts

*float64*

The fraction of newly created scheduled posts, including recurring posts, that are aligned to a shared UTC batch boundary. The default is `0.5`, and the valid range is `0` to `1`.

## ScheduledPostBatchIntervalMinutes

*int*

The number of minutes between shared UTC scheduled-post batch boundaries. The default is `30`, and the valid range is `5` to `60`.

## ScheduledPostBatchMinLeadMinutes

*int*

The minimum lead time in minutes for a shared scheduled-post batch boundary. The default is `10`, and the valid range is `5` to `60`. The next boundary is computed after adding this lead time, so all agents scheduling at the same instant select the same upcoming UTC boundary.

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
