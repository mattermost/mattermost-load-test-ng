# Smoke simulations

A lightweight framework for running smoke tests on Mattermost browser sessions without actually starting the load test. This tool can be used to quickly verify simulations before running full load tests.


Smoke simulations allow you to:
- **Quickly validate** core Mattermost functionality before deployments or updates
- **Test multiple user scenarios** simultaneously with different simulations per user
- **Monitor active sessions** in real-time with configurable reporting intervals
- **Run time-limited tests** with automatic cleanup after a specified duration

## Running smoke simulations

1. Create Configuration File

    Copy the sample configuration and customize it for your environment:

    ```bash
    cp smoke_simulation.sample.json smoke_simulation.json
    ```

1. Configure your test by editing `smoke_simulation.json`

1. Run smoke tests

    ```bash
    make smoke-simulation
    ```

## Configuration reference

| Field | Type | Description |
|-------|------|-------------|
| `users` | Array | List of user credentials to create browser sessions for. Must match the length of the `simulations` array. Each user will run the simulations in the order they are specified in the `simulations` array. |
| `users[].username` | string | Username or email for authentication |
| `users[].password` | string | User password |
| `simulations` | Array | Simulation IDs to run for each user. Must match the length of the `users` array. |
| `serverURL` | string | Mattermost server URL (e.g., `http://localhost:8065`) |
| `sessionMonitorIntervalMs` | number | Interval in milliseconds for session status reporting (0 to disable) |
| `testDurationMs` | number | Total test duration in milliseconds before automatically stopping the tests |
| `runInHeadless` | boolean | Run browsers in headless mode (default: true) |

## Monitoring Output

The monitor provides real-time feedback on active sessions:

```
🔍 Starting monitor
🔍 Creating session for user1@example.com
✅ Session created: Session initialized successfully
⌛️ Starting simulation mattermostPostAndScroll for user1@example.com
📋 Active browser sessions: user1@example.com->active, user2@example.com->active
```

## Example workflows

### Basic smoke test (2 minutes)

```json
{
  "users": [
    {"username": "admin@example.com", "password": "Admin123!"}
  ],
  "serverURL": "http://localhost:8065",
  "sessionMonitorIntervalMs": 5000,
  "testDurationMs": 120000,
  "simulations": ["mattermostPostAndScroll"],
  "runInHeadless": false
}
```

### Multi-user load test (10 minutes)

```json
{
  "users": [
    {"username": "user1@example.com", "password": "Pass1!"},
    {"username": "user2@example.com", "password": "Pass2!"},
    {"username": "user3@example.com", "password": "Pass3!"}
  ],
  "serverURL": "https://staging.mattermost.com",
  "sessionMonitorIntervalMs": 10000,
  "testDurationMs": 600000,
  "simulations": ["mattermostPostAndScroll", "mattermostPostAndScroll", "mattermostPostAndScroll"],
  "runInHeadless": true
}
```