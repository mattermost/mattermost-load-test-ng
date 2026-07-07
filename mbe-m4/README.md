# M4 — Control and 0% arms

Full-profile unbounded load tests, each restored from an independent clone of the M3 production
snapshot (`cpmbem3-m3-prod-20260706`, `DBName=cpmbem3db`). Answers Q1 (hook/seam overhead: control
vs 0% MBE traffic). See `mbe-load-test-plan.md` §9 (M4) and the "Verified M4-M6 Setup Instructions".

## Files

| File | Purpose |
|---|---|
| `deployer-control.json` | Control arm deployer. No plugin, no config patch, `ClusterIdentifier=cpmbem4c-restored-db`. Standard report graphs only. |
| `deployer-0pct.json` | 0% arm deployer. MBE plugin installed, `config-patch-0pct.json`, `ClusterIdentifier=cpmbem4z-restored-db`. Standard + MBE report graphs. |
| `config-patch-0pct.json` | Plugin PKCS#11 settings (token `pbe-dev`, key `pbe-kek-dev`). 0% arm only. |
| `coordinator.json` | v11.8 monthly profile verbatim: `NumUsersInc/Dec=6`, `RestTimeSec=2`, `MaxActiveUsers=20000`, 9 alert queries. |
| `config.json` | v11.8 monthly agent/loadtest config verbatim. |
| `simulcontroller.json` | v11.8-equivalent action mix + `MBEChannelWeight=0` (hard exclusion) + `MBEChannelIdsFile`. Same for both arms. |
| `mbe-channels.json` | 50 MBE channel IDs from the M3 snapshot (`mbe-prod-0..49`). Copied from `mbe-m3/`. Must be uploaded to every agent at the `MBEChannelIdsFile` path. |

## Arm differences

| | Control (`cpmbem4c`) | 0% (`cpmbem4z`) |
|---|---|---|
| Plugin | not installed | installed + enabled (bring-up) |
| HSM | none | SoftHSM (bring-up) |
| `ConsumePostHook` | false (default) | true (bring-up env) |
| `AggregatePluginMetrics` | default | true (bring-up env) |
| `channelguards` | DELETE before first boot | keep |
| `MBEChannelWeight` | 0 | 0 |

Everything else (server tarball, agent binary, coordinator thresholds, infra profile, DB lineage) is
identical between arms — they differ only by plugin/HSM/flag state.

## Deploy notes

- scp the whole dir to the flat path `/home/ubuntu/cp/mbe-m4/` on ltserver (ltctl reads deployer file
  references from there, not the git worktree).
- Copy `coordinator.json`, `config.json`, `simulcontroller.json` into the ltctl working dir's
  `config/` before `ltctl loadtest start` (they are read from the process cwd).
- Restore each arm's cluster from the snapshot **before** `deployment create`; terraform attaches the
  DB security group to the restored cluster automatically (`create.go`).
- Run `ltctl loadtest start` from `/home/ubuntu/cp/load-test-ng-mbe`.
