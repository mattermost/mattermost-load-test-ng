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

## ScopedConfig

Settings to load test Scoped Configuration (system setting policies). They only apply when the server exposes the `SystemSettingPolicies` feature flag in the client config. Regardless of these settings, users fetch `/api/v4/config/client/overrides` on login, reload and WebSocket reconnect, and store the overrides received through `system_setting_policy_updated` WebSocket events.

The policies reference user attributes, so the server needs Attribute-Based Access Control enabled (`AccessControlSettings.EnableAttributeBasedAccessControl`), and `AccessControlSettings.EnableUserManagedAttributes` when the attribute fields are user managed, which is the case for the fields created by the load test.

### ScopedConfig.AssignAttributes

*bool*

Whether users assign themselves a random option for each field in `AttributeFields` they don't have a value for yet when they log in, so that each policy matches a fraction of the users.

### ScopedConfig.EnablePolicyChurn

*bool*

Whether the first controller of the agent logs in as the admin user (`ConnectionConfiguration.AdminEmail`) and churns policies. On start, it creates the `AttributeFields` that are missing and enables `SessionAttribute`. Then it periodically creates, updates (user attribute expression, session attribute expression, value or role) or deletes the policy of a random key in `SettingKeys`, or adds or removes a spare option of an attribute field. It also periodically sets a random attribute of a random user. Only the policies with a name starting with `loadtest-` are modified.

When running through the coordinator, only the first agent keeps this enabled so that the churn rate doesn't depend on the number of agents.

### ScopedConfig.PolicyChurnIntervalMs

*int*

The time (in milliseconds) between two policy changes. A value of 0 disables them.

### ScopedConfig.AttributeChurnIntervalMs

*int*

The time (in milliseconds) between two admin updates of a random user's attribute value. A value of 0 disables them.

### ScopedConfig.SettingKeys

*[]string*

The client config settings to create policies for. Each one can have a single policy. Defaults to `["MobileEnableBiometrics"]`.

### ScopedConfig.SessionAttribute

*string*

The session attribute referenced by session expressions. Load test agents report a `user_agent_browser_name` of `Unknown`, so the default `user_agent_browser_name` attribute matches all of their sessions. An empty value disables session expressions.

### ScopedConfig.AttributeFields

*[]{Name string, Options []string}*

The select attribute fields referenced by policy expressions. Defaults to a `Department` field with six options and a `Location` field with three options.
