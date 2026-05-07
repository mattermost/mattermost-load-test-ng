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
