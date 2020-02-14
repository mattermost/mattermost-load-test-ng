# Mattermost loadtest

Mattermost loadtest is a standalone tool written in [Go](https://golang.org/) for profiling [Mattermost](https://github.com/mattermost/mattermost-server) under heavy load simulating real-world usage of a server installation at scale.

## Goals/Features

- No external dependencies.
- Loosely coupled components.
- Short, *do one thing only* functions.
- State handling out of main logic.
- Theoretically no need to bulkload.
- No need to synchronize state between multiple loadtesting instances.
- Easy to add/remove concurrent users at execution time.

## Running

__Pre-requisites__

- Have a running Mattermost server, with known system admin credentials.
- Copy the `config.default.json` file to `config.json` and populate the fields accordingly.
- If you want to run a coordinator too, copy the `coordinator.default.json` file to `coordinator.json` and populate the fields accordingly.

#### Populate an empty instance

- `go run ./cmd/loadtest init`

This populates a Mattermost instance with some teams and channels.

#### Run a basic load test

- `go run ./cmd/loadtest`

This runs for 60 seconds and stops.

#### Start a load test agent

- `go run ./cmd/loadtest server`

This starts a load agent in server mode, where it exposes an HTTP server to interact with the load test. Here are some commands that can be used to interact with it:

- `curl -d @config/config.json http://localhost:4000/loadagent/create?id=lt1`

This creates a load test with id "lt1".

- `curl -X POST http://localhost:4000/loadagent/lt1/run`

This starts the load test "lt1" which was created in the previous step. The load test will be started with `InitialActiveUsers` under `UsersConfiguration` in config.json.

- `curl -X POST http://localhost:4000/loadagent/lt1/addusers?amount=100`

This adds 100 new users to the load test.

- `curl -X POST http://localhost:4000/loadagent/lt1/removeusers?amount=10`

This removes 10 users from the load test.

- `curl -X POST http://localhost:4000/loadagent/lt1/stop`

This stops the load test.

#### Start a load test coordinator (EE edition feature)

- Start a load test agent.
- Start a Prometheus server.
- `go run ./cmd/coordinator`

The objective of the coordinator is to find the number of users that a Mattermost instance can handle without degrading performance. It will slowly ramp up the number of users through the load agent API, and check the Prometheus queries mentioned in `MonitorConfig`.

When the metrics cross their thresholds, it will reduce the number of users and wait for performance to stablize.

#### Deploy a load test environment using Terraform (in-progress)

- Ensure that you have [terraform](https://www.terraform.io/downloads.html) installed.
- Ensure that you have an SSH key pair [generated](https://help.github.com/en/github/authenticating-to-github/generating-a-new-ssh-key-and-adding-it-to-the-ssh-agent).
- Ensure that you have CLI access to your AWS account and have setup an AWS profile using `aws configure`.
- `go run ./cmd/loadtest deploy`

This will deploy a load test environment. It is currently a WIP and more detailed documentation will be added soon.

## Documentation

Documentation and implementation details can be found in the [docs](docs/) folder.

## Development

A sample implementation can be found in the [example](example/) folder.

## Help

If you need any help you can join the [Developers: Performance](https://community.mattermost.com/core/channels/developers-performance) channel and ask developers any question related to this project.
