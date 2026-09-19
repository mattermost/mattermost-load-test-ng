# AI Recaps load testing

This runbook covers steady on-demand recap traffic and an aligned scheduled-recap burst against an unreleased Mattermost Enterprise build.

## Prerequisites

- A Mattermost Enterprise/E20 license. The `mattermost-ai` plugin and Enterprise metrics require a valid license.
- A custom Enterprise Mattermost tarball that includes AI Recaps, available as a local `file://` URI or a URL reachable from every app and job-server instance.
- A `mattermost-ai` plugin tarball available by the same mechanism.
- A load-test-ng package that includes the Recaps simulator actions.
- A deployment with metrics enabled.

The mock intentionally makes no external LLM requests and returns canned responses, so never configure it in production.

## Deploy Mattermost and Agents

Add the custom artifacts, license, config patch, and feature-flag environment variable to `config/deployer.json`:

```json
{
  "MattermostDownloadURL": "file:///absolute/path/to/mattermost-enterprise-linux-amd64.tar.gz",
  "MattermostLicenseFile": "/absolute/path/to/mattermost.mattermost-license",
  "MattermostConfigPatchFile": "/absolute/path/to/recaps-config-patch.json",
  "MattermostEnvVars": {
    "MM_FEATUREFLAGS_ENABLEAIRECAPS": "true"
  },
  "MattermostPlugins": {
    "mattermost-ai": "file:///absolute/path/to/mattermost-ai.tar.gz"
  },
  "JobServerSettings": {
    "InstanceCount": 0,
    "InstanceType": "c7i.xlarge"
  }
}
```

An HTTPS or S3-backed HTTPS URL can replace either `file://` URI. `MattermostEnvVars` is rendered into both application and dedicated job-server services. The flag cannot be enabled through `config.json`.

Use the following `recaps-config-patch.json` as a starting point:

```json
{
  "AIRecapSettings": {
    "Enable": true,
    "DefaultLimits": {
      "MaxRecapsPerDay": -1,
      "MaxScheduledRecaps": -1,
      "CooldownMinutes": 0,
      "MaxPostsPerRecap": -1,
      "MaxPostsPerDay": -1
    },
    "Processing": {
      "MaxConcurrentJobs": 4,
      "MaxConcurrentLLMCalls": 16,
      "MaxDueSchedulesPerTick": 1000
    }
  }
}
```

`-1` disables the corresponding recap limit; `CooldownMinutes: 0` disables the cooldown. Keep finite limits when testing limit enforcement rather than capacity. The server defaults are 10 recaps/day, 5 schedules, a 60-minute cooldown, 500 posts/recap, and 5000 posts/day. Processing defaults are 4 concurrent jobs, 16 concurrent LLM calls, and 1000 due schedules per scheduler tick.

Create the deployment as usual:

```sh
go run ./cmd/ltctl deployment create
```

## Configure the load-test mock

After Mattermost and the plugin are running, send the following dedicated load-test configuration to `PUT /plugins/mattermost-ai/admin/config`. This payload configures both the service and the `ai` bot used by `RecapsConfiguration.AgentUsername`.

```json
{
  "services": [
    {
      "id": "recaps-loadtest-mock",
      "name": "Recaps Load Test Mock",
      "type": "loadtest_mock",
      "loadTestMockConfig": {
        "name": "recaps_load_test",
        "seed": 42,
        "latency_profiles": {
          "realistic_default": {
            "ttft_ms": [3000, 12000],
            "chunk_count": [150, 400],
            "chunk_interval_ms": [30, 80],
            "total_wall_time_ms_per_request": [15000, 25000]
          },
          "realistic_fast": {
            "ttft_ms": [600, 2500],
            "chunk_count": [40, 120],
            "chunk_interval_ms": [40, 100],
            "total_wall_time_ms_per_request": [5000, 10000]
          },
          "realistic_slow": {
            "ttft_ms": [12000, 22000],
            "chunk_count": [400, 1000],
            "chunk_interval_ms": [15, 40],
            "total_wall_time_ms_per_request": [28000, 40000]
          }
        },
        "profile_weights": {
          "realistic_default": 0.7,
          "realistic_fast": 0.2,
          "realistic_slow": 0.1
        },
        "streaming_enabled": true,
        "tool_use_probability": 0,
        "final_response_templates": [
          "{\"highlights\":[\"Highlight %[1]d from the mock LLM\"],\"action_items\":[\"Action item %[1]d\"]}"
        ]
      }
    }
  ],
  "bots": [
    {
      "id": "recaps-loadtest-agent",
      "name": "ai",
      "displayName": "AI",
      "customInstructions": "Return recap highlights and action items.",
      "serviceID": "recaps-loadtest-mock",
      "model": "",
      "enableVision": false,
      "disableTools": true,
      "channelAccessLevel": 0,
      "channelIDs": [],
      "userAccessLevel": 0,
      "userIDs": [],
      "teamIDs": [],
      "maxFileSize": 0,
      "enabledNativeTools": [],
      "enabledMCPTools": [],
      "autoEnableNewMCPTools": false,
      "mcpDynamicToolLoading": false,
      "reasoningEnabled": false,
      "reasoningEffort": "",
      "thinkingBudget": 0,
      "structuredOutputEnabled": false,
      "maxToolTurns": 0
    }
  ],
  "defaultBotName": "ai",
  "transcriptBackend": "",
  "enableTokenUsageLogging": false,
  "enableCallSummary": false,
  "allowedUpstreamHostnames": "",
  "allowUnsafeLinks": false,
  "enableChannelMentionToolCalling": false,
  "allowNativeWebSearchInChannels": false,
  "mcp": {
    "enabled": true,
    "enablePluginServer": false,
    "servers": [],
    "embeddedServer": {
      "enabled": true
    },
    "idleTimeoutMinutes": 30
  },
  "webSearch": {
    "enabled": false,
    "provider": "",
    "google": {},
    "brave": {},
    "domainDenylist": []
  },
  "telemetryOutput": "",
  "openTelemetryEndpoint": ""
}
```

The positional format verb `%[1]d` reuses the mock request sequence while keeping every completion valid JSON. Recap processing parses the completion as `{highlights, action_items}`; ordinary prose templates will fail.

Apply the payload with a system-admin token:

```sh
curl --fail-with-body \
  -X PUT "$MATTERMOST_URL/plugins/mattermost-ai/admin/config" \
  -H "Authorization: Bearer $MATTERMOST_TOKEN" \
  -H "Content-Type: application/json" \
  --data-binary @recaps-agents-config.json
```

The mock has no token, request, or LLM rate limit by design. Its only simulated provider cost is inference time, selected deterministically from the configured latency profiles and weights. Keep the seed and profile fixed when comparing server configurations.

## Configure Recaps simulation

See the [SimulController RecapsConfiguration reference](config/simulcontroller.md#recapsconfiguration) for every field and its default.

### Scenario A: steady-state on-demand

Enable recaps and disable schedule creation so the normal simulative action mix produces only on-demand recap creation and viewing:

```json
"RecapsConfiguration": {
  "Enabled": true,
  "AgentUsername": "ai",
  "MaxChannelsPerRecap": 3,
  "PollIntervalMs": 5000,
  "PollTimeoutMs": 120000,
  "MaxScheduledRecapsPerUser": 0,
  "ScheduledRecapDueTime": ""
}
```

Run the usual bounded simulative load, wait for the active-user count and recap latency to stabilize, and compare equivalent windows across builds. Poll timeouts are reported as action information instead of fatal controller errors, so inspect recap status and server metrics rather than treating a completed coordinator run as proof that all recaps finished.

### Scenario B: scheduled burst ("morning storm")

Choose a future UTC minute with enough lead time to connect all users:

```json
"RecapsConfiguration": {
  "Enabled": true,
  "AgentUsername": "ai",
  "MaxChannelsPerRecap": 3,
  "PollIntervalMs": 5000,
  "PollTimeoutMs": 120000,
  "MaxScheduledRecapsPerUser": 1,
  "ScheduledRecapChannelMode": "specific",
  "ScheduledRecapDueTime": "08:00"
}
```

To seed one aligned schedule for each of N active users:

1. Start a bounded run with `MaxActiveUsers` set to N and wait until all N users are active.
2. Run `go run ./cmd/ltctl loadtest inject CreateScheduledRecap`. Injection runs the action once for every active simulated user, bypassing its low normal action frequency.
3. Check agent logs for creation failures and confirm the expected schedule count before the due minute. Existing schedules count toward `MaxScheduledRecapsPerUser`; use fresh users or remove old schedules between trials.
4. Keep the normal simulative traffic running through the due minute and until the backlog returns to zero and delivery-delay observations stop arriving.

Use `all_unreads` instead of `specific` only when the test data has a controlled unread distribution. Otherwise, changing unread state changes work per recap and makes knob comparisons noisy.

## Observe the run

The default Grafana dashboard includes an **AI Recaps** row. Watch:

| Metric | Interpretation |
|---|---|
| `mattermost_recaps_delivery_delay_seconds` | Scheduled due-to-delivery latency. The primary SLO is p99 below 300 seconds. |
| `mattermost_recaps_scheduled_backlog` | Due schedules seen on the last leader tick, capped by `MaxDueSchedulesPerTick`. Only the leader reports the meaningful value. |
| `mattermost_recaps_llm_calls_in_flight` | Per-node channel summarizations currently using the mock LLM. Compare each node with `MaxConcurrentLLMCalls`; sum only when measuring cluster throughput. |
| `mattermost_recaps_channel_process_time_seconds{success}` | Per-channel processing latency and successful/failed channel throughput. |

Also watch CPU, memory, database latency and connections, HTTP errors, and Mattermost job logs/queue behavior. A low LLM gauge with a growing backlog points to job-worker or scheduler limits; a gauge pinned at `MaxConcurrentLLMCalls` points to the LLM semaphore or the configured inference latency.

For an unbounded "find the ceiling" run, add the delivery SLO to `MonitorConfig.Queries` in `config/coordinator.json`:

```json
{
  "Description": "AI Recaps scheduled delivery delay p99",
  "Legend": "P99 seconds",
  "Query": "histogram_quantile(0.99, sum(rate(mattermost_recaps_delivery_delay_seconds_bucket[5m])) by (le))",
  "Threshold": 300,
  "MinIntervalSec": 300,
  "Alert": true
}
```

This query controls the coordinator feedback loop, so use it only when scheduled deliveries are present. `MinIntervalSec` avoids alerting before the five-minute rate window is populated.

## Sweep processing capacity

Change one variable per deployment and repeat the same N, due time relative to test start, mock seed, latency mix, channel mode, and data set:

1. Establish a baseline with `MaxConcurrentJobs: 4`, `MaxConcurrentLLMCalls: 16`, and `MaxDueSchedulesPerTick: 1000`.
2. Increase `MaxDueSchedulesPerTick` until a due burst is admitted in sufficiently large batches. If the backlog panel repeatedly plateaus at this value, the gauge is reporting the capped tick result, not necessarily the full database backlog.
3. Sweep `MaxConcurrentJobs` to determine whether recap job workers can drain the admitted work.
4. Sweep `MaxConcurrentLLMCalls` while comparing the per-node in-flight gauge with the configured cap. Stop increasing it when delivery p99 no longer improves or CPU, database, or plugin saturation gets worse.
5. Repeat the useful combinations with `JobServerSettings.InstanceCount: 1`. A dedicated job server isolates periodic and recap jobs from app nodes, but changes the number and placement of worker pools; treat it as a separate topology rather than another value in the same-node comparison.

Restart the Mattermost/job-server services after changing processing settings so worker pool sizes are recreated. Record peak backlog, time to drain, delivery p50/p99, channel p50/p99, success/failure throughput, peak in-flight LLM calls, and host/database utilization for every trial.

## Troubleshooting

- No recap API traffic: confirm `RecapsConfiguration.Enabled`, server version support, and the feature-flag environment in `systemctl show mattermost --property=Environment`.
- Recaps report that the feature is disabled: verify both `MM_FEATUREFLAGS_ENABLEAIRECAPS=true` and `AIRecapSettings.Enable`.
- The agent cannot be resolved: verify the configured bot username is `ai` and matches `AgentUsername`.
- Every recap fails while mock calls complete: inspect the completion. It must remain valid recap-shaped JSON after `%d` formatting.
- Scheduled work never starts on a dedicated job server: confirm the custom server build, license, feature flag, and config patch are present there, then inspect the job-server journal.
- Delivery p99 is empty: histogram quantiles require completed scheduled deliveries in the selected five-minute window.
