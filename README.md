# Mattermost load-test-ng

A load-testing toolkit for [Mattermost](https://github.com/mattermost/mattermost),
written in [Go](https://golang.org/). It simulates large numbers of concurrent
users performing realistic actions against a target Mattermost server so you
can answer questions like "will this deployment handle 2,000 users?" or
"did our last release regress performance?"

It's a rewrite of the original
[mattermost-load-test](https://github.com/mattermost/mattermost-load-test) tool
and is the supported way to load-test Mattermost going forward.

> Join the [Developers: Performance](https://community.mattermost.com/core/channels/developers-performance) for questions, feedback, and ideas.

## What's in the box

Four binaries, all built from this repo:

| Binary | Role |
| --- | --- |
| `ltagent` | Generates load. Simulates one or more users running actions against a Mattermost server. |
| `ltapi` | HTTP API server wrapping the agent. Lets you create, start, scale, and stop load-test agents over the network. |
| `ltcoordinator` | Orchestrates a cluster of agents to find the maximum number of concurrent users a Mattermost deployment supports. Drives the feedback loop using Prometheus metrics. |
| `ltctl` | CLI for managing AWS-based load-test deployments (provision via Terraform, drive tests, collect reports). |

You can use these individually or together depending on how big a test you
need to run.

## How the pieces fit together

For a full coordinator-driven test, the architecture looks like:

```
                        +--------------------------------+
                        |                                |
                        |                                |
              +---------v---------+                      |
              |                   |                      |
       +------|    coordinator    |------+               |
       |      |                   |      |               |
       |      +-------------------+      |               |
       |                |                |               |
       |                |                |               |
+------v------+  +------v------+  +------v------+        |
|  load-test  |  |  load-test  |  |  load-test  |        |
|    agent    |  |    agent    |  |    agent    |        |
+-------------+  +-------------+  +-------------+        |
       |                |                |               |
       |                |                |               |
       |         +------v------+         |               |
       |         |  mattermost |         |               |
       +--------->   instance  <---------+               |
                 +-------------+                         |
                        |                                |
                        |                                |
                 +------v-------+                        |
                 |              |                        |
                 |  prometheus  |------------------------+
                 |              |
                 +--------------+
```

Agents drive load against Mattermost. Prometheus scrapes both Mattermost and
the agents. The coordinator queries Prometheus to decide when to add or
remove users from the test, eventually converging on the maximum number the
target can sustain. Simpler setups skip the coordinator and Prometheus — see
the "What do you want to do?" section below.

## What do you want to do?

### Try a load test against an existing Mattermost

Fastest path. One container, point it at your Mattermost, watch your existing
Grafana while you ramp up.

```bash
docker run --rm \
  -v "$(pwd)/config.json:/mattermost-load-test/config/config.json:ro" \
  mattermost/mattermost-load-test-ng:vX.Y.Z \
  ltagent -n 100 -d 600
```

→ Full guide: [docs/docker_loadtest.md](docs/docker_loadtest.md)

### Find the maximum supported users automatically

Run the [coordinator](docs/coordinator.md) against a Prometheus that's
scraping your target. The coordinator gradually increases active users until
performance metrics indicate degradation, then converges on the supported
maximum.

→ Coordinator guide: [docs/coordinator.md](docs/coordinator.md)
→ Full local walk-through: [docs/local_loadtest.md](docs/local_loadtest.md)

### Stand up a full benchmarking environment on AWS

Multi-VM Terraform deployment that provisions agents, a coordinator,
Mattermost, Postgres, Prometheus, and Grafana. Driven by `ltctl`.

```bash
cp config/deployer.sample.json config/deployer.json
$EDITOR config/deployer.json
ltctl deployment create
ltctl loadtest start
```

→ Full guide: [docs/terraform_loadtest.md](docs/terraform_loadtest.md)

### Compare performance across two Mattermost versions

Drive identical load against two builds and produce a side-by-side report.

→ Full guide: [docs/comparison.md](docs/comparison.md) and [docs/compare.md](docs/compare.md)

### Develop the tool itself

Clone, build with the Makefile, run tests, contribute.

```bash
git clone https://github.com/mattermost/mattermost-load-test-ng
cd mattermost-load-test-ng
make install        # builds ltagent, ltapi, ltctl into ./bin
make test           # runs the test suite
make check-style    # lints + validates the sample config files
```

→ Developer notes: [docs/developing.md](docs/developing.md)

## Contributing

Issues and pull requests welcome. Before opening a PR:

- Read [docs/developing.md](docs/developing.md) for repo conventions and how
  to add new load-testing actions.
- Run `make check-style` and `make test` locally.
- For load-testing coverage of a new Mattermost feature, see
  [docs/coverage.md](docs/coverage.md).

## Help

If you need any help you can join the [Developers: Performance](https://community.mattermost.com/core/channels/developers-performance) channel and ask developers any question related to this project.

Code-level documentation is also available on [GoDoc](https://godoc.org/github.com/mattermost/mattermost-load-test-ng).

## Additional information

More documentation on individual components and workflows can be found in
the `docs/` directory:

**Getting started**
- [docs/docker_loadtest.md](docs/docker_loadtest.md) — Single-container load test via Docker
- [docs/local_loadtest.md](docs/local_loadtest.md) — Running load tests locally from source
- [docs/terraform_loadtest.md](docs/terraform_loadtest.md) — Multi-VM AWS deployment
- [docs/load-test-how-to-use.md](docs/load-test-how-to-use.md) — End-to-end walkthrough for testing a new feature

**Architecture & internals**
- [docs/coordinator.md](docs/coordinator.md) — How the coordinator finds max supported users
- [docs/controllers.md](docs/controllers.md) — User controllers (simple, simulative, browser)
- [docs/loadtest_system.md](docs/loadtest_system.md) — Overall system design
- [docs/implementation.md](docs/implementation.md) — Implementation notes

**Configuration**
- [docs/config/](docs/config/) — Reference docs for each config file (`config.json`, `coordinator.json`, `deployer.json`, etc.)

**Workflows**
- [docs/comparison.md](docs/comparison.md) — Comparing performance across Mattermost versions
- [docs/compare.md](docs/compare.md) — Generating and reading comparison reports
- [docs/generating_data.md](docs/generating_data.md) — Pre-populating Mattermost with test data
- [docs/coverage.md](docs/coverage.md) and [docs/coverage-frequency.md](docs/coverage-frequency.md) — Action coverage and frequency tuning
- [docs/plugin_browser_loadtest.md](docs/plugin_browser_loadtest.md) — Browser-based load with the LTBrowser sidecar
- [docs/browser_simulations_registry.md](docs/browser_simulations_registry.md) — Registry of available browser simulations
- [docs/external_auth_providers.md](docs/external_auth_providers.md) — Load-testing with SAML / OpenID Connect

**Reference**
- [docs/faq.md](docs/faq.md) — Frequently asked questions and troubleshooting
- [docs/developing.md](docs/developing.md) — Developer setup and conventions
- [docs/release.md](docs/release.md) — Release process

## License

Apache License 2.0 — see [LICENSE.txt](LICENSE.txt).
