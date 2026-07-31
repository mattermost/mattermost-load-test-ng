# Azure deployment record - Mattermost HA + metrics + portable load-test

This document records what was actually built, in what order, and why - including every bug
hit along the way and how it was fixed. It supersedes the plan in spirit (the plan was written
before several constraints were discovered); this is the ground truth of what's running.

Scripts live in `deployment/azure/*.sh`, cloud-init/config assets in `deployment/azure/assets/`,
generated secrets in `secrets/` (gitignored), scratch state in `deployment/azure/.state/`
(gitignored). The portable load-test image is `Dockerfile.portable` + `.dockerignore` at the
repo root (see [Switching ltapi to the portable Docker image](#switching-ltapi-to-the-portable-docker-image-mimicking-crimson-1)).

Resource group: **`rg-mm-loadtest-ha`**, region **`westus2`** (see [Region and VM sizing](#region-and-vm-sizing-the-quota-wall) for why not `eastus`).

---

## Architecture

```
   Operator's machine (this dev machine)
   +--------------------------+
   | ltcoordinator            |
   +----+-----------------+---+
        | 4000            | 9090
        | (control)       | (Prometheus queries)
        v                 v
                                   Internet
                                      |
                     443 (TLS)  |         | 443 (TLS)
                                 v         v
                        +-----------+   +----+------+
                        | mm-proxy  |   | mm-metrics|<---------+
                        | nginx     |   | Prometheus|          |
                        | +certbot  |   | Loki      |          |
                        +-----+-----+   | Grafana   |          |
                              |         | +nginx    |          |
                        8065  |         | +certbot  |          |
                  (loadtest   |         +-----+-----+          |
                   only, NSG- |               ^                |
                   restricted)|               | scrape (9090/8067/9100/4000)
                              v               | push (3100, otelcol)
                 +------------+------------+  |                |
                 |            |            |  |                |
           +-----v-----+ +----v------+     |  |                |
           | mm-app-1  | | mm-app-2  |-----+--+                |
           | Mattermost| | Mattermost|     |                   |
           +-----+-----+ +-----+-----+     |                   |
                 |             |           |                   |
                 +------+------+           |                   |
                        |                  |                   |
                        v                  |                   |
              Postgres Flexible Server     |                   |
              (mm-loadtest-pg, private)    |                   |
                        ^                  |                   |
                        |                  |                   |
                 +------+------+           |                   |
                 | mm-loadtest |-----------+-------------------+
                 | ltapi (Docker, port 4000)
                 | ltagent (native binary, unused - DB dump loaded instead)
                 +-------------+
```

Note: `ltcoordinator` forwards Mattermost connection details to `ltapi` over the 4000
control channel - it never talks to Mattermost directly itself (see
[Moving ltcoordinator to the operator's machine](#moving-ltcoordinator-to-the-operators-machine)).

All 5 VMs sit in one VNet, one subnet per tier, one NSG per subnet. Every VM also has its own
public IP for SSH (see [NSG design](#nsg-design) for why, and the NIC-level-NSG bug that nearly
broke this).

---

## Network configuration

### VNet and subnets

VNet **`vnet-mm-loadtest`**, address space `10.20.0.0/16`, 5 subnets:

| Subnet | CIDR | Purpose | Delegation |
|---|---|---|---|
| `snet-proxy` | `10.20.1.0/24` | nginx proxy | - |
| `snet-app` | `10.20.2.0/24` | Mattermost app nodes | service endpoint `Microsoft.Storage` (for Azure Files) |
| `snet-db` | `10.20.3.0/24` | Postgres Flexible Server | `Microsoft.DBforPostgreSQL/flexibleServers` |
| `snet-metrics` | `10.20.4.0/24` | Prometheus/Loki/Grafana | - |
| `snet-loadtest` | `10.20.5.0/24` | ltapi/ltcoordinator | - |

### NSG design

One NSG per subnet (`nsg-proxy`, `nsg-app`, `nsg-db`, `nsg-metrics`, `nsg-loadtest`), built to
allow only the minimum traffic each tier actually needs. **SSH (22) is restricted to the
operator's own detected public IP** (`curl https://api.ipify.org` at deploy time) on every NSG -
every VM has a public IP for admin access, but only from that one address.

| NSG | Rule | Source -> Port | Why |
|---|---|---|---|
| `nsg-proxy` | Allow-SSH-Operator | `<my-ip>/32` -> 22 | admin access |
| | Allow-HTTP-Internet | `Internet` -> 80 | Let's Encrypt HTTP-01 + redirect |
| | Allow-HTTPS-Internet | `Internet` -> 443 | public Mattermost |
| | Allow-LoadTest-8065 | `snet-loadtest` -> 8065 | load-test traffic (plain HTTP, never public - see below) |
| | Allow-NodeExporter-Metrics | `snet-metrics` -> 9100 | Prometheus node_exporter scrape |
| `nsg-app` | Allow-SSH-Operator | `<my-ip>/32` -> 22 | admin access |
| | Allow-MM-FromProxy | `snet-proxy` -> 8065 | proxy -> app traffic only (never public, never from loadtest directly) |
| | Allow-Metrics-FromMetricsVM | `snet-metrics` -> 8067, 9100 | Mattermost metrics + node_exporter scrape |
| | Allow-Gossip-Intracluster | `snet-app` (self) -> 8074, all protocols | Mattermost HA gossip between app-1/app-2 |
| `nsg-db` | Allow-Postgres-FromApp | `snet-app` -> 5432 | only app nodes talk to the DB |
| `nsg-metrics` | Allow-SSH-Operator | `<my-ip>/32` -> 22 | admin access |
| | Allow-HTTP/HTTPS-Internet | `Internet` -> 80/443 | public Grafana |
| | Allow-Loki-Push | `snet-app`+`snet-proxy`+`snet-loadtest` -> 3100 | otelcol-contrib log shipping from all 3 node types |
| | Allow-Prometheus-FromLoadtest | `snet-loadtest` -> 9090 | (historical) `ltcoordinator`'s `MonitorConfig.PrometheusURL` when co-located on `mm-loadtest` |
| | Allow-Operator-Prometheus | `<my-ip>/32` -> 9090 | `ltcoordinator`'s `MonitorConfig.PrometheusURL`, now that it runs on the operator's own machine - see [Moving ltcoordinator to the operator's machine](#moving-ltcoordinator-to-the-operators-machine) |
| `nsg-loadtest` | Allow-SSH-Operator | `<my-ip>/32` -> 22 | admin access |
| | Allow-Metrics-FromMetricsVM | `snet-metrics` -> 4000, 9100 | Prometheus scrapes `ltapi` + node_exporter |
| | Allow-Operator-4000 | `<my-ip>/32` -> 4000 | operator's `ltcoordinator` controls `ltapi` directly |

**Why the load-test VM reaches Mattermost on plain-HTTP 8065 instead of the public 443/TLS
path:** the live Confluence spec (not the stale local draft that was initially checked) lists the
agent->Mattermost hop as port 8065/HTTP+WS. nginx on the proxy has **two server blocks**: one on
443 with a real Let's Encrypt cert for the public internet, and a second plain-HTTP listener on
8065 load-balancing across both app nodes - NSG-restricted to `snet-loadtest` only, never exposed
to `Internet`. Load-test traffic still goes through the proxy (same load-balancing as real
traffic), it just skips the TLS handshake.

### DNS

Both public-facing VMs got Azure-managed DNS labels (globally unique, publicly resolvable out of
the box - no custom domain needed):
- Proxy: `mm-lt-proxy-udln2nkd.westus2.cloudapp.azure.com` -> `20.230.210.126`
- Metrics: `mm-lt-grafana-udln2nkd.westus2.cloudapp.azure.com` -> `20.236.48.66`

These are computed *deterministically* in `lib/common.sh` (`proxy_fqdn()` / `metrics_fqdn()`,
based on a cached random suffix) so scripts that run *before* those VMs exist - e.g. the app
nodes need `SiteURL` set to the proxy's future HTTPS URL - can still reference the right FQDN.

### Current IPs (for reference - will change if VMs are recreated)

| VM | Public IP | Private IP |
|---|---|---|
| mm-app-1 | 20.57.158.176 | 10.20.2.4 |
| mm-app-2 | 20.112.86.49 | 10.20.2.5 |
| mm-proxy | 20.236.10.155 | 10.20.1.4 |
| mm-metrics | 20.109.174.15 | 10.20.4.4 |
| mm-loadtest | 20.236.60.44 | 10.20.5.4 |

### Network bugs hit and fixed

1. **ARM read-after-write race.** Creating a subnet immediately after the VNet (or an NSG rule
   immediately after the NSG) intermittently 404'd - `ResourceNotFound` - even though the parent
   resource's own create call had already returned success, and even after explicitly polling
   `az network vnet wait --created` on the same resource. This is a cross-replica ARM consistency
   lag, not a provisioning-state issue. Fixed with a generic `retry()` bash helper (exponential
   backoff, 6 attempts) in `lib/common.sh`, wrapped around every dependent `az network ...`
   create call in `00-network.sh`, `01-database.sh`, `02-storage.sh`.

2. **`az vm create` silently attaches a redundant per-NIC NSG.** By default, `az vm create`
   creates its *own* NSG (SSH-only) and attaches it directly to the VM's NIC, **in addition to**
   the subnet-level NSG. Azure requires traffic to pass **both** - so this auto-created,
   SSH-only NSG silently blocked everything else (public 443/80 to the proxy, proxy->app-node
   8065, metrics->app-node 8067/9100, etc.) despite the subnet NSG explicitly allowing it. This
   broke the *entire* cluster's inter-VM traffic, not just the public endpoint. Fixed by:
   - Adding `--nsg ""` to every `az vm create` call (disables the auto-create), and
   - For the VMs already created before the fix (`mm-app-1`, `mm-app-2`), detaching the
     redundant NSG from the existing NIC: `az network nic update -n <nic> --remove
     networkSecurityGroup` (the proxy VM was recreated fresh at a smaller size anyway, so it
     picked up the fix automatically).

3. **Postgres Flexible Server `--private-dns-zone` validation is broken for new zones.**
   `az postgres flexible-server create --private-dns-zone <any-name>.private.postgres.database.azure.com`
   rejected *every* name tried (including ones bearing no resemblance to the server name) with
   `(PrivateDnsZoneNameNotValid) ... can not be server name plus zone suffix` - a misleading
   error for what's actually a broken auto-create code path. Worked around by pre-creating the
   private DNS zone and its VNet link ourselves (`az network private-dns zone create` +
   `az network private-dns link vnet create`) and pointing `--private-dns-zone` at the
   already-existing zone, which bypasses that path entirely.

---

## Region and VM sizing (the quota wall)

Originally targeted **East US** per the user's initial preference, sized for 2,000 concurrent
users per the Confluence spec (app nodes `Standard_D4s_v5` x2, proxy `Standard_D2s_v5`, metrics
`Standard_D4s_v5`, load-test `Standard_D8s_v5`).

- **eastus** had zero Postgres Flexible Server SKU availability for this subscription
  (`az postgres flexible-server list-skus -l eastus` returned nothing) and the entire `Dsv5` VM
  family was `NotAvailableForSubscription` at the *location* level (not just a specific zone).
  -> Moved to **westus2**, which had both available, and bumped VM sizes from `v5` to `v6`
  (`Standard_D2s_v6`/`D4s_v6`/`D8s_v6`) since `v5` was blocked there too.
- Even after that, **`az vm create` for the metrics VM failed with `QuotaExceeded`**:
  `Total Regional Cores quota. Current Limit: 10, Current Usage: 10`. This is a **hard
  subscription-wide cap of 10 vCPUs total across every VM series combined** in westus2 - already
  fully consumed by just the 2 app nodes (4 vCPU each) + proxy (2 vCPU) = 10.
  - A self-service `az quota update` request for `standardDFamily` -> 24 was attempted and
    silently failed (still capped at 10 after waiting).
  - Given the user's explicit choice (downsize now vs. wait on a portal-based quota increase),
    everything was resized to fit in 10 vCPUs total:

| VM | Original plan | **Actual (deployed)** |
|---|---|---|
| mm-app-1, mm-app-2 | `Standard_D4s_v5` (4 vCPU) each | **`Standard_D2s_v6`** (2 vCPU) each |
| mm-proxy | `Standard_D2s_v5` (2 vCPU) | **`Standard_B1ms`** (1 vCPU) |
| mm-metrics | `Standard_D4s_v5` (4 vCPU) | **`Standard_B1ms`** (1 vCPU) |
| mm-loadtest | `Standard_D8s_v5` (8 vCPU) | **`Standard_B4ms`** (4 vCPU) |
| **Total** | 22 vCPU | **10 vCPU** |

  - Resizing `D-series -> D-series` (app nodes) was a simple `az vm deallocate` + `az vm resize`
    + `az vm start`. Resizing `D-series -> B-series` (proxy) **failed** with
    `(InvalidParameter) The VM size 'Standard_B1ms' cannot boot with DiskControllerType 'NVMe'`
    - B-series doesn't support the NVMe disk controller the D-series VM was provisioned with.
    That VM had to be deleted (VM + its orphaned NIC + OS disk) and recreated fresh at the
    smaller size instead of resized in place.

**Caveat this leaves**: `Standard_B*` sizes are *burstable* (CPU-credit-based), not the
consistent-performance D-series. This deployment is fully functional end-to-end, but at this
sizing it cannot genuinely simulate 2,000 concurrent users - realistically a few hundred at most
before the proxy/metrics VMs' CPU credits and the app nodes' 2 vCPUs become the bottleneck rather
than Mattermost itself. A real quota increase (portal-based, not instant via CLI) would be
needed to reach the originally-planned scale.

---

## Compute and storage

- **Database**: Azure Database for PostgreSQL Flexible Server `mm-loadtest-pg`, `Standard_D4ds_v5`,
  128 GB, Postgres 16, VNet-integrated on `snet-db` (private access only, no public network
  access). Private DNS zone `testzone123.private.postgres.database.azure.com` (see bug #3 above
  for why this specific, otherwise-arbitrary name).
- **Storage**: storage account `mmlt<random-suffix>`, SMB file share `mattermost-data` (256 GB
  quota), mounted at `/opt/mattermost/data` on both app nodes via `/etc/fstab` (cifs). Network
  rules: `--default-action Deny` + a VNet rule scoped to `snet-app` only - no public network
  access. The file share itself is created via `az storage share-rm create` (ARM control-plane),
  not `az storage share create` (data-plane), specifically because the latter would need to hit
  the storage account's public data-plane endpoint directly, which the network rule blocks.

---

## Mattermost HA app nodes

- Mattermost Enterprise **v11.9.0** (latest stable at deploy time), downloaded from
  `releases.mattermost.com`, installed to `/opt/mattermost`.
- Identical `config.json` on both nodes, patched from the shipped default via `jq`:
  `SqlSettings` -> Postgres DSN; `ClusterSettings.Enable=true`, `GossipPort=8074`,
  `EnableExperimentalGossipEncryption=true`; `MetricsSettings.Enable=true`;
  `FileSettings.Directory=/opt/mattermost/data`; `ServiceSettings.SiteURL` -> the proxy's HTTPS
  FQDN; `TeamSettings.EnableOpenServer=true`.
- **Bug hit**: forgot `ClusterSettings.ReadOnlyConfig=false`. This defaults to `true`, and with
  HA enabled that makes the config read-only - which broke a startup migration
  (`Advanced Permissions Migration`) needing to persist a config change, crash-looping
  `mattermost.service` on both nodes (`configuration is read-only`). Fixed by explicitly setting
  it to `false` (both live, via SSH+jq+restart, and in the source template).
- systemd unit `mattermost.service`: `LimitNOFILE=49152`, standard feature-flag env vars, and
  `Environment=MM_SERVICEENVIRONMENT=test` - see [Licensing](#licensing) for why this matters.
- Server-side sysctl tuning (`serverSysctlConfig` - BBR congestion control, larger TCP buffers,
  bumped `somaxconn`/backlog) + `nofile`/`nproc` limits, lifted from
  `deployment/terraform/strings.go` (the existing AWS Terraform deployment's tuning values).

### Sysadmin account

There have been **two different sysadmin accounts** over the course of this session - worth
being precise about which is real:
1. `sysadmin@loadtest.mattermost.com` - created via the public signup API early on (the
   Mattermost server binary has no `user create` CLI subcommand; on a fresh instance the first
   signup auto-promotes to System Admin). **This account no longer exists** - it lived in the DB
   that was later dropped and recreated for the dump restore.
2. **`sysadmin` / `Sys@dmin-sample1` (`sysadmin@sample.mattermost.com`)** - the account actually
   in use now, built into the `12M_610_fixed_psql.sql.gz` dump itself. `secrets/mm_admin_password`
   and `secrets/mm_admin_note.txt` reflect this.

---

## Licensing

Uploading the license (`/home/alejandro/credentials/licenses/loadtest-dev-license-500K.mattermost-license`)
initially failed with `(api.license.add_license.invalid.app_error) Invalid license file`, even
when uploading the original file directly (byte-identical checksums confirmed - ruled out any
transfer/encoding bug). **Fix (per the user): set `MM_SERVICEENVIRONMENT=test` in the
Mattermost systemd unit's environment, then reboot.** After that, the exact same license file
uploaded successfully:

```json
{"sku_name":"Mattermost Enterprise Advanced","sku_short_name":"advanced",
 "expires_at":"1860469200000", "features":{"cluster":true,"metrics":true,...}}
```

Sequence that got both nodes fully licensed:
1. Set `MM_SERVICEENVIRONMENT=test` on both app nodes, `daemon-reload`.
2. `az vm restart` both app nodes.
3. Re-uploaded the license via the API - it landed on whichever node the proxy's load balancer
   happened to route the request to (app-2, in this case) and set it in the shared DB.
4. `mattermost.service` restarted on app-1 to pick up the license from the DB (each node reads
   license state at startup; it doesn't propagate live without an already-working cluster -
   chicken-and-egg, since cluster mode itself needs the license).
5. Both nodes then genuinely joined the HA gossip cluster (confirmed via
   `GET /api/v4/cluster/status` showing both `mm-app-1` and `mm-app-2`, with app-1 elected
   leader) and `/metrics` (Mattermost's own Prometheus endpoint, license-gated) started
   returning 200 instead of 404 on both.

---

## Metrics stack (mm-metrics)

- **Prometheus**: apt package, scrape config templated with explicit `instance` labels
  (`app-0`, `app-1`, `proxy`, `loadtest-0`, `metrics`) matching what
  `config/coordinator.sample.json`'s built-in alert queries expect
  (`instance=~"app.*"` etc.), so those queries stay usable if wired into `ltcoordinator`.
  Jobs: `node` (node_exporter on all 5 VMs), `mattermost` (app nodes' `:8067`), `loadtest`
  (`ltapi`'s `:4000`), `prometheus` (self).
- **Loki**: packaged `.deb` (v3.2.0), default config, `:3100`. Receives logs via `otelcol-contrib`
  on app nodes (journald), proxy (nginx access/error logs), and the load-test VM (`ltagent.log`/
  `ltcoordinator.log`), all pushing to Loki's OTLP ingest endpoint.
- **Grafana**: v11.6.1, Prometheus + Loki datasources pre-provisioned. **Differs from the AWS
  Terraform default**: anonymous access disabled (that default enables anonymous viewer access,
  not appropriate for a public internet-facing instance) - real `admin` login with a generated
  password.
  - **Bug hit**: `grafana-cli admin reset-admin-password` runs its own embedded migration runner
    against the same sqlite file as the systemd-managed `grafana-server` process. Running it
    concurrently with (or right after starting) `grafana-server` raced both processes against
    the same DB file, corrupting the migration (`no such column: "def_org_id"`) and - because
    the script used `set -e` - silently aborting the rest of provisioning (nginx/certbot never
    ran). Fixed by resetting the password with the server **stopped**, then starting it
    afterward.
- **Known gap**: the AWS deployment's default Grafana dashboards
  (`default_dashboard_tmpl.json`, `coordinator_dashboard_tmpl.json`) were **not** imported -
  they're Go `text/template`-generated at deploy time in `deployment/terraform/metrics.go`
  (including a backtick-escaping trick to emit literal Grafana templating syntax), not flat
  files, and weren't worth reverse-engineering in bash for this. Datasources work fine; no
  pre-built dashboard JSON.
- Public exposure: nginx + certbot in front of Grafana (443 -> `localhost:3000`), same pattern as
  the Mattermost proxy. Prometheus/Loki are never exposed publicly.

---

## Load-test VM (mm-loadtest)

- Originally built `ltapi`/`ltcoordinator`/`ltagent` **from source** (`git clone` this repo,
  `go build`) rather than the spec's Docker-image approach, on the reasoning that this VM has
  normal internet access (only Crimson-1's real air-gapped environment strictly needs the
  portable Docker path) - **this was later replaced for `ltapi` specifically; see
  [Switching ltapi to the portable Docker image](#switching-ltapi-to-the-portable-docker-image-mimicking-crimson-1)
  below.** `ltcoordinator`/`ltagent` are still the natively-built binaries from this step.
- **Bug hit**: `az vm run-command` executes with `HOME` unset, so Go can't compute its default
  `GOPATH`/`GOMODCACHE` and fails outright (`module cache not found`). Fixed by explicitly
  exporting `HOME`/`GOPATH`/`GOMODCACHE` before the `go build` calls.
- **Bug hit (post-provisioning)**: since that same run-command execution runs as root, the
  entire cloned repo ended up owned by `root`, including `bin/`. When later running
  `ltcoordinator` interactively as `mmadmin` over SSH, it failed with
  `can't open new logfile: open ltcoordinator.log: permission denied` (couldn't write into its
  own working directory). Fixed with `sudo chown -R mmadmin:mmadmin mattermost-load-test-ng`.
- `ltcoordinator` was initially run interactively in a `tmux` session over SSH on this VM - later
  moved to the operator's own machine instead, see
  [Moving ltcoordinator to the operator's machine](#moving-ltcoordinator-to-the-operators-machine).
  No systemd unit for it either way - it's meant to be run interactively, not as a daemon.
- Client-side sysctl tuning (`clientSysctlConfig` - extended port range, TCP buffer sizes tuned
  for the load-generator role, distinct from the server-side tuning on app/proxy nodes).
- **Bug hit**: `ltagent init -n 2000` - no such flag. `ltagent init` has no user-count flag at
  all; the count comes from `config.json`'s `UsersConfiguration.MaxActiveUsers`. Moot anyway once
  the DB dump was restored (see below) - `ltagent init`'s job is just creating teams/channels for
  simulcontroller, and the dump already has real ones.

### Switching ltapi to the portable Docker image (mimicking Crimson-1)

The Confluence spec's actual scenario (Crimson-1: a genuinely air-gapped Azure environment) has
no Go toolchain, no `git`, and no internet access on the target VM at all - the *only* way
binaries get there is a Docker image built on an operator machine and carried across the air-gap
boundary. The from-source build above works for our environment (which does have internet) but
doesn't exercise that actual mechanism, so `ltapi` was switched over to it:

1. **`Dockerfile.portable`** added at the repo root, matching the spec's sketch almost verbatim:
   a `golang:1.26` builder stage (matches this repo's `go.mod` toolchain version) building all
   three binaries, copied into a `debian:bookworm-slim` final stage (not Alpine - avoids libc
   issues from any CGo dependency; not distroless - operators need a shell to troubleshoot).
   Exposes `4000`, default `CMD ["ltapi"]`.
   - **Trimmed later (2026-07-31)**: `ltcoordinator` and `ltagent` were dropped from the image.
     Neither was ever actually run from it in this deployment - `ltcoordinator` is built and run
     natively on the operator's own machine (see
     [Moving ltcoordinator to the operator's machine](#moving-ltcoordinator-to-the-operators-machine)
     below), and `ltagent` went unused once the DB dump replaced synthetic seeding (see
     [Test data: DB dump restore](#test-data-db-dump-restore)). The image now builds and ships
     only `ltapi`.
   - **Switched to a GitHub release download (2026-07-31)**: the `golang:1.26` builder stage was
     replaced with a single `debian:bookworm-slim` stage that `curl`s the tagged release tarball
     from `github.com/mattermost/mattermost-load-test-ng/releases` (`ARG LTNG_VERSION`, default
     `1.32.0`) and copies `bin/ltapi` out of it - no Go toolchain needed to build the image at
     all. This works because the release tarball already ships `bin/ltapi` (and `bin/ltagent`,
     unused here) per `.goreleaser.yml`; `ltcoordinator` was never part of the release, which is
     moot now that the image doesn't need it either. Building the image now requires internet
     access on the build machine (unchanged from before - it always did, to reach `golang:1.26`
     or `debian:bookworm-slim` on Docker Hub); the resulting image still needs none at runtime.
2. **`.dockerignore`** added at the repo root - without it, `COPY . .` in the Dockerfile would
   have baked `secrets/` and `deployment/azure/.state/` directly into the image's build context
   and layers.
3. Built **locally** (the operator machine, in the spec's framing) with
   `docker build -f Dockerfile.portable -t ltapi:latest .` - about 100 MB compressed.
4. `docker save ltapi:latest | gzip > ltapi-image.tar.gz`, then `scp` to the load-test VM. This
   step is the actual point of the exercise: in Crimson-1's real environment this would be a
   security-reviewed transfer across the air-gap boundary; here there's no real air gap, so a
   plain `scp` stands in for "approved channel."
5. Installed Docker on the load-test VM (`apt-get install docker.io`, matching the general
   pattern of preferring distro-packaged tools over a third-party repo where the packaged
   version is sufficient - we only need to run containers, not build complex multi-arch images).
6. `docker load -i ltapi-image.tar.gz`, then stopped and disabled the native
   `ltapi.service` from the from-source build (would otherwise fight the container for port
   4000), and started the container with the **exact command from the spec**:
   ```
   docker run -d --name ltapi --restart always \
     --ulimit nofile=100000:100000 -p 4000:4000 ltapi:latest ltapi
   ```
   (`--restart always` added on top of the spec's example so it survives VM reboots/auto-shutdown
   wake cycles, matching the intent of the systemd unit it replaced.)
7. Verified: container listening on `:4000` (`docker ps`), and Prometheus's `loadtest` scrape
   target stayed `up` throughout the switch (same port, no NSG or scrape-config changes needed).

Automated as `deployment/azure/06b-loadtest-docker.sh`, wired into `deploy.sh` right after
`06-loadtest.sh`. `ltagent` stays as the natively-built binary from `06-loadtest.sh` (unused for
now - moot with the DB dump loaded, see below). `ltcoordinator` was *initially* left as the
natively-built binary too, co-located with `ltapi` on the same VM - see the next section for why
that changed.

**Caveat**: this is a *simulation* of the air-gap transfer, not a real one - the load-test VM has
normal outbound internet access, so `docker build` could just as easily have run there directly.
The `scp`-based build-elsewhere-then-transfer flow was kept specifically to mimic Crimson-1's
actual constraint faithfully, per the user's request, not because it's technically required here.

### Moving ltcoordinator to the operator's machine

The spec is explicit that `ltcoordinator` is meant to run "on the operator's laptop or any
machine that can reach the agent VMs on port 4000" - not on the agent VM itself. The initial
setup co-located it with `ltapi` on `mm-loadtest` instead, specifically to avoid exposing port
4000 to the public internet. That's a real, deliberate security tradeoff, not a mistake - but at
the user's explicit request (full fidelity to the Crimson-1 scenario is worth the extra
friction, even where not technically required), `ltcoordinator` was moved back out to the
operator's own machine.

**What actually needs to be reachable from the operator's machine, and what doesn't:**
`ltcoordinator` doesn't connect to Mattermost directly - the coordinator package's own comment
says it plainly (`coordinator/coordinator.go`: "The ltConfig parameter is used to create and
configure load-test agents"). `ltConfig` (the `-l`/`config.json` file, containing
`ConnectionConfiguration.ServerURL` etc.) is *forwarded* to `ltapi` over the network; it's
`ltapi` - still running inside the VNet on `mm-loadtest` - that actually opens connections to
Mattermost and generates traffic. So:
- `config.json`'s `ServerURL`/`WebSocketURL` **stay pointed at the proxy's private IP**
  (`10.20.1.4:8065`) - unchanged, still correct, since `ltapi` (not `ltcoordinator`) is what
  uses them.
- `coordinator.json`'s `ClusterConfig.Agents[].ApiURL` and `MonitorConfig.PrometheusURL` **do**
  need to change, from `http://localhost:4000` / the metrics VM's private IP to the load-test
  VM's and metrics VM's **public** IPs - since `ltcoordinator` itself now runs outside the VNet
  and needs to reach both directly.

**Network changes** (both scoped to the operator's own detected IP, same pattern as the existing
SSH rules - never opened to the public internet):
- `nsg-loadtest`: added `Allow-Operator-4000` (`<my-ip>/32` -> 4000) so the operator's machine can
  send ltapi control commands (ramp users up/down).
- `nsg-metrics`: added `Allow-Operator-Prometheus` (`<my-ip>/32` -> 9090) so `ltcoordinator`'s
  `MonitorConfig` can query Prometheus directly for its built-in alert queries.

**Steps** (automated as `deployment/azure/08-operator-setup.sh`, run locally - not part of
`deploy.sh`'s Azure-provisioning loop, since it operates on the operator's own machine, not on
Azure):
1. Open the two NSG rules above.
2. `go build -o bin/ltcoordinator ./cmd/ltcoordinator` locally.
3. Render operator-side config into `deployment/azure/operator/` (gitignored - `config.json`
   contains the Mattermost admin password): `config.json` unchanged (private proxy IP, as
   above), `coordinator.json` re-rendered with the load-test VM's and metrics VM's public IPs.
4. Verify reachability (`curl` to both ports).

**Verified working**: ran `ltcoordinator` locally for a short smoke test -
`./bin/ltcoordinator -c deployment/azure/operator/coordinator.json -l deployment/azure/operator/config.json`
- successfully drove the remote `ltapi` agent (`cluster: successfully started an agent`),
ramped simulated users up (`active_users=8`, `16`, ... `errors=0`), confirming the full control
path (operator machine -> `ltapi` on `mm-loadtest` -> Mattermost via the proxy) works end to end.

To actually run a test, no SSH/VM hop needed - just run locally from the repo root:
```
./bin/ltcoordinator -c deployment/azure/operator/coordinator.json -l deployment/azure/operator/config.json
```

---

## Test data: DB dump restore

Per the user's direction mid-session, skipped `ltagent init`'s synthetic team/channel seeding in
favor of restoring a real dataset: `https://lt-public-data.s3.amazonaws.com/12M_610_fixed_psql.sql.gz`
(~1.6 GB compressed).

Procedure (confirmed with the user first, since `DROP DATABASE` is destructive):
1. Stopped `mattermost.service` on both app nodes.
2. `DROP DATABASE mattermost; CREATE DATABASE mattermost;` on the Flexible Server.
3. Streamed the restore directly - `curl -sL <url> | gunzip | psql ... -v ON_ERROR_STOP=0` - from
   app-1, avoiding ever storing the ~10+ GB uncompressed SQL on disk.
4. The only errors were benign: `role "rdsadmin"/"mmuser" does not exist` (ownership-transfer
   statements referencing roles from the dump's original AWS RDS source, which don't apply to our
   `mmdbadmin`-owned Azure database - the actual schema/data/indexes all loaded fine as
   `mmdbadmin`).
5. Verified: **11,841,269 posts, 16,088 users, 2 teams**.
6. Restarted `mattermost.service` on app-1 first (watched it migrate the dump's older schema up
   to v11.9.0 cleanly), then app-2.
7. **Follow-on issue**: restoring the dump wiped the sysadmin account created in step 3 of the
   original deploy (it lived in the DB that got dropped). Discovered the dump ships its own
   sysadmin account (`sysadmin` / `Sys@dmin-sample1`) - see [Sysadmin account](#sysadmin-account)
   above. Also hit Mattermost's unlicensed "safe user limit" blocking new signups once the DB
   already had 16k+ users, which would have blocked creating a replacement account anyway (moot
   once the real license activated).
8. Regenerated the load-test VM's `config.json` with the dump's real admin credentials (a
   `jq`-based patch attempt failed silently since `jq` isn't installed on that VM - caught before
   it caused confusion, file was regenerated from `assets/config.json.tmpl`).

---

## Verification: running an actual load test

Once `ltcoordinator` was moved to the operator's machine (see above), it was run for real -
not just the earlier connectivity smoke test - from the repo root:

```
./bin/ltcoordinator -c deployment/azure/operator/coordinator.json -l deployment/azure/operator/config.json
```

**Bug hit**: the very first real run failed immediately with
`cluster: failed to start agent: id: lt0, agent: load-test agent api request error: LoadTester
has not stopped`. The earlier smoke test (killed via `timeout`, i.e. SIGTERM) hadn't cleanly told
`ltapi` to stop/destroy its in-progress load-tester before the coordinator process died, leaving
`ltapi` with a stale "running" load-tester (confirmed via
`GET http://<loadtest-ip>:4000/loadagent/lt0/status`: `State: running, NumUsers: 24,
NumErrors: 2183` - all left over from the smoke test). `ltapi`'s own REST API (see
`api/server.go`) exposes exactly what's needed to clear this without touching the VM:
```
POST /loadagent/lt0/stop      # stops the load-tester
DELETE /loadagent/lt0         # destroys it, freeing the id for a new run
```
After that, `ltcoordinator` started cleanly.

**The run**: `coordinator.json`'s `ClusterConfig.MaxActiveUsers` was edited down from 2000 to
**200** for this test (directly in `deployment/azure/operator/coordinator.json`, not through a
script - a manual, deliberate override for this specific run). Users ramped cleanly
(`NumUsersInc: 8` every 2s) up to the 200 cap with only a handful of errors along the way. Once
flat at 200, however, **the error count kept climbing steadily even with no new users being
added** - roughly 9-15 new errors every ~1.7s status tick, climbing from ~280 to 950+ errors
over about 90 seconds before the test was stopped. This is ongoing failure among *already*
connected simulated users, not ramp-up noise, and is consistent with the
[downsized VM sizing](#region-and-vm-sizing-the-quota-wall) caveat already documented - at just
200 concurrent users, the 2-vCPU app nodes appear to already be under real strain. This wasn't
investigated further (stopped at the user's request before digging into which resource -
app-node CPU, DB connections, request latency - was actually saturating).

Stopped with `SIGINT` (`Ctrl+C`, or `pkill -SIGINT -f "bin/ltcoordinator"` when running
detached) - `ltcoordinator` logs `coordinator: shutting down` / `monitor: stop` and cleanly
tears down the remote load-tester (confirmed via the same status endpoint returning
`"error":"load-test agent with id lt0 not found"` afterward). No systemd unit, no Docker
container for `ltcoordinator` - it really is just a foreground process on the operator's
machine, exactly as the spec frames it.

---

## Lifecycle

- Auto-shutdown configured on all 5 VMs, 03:00 UTC daily (`az vm auto-shutdown`).
- `deployment/azure/teardown.sh`: prompts for the resource group name as a confirmation, then a
  single `az group delete --name rg-mm-loadtest-ha --yes` tears down every resource above
  (VMs, VNet/NSGs, Postgres Flexible Server, storage account - everything lives in this one RG).

## Teardown

Ran `deployment/azure/teardown.sh` and confirmed the resource group name to proceed. Deletion
completed at **2026-07-27 16:40 UTC** (`az group exists -n rg-mm-loadtest-ha` returned `false`).
All Azure resources described in this document - the 5 VMs (`mm-app-1`, `mm-app-2`, `mm-proxy`,
`mm-metrics`, `mm-loadtest`), the Postgres Flexible Server, the storage account, and the
VNet/NSGs/public IPs - have been deleted. Nothing from this deployment remains running in Azure.

---

## Redeployment (2026-07-28)

After the teardown above, the environment was rebuilt from scratch via `deploy.sh` +
`08-operator-setup.sh`, then loaded with the real dump and load-tested. Several bugs surfaced
along the way - two of them were regressions of issues already documented earlier in this file,
because the original fix had only been applied live in that session and never actually ported
into the template it was fixing.

### Bugs hit on the redeploy itself

1. **Grafana crash-looped with "unable to open database file: permission denied".** Public
   Grafana URL returned `502 Bad Gateway` - nginx/TLS were fine, but `grafana-server` had
   crash-looped 5 times and systemd gave up (`start-limit-hit`). Root cause: line 45 of
   `provision-metrics.sh.tmpl` runs `sudo grafana-cli admin reset-admin-password` *before*
   `grafana-server` has ever started - since `grafana-cli` runs as root and is the first thing
   to touch `grafana.db`, it creates that file owned by `root`. When `grafana-server` then starts
   as the `grafana` user, it can't open its own database. This is a different failure mode of the
   same underlying race the [original Grafana bug](#metrics-stack-mm-metrics) worked around
   (server-stopped-during-password-reset), just not one that bug's fix anticipated. Fixed live
   (`chown -R grafana:grafana /var/lib/grafana`, `systemctl reset-failed`, restart) and in the
   template (`assets/provision-metrics.sh.tmpl`, chown right after the `grafana-cli` call, before
   `systemctl start grafana-server`).

2. **Load-test VM repo was root-owned again - the [prior fix](#load-test-vm-mm-loadtest) was
   never ported into the template.** `09-restore-dump.sh`'s final step (regenerating and
   `scp`-ing the load-test VM's local `config.json`) failed with `Permission denied`. Same root
   cause as the original bug: `provision-loadtest.sh.tmpl` runs via `az vm run-command` (as
   root), so `git clone` leaves the whole repo root-owned - but unlike the `ReadOnlyConfig` fix
   from the licensing section, the `chown` fix for this one was only ever applied live via SSH in
   the original session, not added to the template. Fixed live again (`sudo chown -R
   mmadmin:mmadmin mattermost-load-test-ng`), added the `chown` to
   `assets/provision-loadtest.sh.tmpl` right after the `git clone`, and made
   `09-restore-dump.sh` itself defensively re-`chown` before every `scp` so it stays robust even
   against older deployments that predate the template fix.

3. **Transient permission errors during `ltagent init`'s synthetic seed step (before the dump
   restore).** 2 of the 50 seed users repeatedly hit `You do not have the appropriate
   permissions.` on `joinAllTeams` (`AddTeamMember`/`GetChannelMembersForUser`) for
   about a minute right after team/channel creation. Reproducing the same calls manually via curl
   moments later succeeded fine, and the stuck controllers self-resolved and reached `user done`
   without intervention - looked like a brief HA cache/gossip propagation lag right after the
   teams were created, not a real permissions bug. `ltagent init complete` printed normally
   afterward; no fix needed.

4. **`TeamSettings.MaxUsersPerTeam` default (50) is far too low for the restored dump.** Hit
   during the first load-test run (see below): the dump packs 16,088 real users into just 2
   teams, but a fresh Mattermost install defaults `MaxUsersPerTeam` to 50 - so every *new*
   simulated user trying to join either team failed with `Unable to create the new team
   membership because the team has reached the limit of members`, and those stuck users then
   threw a second, downstream error (`memstore: channel store is empty` on `switchChannel`, since
   they'd never actually joined a team or loaded its channels). Fixed live via
   `PUT /api/v4/config/patch` (`TeamSettings.MaxUsersPerTeam: 100000`) - took effect immediately,
   no restart needed, and the error rate collapsed within ~30s as the coordinator cycled the
   already-stuck users out and back in. Also added `.TeamSettings.MaxUsersPerTeam=100000` to the
   `jq` patch in `assets/provision-app.sh.tmpl` so a fresh deploy starts with a sane value before
   any dump is even restored.

Also worth noting, not a bug: the `mm-loadtest` VM's `Standard_B4ms` size is burstable, and
`06b-loadtest-docker.sh`'s `apt-get install docker.io` crawled at one point (~33KB/s, confirmed
via `/proc/<pid>/io` on a genuinely-progressing-but-slow fetch, not a hang) - took about 30
minutes end to end instead of the usual under a minute. Consistent with the
[VM-sizing caveat](#region-and-vm-sizing-the-quota-wall) already documented; no fix, just patience.

### `deployment/azure/09-restore-dump.sh`: automating the dump restore

Added a new script that automates the manual procedure from the
["Test data: DB dump restore"](#test-data-db-dump-restore) section above, so it can be re-run
without redoing that whole sequence by hand: stop `mattermost.service` on both app nodes ->
terminate lingering DB connections -> `DROP`/`CREATE DATABASE` -> stream the dump from `mm-app-1`
(same avoid-storing-10GB-on-disk approach as before) -> restart app-1 (watch the schema migrate),
then app-2 -> re-upload the license as the dump's own sysadmin (license state lives in the DB, so
the drop wipes it) -> restart both nodes once more and verify `cluster/status` shows both -> flip
`secrets/mm_admin_password`/`mm_admin_note.txt` and the load-test VM's local `config.json` over to
the dump's credentials. Destructive (`DROP DATABASE`), so it prompts for a typed `restore`
confirmation before touching anything, same pattern as `teardown.sh`. Takes an optional dump-URL
argument to point at a different dataset later if needed.

Ran it against the live redeploy: **11,841,269 posts, 16,088 users, 2 teams** restored, license
re-uploaded, `cluster/status` confirmed both `mm-app-1` and `mm-app-2`, login verified through the
public HTTPS proxy as `sysadmin@sample.mattermost.com` / `Sys@dmin-sample1`.

### Load test run: 500 users, `NumUsersInc: 2`

Ran `ltcoordinator` from the operator's machine against the restored dataset, with
`deployment/azure/operator/coordinator.json`'s `ClusterConfig.MaxActiveUsers` set to **500** and
`NumUsersInc` set to **2** (down from the defaults of 2000/8) - a deliberate, slower/smaller ramp
for this run, edited directly in that file the same way the earlier 200-user run edited it.
Hit the `MaxUsersPerTeam` bug above almost immediately (errors climbing from the first few users
added); fixed live mid-run without stopping the coordinator, and the error rate dropped off
within about 30 seconds as already-stuck users got cycled out and successfully rejoined.

**Stop/restart, and the clean confirmation run.** Stopped that first run with `SIGINT` (clean
shutdown: `coordinator: shutting down` / `monitor: stop`), then explicitly `POST
/loadagent/lt0/stop` + `DELETE /loadagent/lt0` against `ltapi` to fully clear its state (status
went from `stopped` to `"load-test agent with id lt0 not found"`) before starting a second,
identical run (`MaxActiveUsers: 500`, `NumUsersInc: 2`) to confirm the `MaxUsersPerTeam` fix held
under a fresh ramp rather than just mid-flight.

It did: zero new "reached the limit of members" errors this time. The only errors were
`memstore: channel store is empty` on `switchChannel`, and they turned out to be a normal,
one-time-per-user startup race, not a bug - across the whole run, 501 *distinct* users hit that
error exactly once each (essentially every newly-ramped user hits it once, right before their
local channel list finishes loading), and unlike the first run, **zero users had to be removed
and re-added** by the coordinator. Ramped cleanly to the full 500-user target and sat flat there
(`active_users=500 errors=508`, unchanging) for the rest of the run. The recurring `monitor:
error while querying Prometheus ... vector has length = 0` warnings in the log are cosmetic - the
alert queries divide by the 5xx-error rate, which was legitimately zero.

Stopped again with `SIGINT` after running for 30 minutes (per the user's request, timed via a
scheduled follow-up rather than blocking on it). `ltcoordinator`'s own shutdown path had already
torn down the remote load-tester by the time the status was checked (`lt0 not found` immediately,
no separate `stop`/`DELETE` needed this time). **Final stats: ~31.5 min duration, ramped to and
held 500/500 active users, 508 total errors (all accumulated during the initial ramp, flat for
the ~20+ minutes at steady state), 0 users removed** - a clean, healthy run at this scale once the
`MaxUsersPerTeam` config matched the restored dataset.

## Second teardown (2026-07-28)

Ran `deployment/azure/teardown.sh` again and confirmed the resource group name to proceed.
Verified deletion **twice**, from independent angles, after `az group exists` first reported
`false`:

1. `az group show -n rg-mm-loadtest-ha` -> `(ResourceGroupNotFound)`; `az group list` filtered to
   that name -> `[]`.
2. `az vm list -g rg-mm-loadtest-ha` -> `(ResourceGroupNotFound)`; a subscription-wide search for
   any resource (of any type, in any group) whose name contains `mm-loadtest`, `mm-lt`, or `mmlt`
   -> empty.

All 5 VMs (`mm-app-1`, `mm-app-2`, `mm-proxy`, `mm-metrics`, `mm-loadtest`), the Postgres Flexible
Server, the storage account, and the VNet/NSGs/public IPs from this redeployment are gone. Nothing
from this session remains running in Azure.

---

## Third redeployment and timing measurement (2026-07-29)

After the second teardown above, the environment was rebuilt again via `deploy.sh` +
`09-restore-dump.sh` + `08-operator-setup.sh`, this time specifically to measure end-to-end
deployment time. Cached `.state/` values (DNS suffix, storage account name, operator IP) were
still valid and reused - only the Postgres server name needed a real fix, below.

### Bugs hit on this redeploy

1. **Postgres Flexible Server name collided with another user's server in the same shared
   subscription.** First attempt failed immediately at `01-database.sh` with `ERROR: Specified
   server name is already used`, even though `rg-mm-loadtest-ha` didn't exist yet (`az group
   exists` confirmed `false` before starting). A subscription-wide `az resource list` search
   found the real cause: an unrelated Postgres Flexible Server also named `mm-loadtest-pg`,
   owned by a different engineer in a different resource group (`robert.davis-k8s-scale`,
   `eastus2`) in this same `Mattermost Dev` subscription - not a leftover from our own teardown.
   Postgres Flexible Server names are globally unique across all of Azure (like storage account
   names), and the original script hardcoded the name with no suffix. Fixed by adding
   `db_server_name()` to `lib/common.sh`, suffixing the name with the same cached `dns_suffix()`
   already used for the storage account and proxy/Grafana DNS labels (`mm-loadtest-pg-<suffix>`),
   and updating `01-database.sh`/`09-restore-dump.sh` to use it.

2. **`01-database.sh`'s private-DNS-zone setup wasn't idempotent against its own partial
   failure.** Retrying after fix #1 above hit two further, self-inflicted issues from the first
   attempt's partial run: `az network private-dns zone create` errored `PreconditionFailed` on a
   zone that a prior failed attempt had already created (unlike virtually every other `az ...
   create` call in this codebase, this one isn't create-or-update); and once that was guarded
   with an existence check, `az network private-dns link vnet create` still errored `Conflict`
   ("already linked to the virtual network") because a zone can have only **one** link to a given
   VNet regardless of link name - the existence check was (wrongly) keyed on the *new* link name
   (which had changed because of fix #1's suffix), not on whether the VNet already had *any*
   link. Fixed both in `01-database.sh`: skip zone creation if `az network private-dns zone show`
   already finds it, and skip link creation if `az network private-dns link vnet list` shows any
   existing link to `${VNET}` on that zone, regardless of its name.

3. **`ltagent` (and effectively the from-source `ltapi`/`ltcoordinator` build) silently failed to
   build on the load-test VM** with `error obtaining VCS status: exit status 128` (visible only
   in `07-post-config`'s final `ltagent init` step failing with `./bin/ltagent: No such file or
   directory` - the build failure itself was buried earlier in `provision-loadtest.sh.tmpl`'s
   output and didn't abort the script). Root cause: `provision-loadtest.sh.tmpl` runs as root (via
   `az vm run-command`) but `chown -R`s the cloned repo to `${VM_ADMIN_USER}` *before* building
   (see the [earlier ownership fix](#load-test-vm-mm-loadtest)) - so `go build`, still running as
   root, hits git's dubious-ownership protection when it tries to stamp VCS info into the binary,
   since the repo directory's owner (`mmadmin`) no longer matches the running user (`root`). Not
   fatal to this deployment's actual goal: `ltapi` runs via the portable Docker image (unaffected,
   built elsewhere and transferred - see
   [Switching ltapi to the portable Docker image](#switching-ltapi-to-the-portable-docker-image-mimicking-crimson-1)),
   `ltcoordinator` is built locally on the operator's machine by `08-operator-setup.sh`
   (unaffected), and `ltagent` itself is unused once the DB dump is loaded. Fixed anyway, for
   correctness, in `assets/provision-loadtest.sh.tmpl`: added `-buildvcs=false` to all three
   `go build` invocations, per the Go toolchain's own suggested fix in the error message - VCS
   stamping isn't needed for these binaries.

### Timing

Measured wall-clock, from a clean `az group exists` returning `false` through a fully restored,
licensed, clustered environment with the real 12M-post dataset loaded:

| Step | Result | Time |
|---|---|---|
| `deploy.sh` attempt 1 | failed (bug #1) | 6m 47s |
| `deploy.sh` attempt 2 | failed (bug #2, zone) | 3m 11s |
| `deploy.sh` attempt 3 | failed (bug #2, link) | 8m 19s |
| `deploy.sh` attempt 4 | **succeeded** | **29m 49s** |
| `09-restore-dump.sh` | **succeeded** (first try) | **15m 47s** |
| `08-operator-setup.sh` | succeeded (first try) | well under a minute |

- **Total wall-clock this session** (first attempt's start to the dump restore finishing,
  including all three failed attempts and the fixes made between them): **67m 33s**.
- **Clean-run total** (`deploy.sh` + `09-restore-dump.sh` alone, on a run that would hit none of
  the three now-fixed bugs above): **45m 36s** (29m49s + 15m47s).

Verified afterward: `https://mm-lt-proxy-udln2nkd.westus2.cloudapp.azure.com/api/v4/system/ping`
and `https://mm-lt-grafana-udln2nkd.westus2.cloudapp.azure.com/login` both `200`; row counts
confirmed at **11,841,269 posts, 16,088 users, 2 teams**; `cluster/status` showed both `mm-app-1`
and `mm-app-2`. DNS labels stayed the same as prior deployments (`udln2nkd`) since the cached
`dns_suffix` in `.state/` survived the teardown/redeploy cycle unchanged.

### Load test run: 2000 users (first successful run at the full target)

Ran `ltcoordinator` from the operator's machine against the restored dataset with
`deployment/azure/operator/coordinator.json` left at its default `ClusterConfig.MaxActiveUsers:
2000` / `NumUsersInc: 8` - the first run at the actual target scale, after the smaller 200- and
500-user confirmation runs above. Ramped cleanly to and held 2000/2000 active users.

**Bug hit while capturing proof of the run**: taking a Grafana snapshot (the "Publish to
snapshot.raintank.io" button) of the `Mattermost Performance Monitoring v2` dashboard failed with
`413 Request Entity Too Large` from nginx. Cause: `assets/nginx-grafana-site.conf.tmpl` had no
`client_max_body_size` directive, so nginx fell back to its 1MB default - too small for the
dashboard-JSON snapshot payload at this many panels/series. Fixed in both places: the template
(`client_max_body_size 50M;` added to the `server` block, for future deploys) and live on
`mm-metrics`'s `/etc/nginx/sites-available/grafana` (via `az vm run-command`, so no direct SSH was
needed), then `nginx -t` + `systemctl reload nginx`.

**Proof**: Grafana snapshot taken after the fix, confirming the successful 2000-user run:
https://snapshots.raintank.io/dashboard/snapshot/usIasoC5RRhoOI8BgUOyye0q0FIVhyc1

---

## Credentials (`secrets/`, gitignored)

| File | Contents |
|---|---|
| `db_admin_password` | Postgres Flexible Server admin (`mmdbadmin`) password |
| `storage_account_key` | Azure Files storage account key |
| `grafana_admin_password` | Grafana `admin` password |
| `mm_admin_password` | `Sys@dmin-sample1` - the dump's real sysadmin password |
| `mm_admin_note.txt` | Explains why this is the real account, not the one originally created |

## Public URLs

- Mattermost: `https://mm-lt-proxy-udln2nkd.westus2.cloudapp.azure.com`
- Grafana: `https://mm-lt-grafana-udln2nkd.westus2.cloudapp.azure.com`
