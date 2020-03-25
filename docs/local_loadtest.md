# How to run a load-test locally

## Introduction

This is a short guide on how to run a load-test locally (and mostly manually).
This is particularly useful when testing changes to the load-test tool itself.
It's also a great way to learn how the whole load-testing process works before
trying more advanced deployments.

There are a few of ways to run a load-test locally:

- Run the `loadtest` command. This is the simplest way to quickly start a load-test.
- Run a load-test through the load-test agent API server.
- Run a load-test through the [`coordinator`](coordinator.md).

## Prerequisites

Before beginning a new load-test, a newly created and running Mattermost instance with system admin credentials is required.  

### Clone the `load-test-ng` repository

```sh
git clone https://github.com/mattermost/mattermost-load-test-ng
```

## Run a basic load-test

A new load-test can be started with the following command:

```sh
go run ./cmd/loadtest -c config/config.default.json -s config/simplecontroller.json -d 60
```

This will run a load-test with default values for 60 seconds.  
It's suggested to copy the required config files and edit accordingly.

```sh
cp config/config.default.json config/config.json
cp config/simplecontroller.default.json config/simplecontroller.json
```

The default [`UserController`](controllers.md) is the `SimpleController` hence
the need for the additional configuration file.

## Run a load-test through the load-test agent API server

A more advanced way to run a load-test is to use the provided load-agent API
server.

### Start the server

```sh
go run ./cmd/loadtest server
```

This will expose an HTTP API on port 4000 (default).
Using another terminal it's possible to issue commands to create and manage a load-test agent.

### Create a new load-test agent

```sh
curl -d @config/config.json http://localhost:4000/loadagent/create?id=lt0
```

### Start the load-test agent

```sh
curl -X POST http://localhost:4000/loadagent/lt0/run
```

### Add users

```sh
curl -X POST http://localhost:4000/loadagent/lt0/addusers?amount=10
```

### Remove users

```sh
curl -X POST http://localhost:4000/loadagent/lt0/removeusers?amount=10
```

### Stop the load-test agent

```sh
curl -X POST http://localhost:4000/loadagent/lt0/stop
```

## Run a load-test through the [`coordinator`](coordinator.md).

A slightly more advanced way to run a load-test is through the use of the [`coordinator`](coordinator.md).

### Prerequisites 

In order to run the `coordinator` a [Prometheus](https://prometheus.io/docs/introduction/overview/) server needs to be running and
correctly [configured](https://docs.mattermost.com/deployment/metrics.html) for the target Mattermost instance.  
Before running, the default configuration file `config/coordinator.default.json` should also be copied and modified accordingly.

### Start the load-test agent API server

```sh
go run ./cmd/loadtest server
```

### Run the `coordinator`

```sh
go test -v ./cmd/coordinator -c config/coordinator.json -l config/config.json
```

