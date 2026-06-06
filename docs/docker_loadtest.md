# Running a load test via the Docker container

This guide covers the simplest production-style workflow: pull the published
image to any host with Docker, point it at a Mattermost instance, and watch
your existing Grafana while a manual load ramp finds the cliff.

No coordinator, no extra Prometheus, no compose stack. Just one container and
your eyes on a dashboard.

## When to use this guide

Use this when:

- You want to find when a specific Mattermost instance starts to fall over.
- Mattermost is already running and its `/metrics` endpoint is scraped by
  some Prometheus, with Grafana on top.
- You can run Docker on a host that has network reachability to Mattermost.

If you instead want unattended, automated "find max users" runs that converge
on a single number, you want the coordinator + Prometheus setup — see
[`local_loadtest.md`](./local_loadtest.md) and [`coordinator.md`](./coordinator.md).

## Prerequisites

Verify each before starting:

1. **Docker** is installed on the host. Test: `docker version`.
2. **The image** is available to the host. Options:
   - Pull from Docker Hub (open networks): `docker pull mattermost/mattermost-load-test-ng:vX.Y.Z`
   - Side-load (closed networks): `docker load -i mattermost-load-test-ng.tar`
     where the tar file came from `docker save` on a connected build machine.
3. **Mattermost is reachable** from the host. Test:
   `curl -v https://mattermost.example.com/api/v4/system/ping`. Expect HTTP 200.
4. **Mattermost system admin credentials** (email + password).
5. **A Grafana that's already viewing Mattermost metrics.** Cluster Grafana,
   the team's existing one, anything that has at least a panel for
   `mattermost_api_time_*` and request rate / error rate / latency.
6. **For Enterprise:** a valid license already applied to Mattermost.

## Step 1: Write a config file

On the load-test host, create `config.json`. Adjust `ServerURL`,
`WebSocketURL`, `AdminEmail`, `AdminPassword`, and `MaxActiveUsers` for your
environment. The other defaults are a reasonable simulative-user profile.

```json
{
  "ConnectionConfiguration": {
    "ServerURL": "https://mattermost.example.com",
    "WebSocketURL": "wss://mattermost.example.com",
    "AdminEmail": "sysadmin@example.com",
    "AdminPassword": "REPLACE-ME"
  },
  "UserControllerConfiguration": {
    "Type": "simulative",
    "RatesDistribution": [
      { "Rate": 1.0,  "Percentage": 0.05 },
      { "Rate": 2.0,  "Percentage": 0.10 },
      { "Rate": 3.0,  "Percentage": 0.15 },
      { "Rate": 6.0,  "Percentage": 0.40 },
      { "Rate": 30.0, "Percentage": 0.30 }
    ],
    "ServerVersion": ""
  },
  "InstanceConfiguration": {
    "NumTeams": 2,
    "NumChannels": 10,
    "NumPosts": 0,
    "NumReactions": 0,
    "NumAdmins": 0,
    "PercentReplies": 0.5,
    "PercentRepliesInLongThreads": 0.05,
    "PercentUrgentPosts": 0.001,
    "PercentPublicChannels": 0.2,
    "PercentPrivateChannels": 0.1,
    "PercentDirectChannels": 0.6,
    "PercentGroupChannels": 0.1
  },
  "UsersConfiguration": {
    "InitialActiveUsers": 0,
    "UsersFilePath": "",
    "MaxActiveUsers": 2000,
    "MaxActiveBrowserUsers": 0,
    "AvgSessionsPerUser": 1,
    "PercentOfUsersAreAdmin": 0.0005
  },
  "LogSettings": {
    "EnableConsole": true,
    "ConsoleLevel": "ERROR",
    "ConsoleJson": false,
    "EnableFile": true,
    "FileLevel": "INFO",
    "FileJson": true,
    "FileLocation": "/mattermost-load-test/logs/ltagent.log",
    "EnableColor": false
  }
}
```

Note: `AdminPassword` sits in this file as plaintext. For test/dev island
use this is fine. For production-similar runs, the operator should put the
password in a Docker secret or an env file with restricted file permissions
on the host, not check it into version control.

## Step 2: Seed Mattermost with test data (one-time)

Run `ltagent init` to create the initial teams and channels that simulated
users will join.

```bash
mkdir -p ./logs
docker run --rm \
  -v "$(pwd)/config.json:/mattermost-load-test/config/config.json:ro" \
  -v "$(pwd)/logs:/mattermost-load-test/logs" \
  mattermost/mattermost-load-test-ng:vX.Y.Z \
  ltagent init
```

This is safe to skip if you've already seeded the target with previous runs.

## Step 3: Run the load test — manual ramp

The pattern: start small, watch Grafana, double or step up until you see
degradation.

```bash
# 100 users for 10 minutes
docker run --rm \
  -v "$(pwd)/config.json:/mattermost-load-test/config/config.json:ro" \
  -v "$(pwd)/logs:/mattermost-load-test/logs" \
  mattermost/mattermost-load-test-ng:vX.Y.Z \
  ltagent -n 100 -d 600
```

Re-run with larger `-n` until your Grafana dashboard shows the system under
stress:

```bash
docker run --rm ... ltagent -n 250  -d 600
docker run --rm ... ltagent -n 500  -d 600
docker run --rm ... ltagent -n 1000 -d 600
```

## Step 4: What to watch on Grafana

The metrics that matter, in roughly the order they degrade:

| Signal                                  | What "starting to crack" looks like                  |
| --------------------------------------- | ---------------------------------------------------- |
| Average / P99 API response time         | Climbing past your team's SLO (often P99 > 2s)       |
| HTTP error rate (4xx and 5xx)           | Sustained > ~0.5%                                    |
| Mattermost pod CPU                      | Sustained > 85%                                      |
| Mattermost pod memory                   | Climbing without leveling off                        |
| Database connection pool utilization    | Saturated; new connections waiting                   |
| WebSocket connections established       | Diverging from active-user count                     |
| Goroutine count (`go_goroutines`)       | Climbing linearly with no plateau — signals leak     |

**Define "the cliff" up front so the number you report survives review.**
A useful default: "P99 API response time exceeds 2 seconds for three
consecutive minutes, OR 5xx error rate exceeds 0.5% for one minute." Whatever
you pick, write it down with the test result.

## Step 5: Stop and clean up

`ltagent -d 600` exits on its own after the configured duration. To stop
early, `Ctrl-C` the docker run or:

```bash
docker ps        # find the container ID
docker stop <id>
```

Logs are in `./logs/ltagent.log` on the host.

## Common configurations and troubleshooting

### "Connection refused" or DNS failures
The container can't reach Mattermost. From the host: `curl -v <ServerURL>/api/v4/system/ping`.
If that fails, the issue is host-level connectivity (firewall, DNS, VPN),
not the container.

### "401 Unauthorized" during init
`AdminEmail` / `AdminPassword` in config.json don't match a system admin
account in Mattermost. Verify by logging into Mattermost manually with the
same credentials.

### Test starts but no users actually log in
Check `./logs/ltagent.log`. Common cause: Mattermost rate-limiting at the
ingress (nginx-ingress default is 5 req/s per IP). Either raise the limit
in Mattermost's ingress config for the test window, or add the load-test
host to a rate-limit-exempt allowlist.

### "Too many open files"
The kernel ulimit on the host is too low for the user count. On the host:
```bash
ulimit -n              # check current
sudo sysctl -w fs.file-max=1000000
ulimit -n 65536
```
Then re-run the container.

### Closed-network registry handling
If your host can't reach docker.io, build on a connected machine and ship
the tarball:
```bash
# On the connected machine:
docker pull mattermost/mattermost-load-test-ng:vX.Y.Z
docker save mattermost/mattermost-load-test-ng:vX.Y.Z -o lt.tar
scp lt.tar user@host:/tmp/
# On the closed host:
docker load -i /tmp/lt.tar
```

## What this guide deliberately does not cover

- **Multi-agent / distributed load.** Single container generates ~500–2000
  users worth of load depending on host VM size. For higher loads, see the
  multi-agent terraform deployment in `terraform_loadtest.md`.
- **Automated "find max users" runs.** That requires the coordinator and a
  Prometheus the coordinator can query. See `coordinator.md`.
- **Browser-based simulation.** Requires the LTBrowser sidecar. See
  `plugin_browser_loadtest.md`.
- **Comparing two Mattermost versions head-to-head.** See `compare.md`.

The container shipped here can do all of those — this guide just covers the
simplest single-agent manual-ramp use case.
