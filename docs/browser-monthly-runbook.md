# Monthly Browser Load Testing Runbook (7 hours, 10 agents, 5 sessions/agent)

Audience: QA

Purpose: A repeatable, production‑like, monthly procedure to run a 7‑hour browser load test using 10 browser‑agent instances (c7i.xlarge) with 5 headless Chromium sessions per agent (50 total concurrent sessions), monitor health, and capture evidence for a pass/fail decision.


## 1) Test Definition

- Instance type: c7i.xlarge (8 GB RAM, 4 vCPUs) per agent
- Agents: 10 browser agents (one per instance)
- Sessions per agent: 5 (sustained)
- Total sessions: 50
- Simulation: `postAndScroll` (browser side)
- Test duration: 7 hours (25200 seconds)
- Pass criteria (SLOs):
  - Channel switch p90 < 250 ms (steady state)
  - Memory p99 per agent < 70% of system RAM
  - 0 Playwright timeouts in logs
  - No WebSocket connection drops


## 2) Prerequisites

- Access
  - AWS IAM to provision or access the 10 agents
  - SSH to all agents
  - Prometheus + Grafana reachable
- Tooling on your workstation (or coordinator host)
  - Go 1.21+
  - Git
- On each agent
  - Node.js 18+
  - Go 1.21+
  - Chromium dependencies for Playwright
  - This repository present at `~/mattermost-load-test-ng`
- Networking
  - Agent API: 4000/tcp (reachable from the coordinator)
  - LTBrowser API: 5000/tcp (local on agent is sufficient)

If you use Terraform with this repo, see `docs/terraform_loadtest.md`. Otherwise, make sure you have 10 ready agents and 1 coordinator host (can be your workstation if it can reach agents).


## 3) Configuration

### 3.1 Agent config (`config/config.json`) on every agent

Set to 5 browsers per agent and safe browser defaults.

```json
{
  "ConnectionConfiguration": {
    "ServerURL": "https://your-mattermost.example.com"
  },
  "UsersConfiguration": {
    "MaxActiveBrowserUsers": 5,
    "InitialActiveUsers": 0
  },
  "BrowserConfiguration": {
    "Headless": true,
    "SimulationTimeoutMs": 60000
  },
  "BrowserLogSettings": {
    "EnableConsole": false,
    "ConsoleLevel": "error",
    "EnableFile": true,
    "FileLevel": "info",
    "FileLocation": "browseragent.log"
  }
}
```

Notes:
- `SimulationTimeoutMs` is used to set Playwright page default timeout.
- Browser service logs to `browseragent.log` (see `browser/src/utils/config.ts`).


### 3.2 Coordinator config (`config/coordinator.json`) on the coordinator

List all 10 agents (their `ltagent` API endpoints) and set the cluster total to 50.

```json
{
  "ClusterConfig": {
    "Agents": [],
    "MaxActiveUsers": 0,
    "BrowserAgents": [
      { "Id": "agent-1",  "ApiURL": "http://AGENT_1_IP:4000" },
      { "Id": "agent-2",  "ApiURL": "http://AGENT_2_IP:4000" },
      { "Id": "agent-3",  "ApiURL": "http://AGENT_3_IP:4000" },
      { "Id": "agent-4",  "ApiURL": "http://AGENT_4_IP:4000" },
      { "Id": "agent-5",  "ApiURL": "http://AGENT_5_IP:4000" },
      { "Id": "agent-6",  "ApiURL": "http://AGENT_6_IP:4000" },
      { "Id": "agent-7",  "ApiURL": "http://AGENT_7_IP:4000" },
      { "Id": "agent-8",  "ApiURL": "http://AGENT_8_IP:4000" },
      { "Id": "agent-9",  "ApiURL": "http://AGENT_9_IP:4000" },
      { "Id": "agent-10", "ApiURL": "http://AGENT_10_IP:4000" }
    ],
    "MaxActiveBrowserUsers": 50
  },
  "MonitorConfig": {
    "PrometheusURL": "http://PROMETHEUS_HOST:9090"
  },
  "NumUsersInc": 1,
  "NumUsersDec": 1,
  "RestTimeSec": 2
}
```


## 4) Start services on each agent (run on all 10)

### 4.1 Start LTBrowser API (port 5000)

The browser API server is TypeScript and runs with `tsx` (see `browser/src/server.ts`).

```bash
cd ~/mattermost-load-test-ng/browser
npm ci
npx playwright install --with-deps chromium

# Run LTBrowser API (fastify) on port 5000
npx tsx src/server.ts 2>&1 | tee -a ~/browserapi.out &

# Health check
curl -sf http://localhost:5000/health || echo "LTBrowser API not healthy"
```

### 4.2 Start load-test agent (`ltagent`, port 4000)

```bash
cd ~/mattermost-load-test-ng
go run ./cmd/ltagent --config ./config/config.json 2>&1 | tee -a ~/ltagent.out &

# Health check
curl -sf http://localhost:4000/status || echo "ltagent not healthy"
```


## 5) Start the coordinator

On the coordinator host (can be your workstation if it can reach agents on port 4000):

```bash
cd ~/mattermost-load-test-ng
go run ./cmd/ltcoordinator --config ./config/coordinator.json 2>&1 | tee -a ~/ltcoordinator.out &
```


## 6) Launch the 7‑hour test

Use `ltctl` to start and stop.

```bash
# Start clustered browser load test to reach 50 sessions
go run ./cmd/ltctl loadtest start

# Confirm running status
go run ./cmd/ltctl loadtest status

# Schedule an automatic stop after 7 hours (25200 seconds)
( sleep 25200 && go run ./cmd/ltctl loadtest stop ) >/dev/null 2>&1 &
```


## 7) Monitoring during the run

- Grafana dashboards to watch:
  - WebSocket connections per agent and total → should plateau at 50
  - Channel switch time p90 → must remain < 250 ms
  - HTTP error and timeout rates → should be near zero
  - Node memory utilization per agent → p99 < 70% RAM

PromQL (examples):

```promql
# Channel switch time p90 (example metric name – adjust to your dashboard)
histogram_quantile(0.90, rate(mattermost_channel_switch_duration_seconds_bucket[5m])) * 1000

# Active WebSocket connections from the load-test user entities
loadtest_websocket_connections_total

# HTTP error rate
rate(loadtest_http_errors_total[5m])

# HTTP timeout rate
rate(loadtest_http_timeouts_total[5m])
```

Optional per-agent memory CSV sampling (helper script present at repo root):

```bash
./monitor_memory.sh <agent_ssh_alias> > memory_usage_agentX.csv &
```


## 8) Pass/Fail criteria

Evaluate at steady state (post ramp-up):

- Channel switch p90 < 250 ms and < 20% drift across the window
- Memory p99 per agent < 70% of RAM (≈ 5.4 GB of ~7.8 GB usable)
- 0 Playwright timeouts in `browseragent.log`
- Flat WebSocket connections at 50 (no unexplained drops)

If any threshold is exceeded → mark the run Fail and attach findings.


## 9) Artifact collection

After stop:

- From each agent:
  - `~/browseragent.log`
  - `~/browserapi.out`, `~/ltagent.out`
  - Any screenshots under `browser/screenshots/` (should be none on success)
  - Optional: `memory_usage_agentX.csv`
- From coordinator:
  - `~/ltcoordinator.out`
- From Grafana:
  - Export images/CSVs for channel switch time, WebSocket connections, HTTP errors/timeouts, and system memory

Store under a date‑stamped folder, e.g., `runs/YYYY-MM/`.


## 10) Monthly summary template

```text
Monthly Browser Load Test – YYYY-MM

Scope:
- 10 agents (c7i.xlarge), 5 sessions/agent, 7 hours, total 50 sessions
- Simulation: postAndScroll

SLOs:
- Channel switch p90 < 250 ms: PASS/FAIL (observed: ___ ms)
- Memory p99 per agent < 70%: PASS/FAIL (observed: ___ %)
- Playwright timeouts: PASS/FAIL (count: ___)
- WebSocket drops: PASS/FAIL (observed drops: ___)

Issues Observed:
- [ ] None
- [ ] (List)

Links:
- Grafana dashboard: <link>
- Artifacts: s3://.../runs/YYYY-MM/
- Coordinator logs: <path>
- Agent logs: <path>

Decision:
- [ ] PASS – within thresholds
- [ ] FAIL – follow-up required (owner: ___, due: ___)
```


## 11) Troubleshooting

- LTBrowser API not healthy
  - Ensure `npx tsx src/server.ts` is running on port 5000
  - Re‑run `npx playwright install --with-deps chromium`
- Agent not reachable from coordinator
  - Verify security groups allow 4000/tcp from coordinator
  - `curl -sf http://AGENT_IP:4000/status`
- Users don’t reach 50
  - Check each agent’s `config/config.json` (`MaxActiveBrowserUsers: 5`)
  - `go run ./cmd/ltctl loadtest status` for errors
  - Inspect `~/ltagent.out` and `~/browseragent.log`
- High channel switch times
  - Check per‑agent memory p99; if ≥ 70%, reduce sessions or investigate server
  - Review HTTP errors/timeouts and server response times


## 12) Optional: systemd units

`/etc/systemd/system/mm-browser-api.service`

```ini
[Unit]
Description=Mattermost LT Browser API
After=network.target

[Service]
WorkingDirectory=/home/ubuntu/mattermost-load-test-ng/browser
ExecStart=/usr/bin/npx tsx /home/ubuntu/mattermost-load-test-ng/browser/src/server.ts
Restart=always
Environment=NODE_ENV=production
StandardOutput=append:/home/ubuntu/browserapi.out
StandardError=append:/home/ubuntu/browserapi.out

[Install]
WantedBy=multi-user.target
```

`/etc/systemd/system/mm-ltagent.service`

```ini
[Unit]
Description=Mattermost Load Test Agent
After=network.target

[Service]
WorkingDirectory=/home/ubuntu/mattermost-load-test-ng
ExecStart=/usr/local/go/bin/go run /home/ubuntu/mattermost-load-test-ng/cmd/ltagent --config /home/ubuntu/mattermost-load-test-ng/config/config.json
Restart=always
StandardOutput=append:/home/ubuntu/ltagent.out
StandardError=append:/home/ubuntu/ltagent.out

[Install]
WantedBy=multi-user.target
```

Enable:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now mm-browser-api.service mm-ltagent.service
```


## 13) Monthly cadence checklist

- 7 days before: verify infra, access, Grafana/Prometheus
- 1 day before: 30‑minute smoke (2 agents × 2 sessions)
- Test day:
  - Start services on all agents
  - Start coordinator
  - Launch test; schedule stop at 7h
  - Monitor first 30m and last 30m closely
- After test:
  - Collect artifacts and complete summary
  - If FAIL: file follow‑up ticket(s) with owners and due dates
  - Archive `runs/YYYY-MM/`


---

References:
- Agent API & coordination flow: `docs/load-test-how-to-use.md`
- Terraform deployment: `docs/terraform_loadtest.md`
- Browser service configuration: `browser/src/utils/config.ts`
- Browser controller → LTBrowser API contract: `loadtest/control/browsercontroller/controller.go`


