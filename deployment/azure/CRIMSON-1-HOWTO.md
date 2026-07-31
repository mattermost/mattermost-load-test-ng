# How-to: Portable load test in Crimson-1's Azure environment

Audience: Crimson-1 operators who already have Mattermost, Prometheus, and Grafana running in their Azure environment, plus a machine that can build/receive the Docker image (referred to below as the **build machine** and the **agent VM**: these may be the same machine, or the build machine may be your own laptop and the image transferred across an air-gap boundary via your normal approved channel).

This follows the [Portable Load Testing spec](https://mattermost.atlassian.net/wiki/spaces/XYZ/pages/4675403782/Portable+Load+Testing+-+Spec+Document). It does **not** cover deploying Mattermost/Prometheus/Grafana themselves: see `deployment/azure/DEPLOYMENT.md` for a full reference deployment (built and load-tested end to end in a non-air-gapped Azure subscription) if you need to stand those up too.

---

## 1. Build the Docker image

Write these contents to a file called `Dockerfile.portable`:

```Dockerfile
FROM debian:bookworm-slim
ARG LTNG_VERSION=1.32.0
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl \
    && curl -fsSL -o /tmp/ltng.tar.gz \
      "https://github.com/mattermost/mattermost-load-test-ng/releases/download/v${LTNG_VERSION}/mattermost-load-test-ng-v${LTNG_VERSION}-linux-amd64.tar.gz" \
    && tar -xzf /tmp/ltng.tar.gz -C /tmp \
    && cp /tmp/mattermost-load-test-ng-v${LTNG_VERSION}-linux-amd64/bin/ltapi /usr/local/bin/ltapi \
    && rm -rf /tmp/ltng.tar.gz /tmp/mattermost-load-test-ng-v${LTNG_VERSION}-linux-amd64 \
    && apt-get purge -y curl && apt-get autoremove -y && rm -rf /var/lib/apt/lists/*
EXPOSE 4000
CMD ["ltapi"]
```

On the build machine, in the directory containing the `Dockerfile.portable` file, build the image:

```shell
docker build -f Dockerfile.portable -t ltapi:latest .
```

This downloads the `ltapi` binary from a tagged GitHub release (the default version is pinned in the Dockerfile's `LTNG_VERSION` build arg), and produces a self-contained image containing only `ltapi`. To pick up a different release, pass `--build-arg LTNG_VERSION=x.y.z`.

## 2. Transfer the image to the agent VM

Export the image and move it across whatever approved channel the air-gap boundary requires. Here we're assuming we can use `scp`:

```shell
docker save ltapi:latest | gzip > ltapi-image.tar.gz
scp ltapi-image.tar.gz <you>@<agent-vm-ip>:~/
```

On the agent VM, before running the container, tune the OS (these are the same tunings done by `ltctl` automatically on AWS to sustain 2000 concurrent connections):

`/etc/security/limits.conf`:
```
* soft nofile 100000
* hard nofile 100000
```

`/etc/sysctl.conf` (then `sysctl -p`):
```
net.ipv4.ip_local_port_range = 1025 65000
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_rmem = 4096 156250 625000
net.ipv4.tcp_wmem = 4096 156250 625000
net.core.rmem_max = 312500
net.core.wmem_max = 312500
net.core.rmem_default = 312500
net.core.wmem_default = 312500
net.ipv4.tcp_mem = 1638400 1638400 1638400
```

Then, load and run the iamge:

```shell
docker load -i ltapi-image.tar.gz

docker run -d --name ltapi --restart always \
  --ulimit nofile=100000:100000 \
  -p 4000:4000 \
  ltapi:latest ltapi
```

`--restart always` keeps it running across VM reboots. Confirm it's up: `docker ps` should show `ltapi` listening on `4000`.

---

## 3. Verify the network is ready

Only the following ports need to be open. Every rule below is scoped to a specific source (a subnet, or your own IP):

| # | Source        | Destination | Port   | Protocol    | Purpose                                                   |
|---|---------------|-------------|--------|-------------|-----------------------------------------------------------|
| 1 | Your laptop   | Agent VM    | `4000` | TCP/HTTP    | `ltcoordinator` controls `ltapi` (ramp users, start/stop) |
| 2 | Agent VM      | Mattermost  | `8065` | TCP/HTTP+WS | Simulated user traffic (API + WebSocket)                  |
| 3 | Prometheus VM | Agent VM    | `4000` | TCP/HTTP    | Prometheus scrapes `ltapi`'s `/metrics`                   |
| 4 | Prometheus VM | Mattermost  | `8067` | TCP/HTTP    | Prometheus scrapes Mattermost's own metrics               |

### Azure NSG configuration

This section explains how I did this in Azure, but it may need some changes in Crimson-1's infrastructure.

Each of these VMs sits on some subnet with an NSG attached (one NSG per subnet is the recommended pattern). Add inbound rules on the **agent VM's NSG**:

```shell
# Row 1: your laptop -> agent VM :4000
MY_IP=$(curl -s https://api.ipify.org)
az network nsg rule create \
  --resource-group <rg> --nsg-name <agent-vm-nsg> \
  --name Allow-Operator-4000 --priority 100 \
  --source-address-prefixes "${MY_IP}/32" --destination-port-ranges 4000 \
  --access Allow --protocol Tcp --direction Inbound

# Row 3: Prometheus subnet -> agent VM :4000
az network nsg rule create \
  --resource-group <rg> --nsg-name <agent-vm-nsg> \
  --name Allow-Metrics-4000 --priority 110 \
  --source-address-prefixes <prometheus-subnet-cidr> --destination-port-ranges 4000 \
  --access Allow --protocol Tcp --direction Inbound
```

And on the **Mattermost NSG**:

```shell
# Row 2: agent VM subnet -> Mattermost :8065
az network nsg rule create \
  --resource-group <rg> --nsg-name <mattermost-nsg> \
  --name Allow-LoadTest-8065 --priority 120 \
  --source-address-prefixes <agent-vm-subnet-cidr> --destination-port-ranges 8065 \
  --access Allow --protocol Tcp --direction Inbound

# Row 4: Prometheus subnet -> Mattermost :8067
az network nsg rule create \
  --resource-group <rg> --nsg-name <mattermost-nsg> \
  --name Allow-Metrics-8067 --priority 130 \
  --source-address-prefixes <prometheus-subnet-cidr> --destination-port-ranges 8067 \
  --access Allow --protocol Tcp --direction Inbound
```

**Gotcha to check for**: `az vm create` attaches its own per-NIC NSG (SSH-only) *in addition to* the subnet NSG by default, unless it was created with `--nsg ""`. Azure enforces both, so if traffic is still blocked after the subnet rules above look correct, check the VM's NIC directly:

```shell
az network nic show --ids <agent-vm-nic-id> --query networkSecurityGroup
```

If that returns a second NSG (not the subnet one), either add the same rules there too or detach it: `az network nic update --ids <agent-vm-nic-id> --remove networkSecurityGroup`.

### Confirm each hop actually works

Run these before starting the load test, from each source machine:

```shell
# From your laptop (row 1)
curl http://<agent-vm-ip>:4000/loadagent/

# From the agent VM (row 2) - confirm it can reach Mattermost on plain HTTP, not just 443
curl -i http://<mattermost-ip-or-internal-lb>:8065/api/v4/system/ping

# From the Prometheus VM (rows 3 and 4)
curl http://<agent-vm-ip>:4000/metrics
curl http://<mattermost-ip>:8067/metrics
```

All four should return a response, not a timeout or a connection-refused error. If row 3 or 4 fail but Grafana already shows other scrape targets healthy, double check the specific source subnet/IP in the NSG rule.

### Mattermost-side settings

Confirm these are set on the Mattermost server, so that the agents can create new users and Prometheus can scrape metrics:

| Setting                         | Value   |
|---------------------------------|---------|
| `TeamSettings.EnableOpenServer` | `true`  |
| `MetricsSettings.Enable`        | `true`  |
| `MetricsSettings.ListenAddress` | `:8067` |

**OS tuning on the Mattermost machine(s)**. These tunings are usually done automatically by `ltctl` on AWS deployments (see `deployment/terraform/strings.go`'s `serverSysctlConfig`), but need to be done manually for Crimson-1's infra:

`/etc/security/limits.conf`:
```
* soft nofile 100000
* hard nofile 100000
* soft nproc 8192
* hard nproc 8192
```

`/etc/sysctl.conf` (then `sysctl -p`):
```
net.ipv4.ip_local_port_range = 1025 65000
net.ipv4.tcp_fin_timeout = 30
net.ipv4.tcp_tw_reuse = 1
net.core.somaxconn = 4096
net.ipv4.tcp_max_syn_backlog = 8192
vm.min_free_kbytes = 167772
net.ipv4.tcp_slow_start_after_idle = 0
net.ipv4.tcp_congestion_control = bbr
net.core.default_qdisc = fq
net.ipv4.tcp_notsent_lowat = 16384
net.ipv4.tcp_rmem = 4096 156250 2500000
net.ipv4.tcp_wmem = 4096 156250 2500000
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
```

The `mattermost.service` systemd unit should set `LimitNOFILE=49152` (then `systemctl daemon-reload && systemctl restart mattermost` to pick it up).

---

## 4. Run the load test from your laptop

Everything below runs on **your laptop**, not the agent VM: `ltcoordinator` only needs to reach the agent VM on port 4000 (row 1 above); it never talks to Mattermost directly.

1. Build `ltcoordinator` locally (or reuse the binary from a prior build):
   ```shell
   go build -o bin/ltcoordinator ./cmd/ltcoordinator
   ```
2. Copy the sample configs and point them at your environment:
   ```shell
   cp config/config.sample.json config.json
   cp config/coordinator.sample.json coordinator.json
   ```
   - `config.json`: set `ConnectionConfiguration.ServerURL`/`WebSocketURL` to Mattermost's `http://...:8065` / `ws://...:8065` address, and the admin credentials.
   - `coordinator.json`: set `ClusterConfig.Agents[0].ApiURL` to `http://<agent-vm-ip>:4000`, and `MonitorConfig.PrometheusURL` to `http://<prometheus-ip>:9090` (if Prometheus is reachable from your laptop too; otherwise leave the monitor block out, it's optional for driving the test itself). Set `ClusterConfig.MaxActiveUsers` to your target user count.
3. Start the test:
   ```shell
   ./bin/ltcoordinator -c coordinator.json -l config.json
   ```
   It ramps users up on stdout (`active_users=...`, `errors=...`) and reports Prometheus-based alerts if `MonitorConfig` is configured. Watch Grafana in parallel for CPU/latency/error-rate on the Mattermost and agent-VM dashboards.
4. Stop with `Ctrl-C` (`SIGINT`): this cleanly tears down the load-testers on `ltapi` and exits. Confirm the teardown completed:
   ```shell
   curl http://<agent-vm-ip>:4000/loadagent/lt0/status
   # expect: {"error":"load-test agent with id lt0 not found"}
   ```
   If a prior run was killed uncleanly (e.g. `kill -9`, not `Ctrl-C`) and a second run fails immediately with `LoadTester has not stopped`, clear the stale state first:
   ```shell
   curl -X POST http://<agent-vm-ip>:4000/loadagent/lt0/stop
   curl -X DELETE http://<agent-vm-ip>:4000/loadagent/lt0
   ```

That's the full loop: build image, transfer, run on agent VM, verify the four network hops, then drive the test from `ltcoordinator` on your laptop.
