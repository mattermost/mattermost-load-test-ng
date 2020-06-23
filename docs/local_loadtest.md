# Running a load-test locally

## Introduction

This guide describes how to run a load-test locally (and mostly manually).  
Doing this is particularly useful when testing changes to the load-test tool itself.  
It's also a great way to learn how the whole load-testing process works before trying with more advanced deployments.

There are a few ways to run a load-test locally, in order of complexity:

- Run the `ltagent` command directly. 
- Run a load-test through the load-test agent API server `ltapi`.
- Run a load-test through the [`coordinator`](coordinator.md).

## Prerequisites

Before starting a new load-test, a newly created (and running) Mattermost instance with system admin credentials is required.  

### Clone the repository

```sh
git clone https://github.com/mattermost/mattermost-load-test-ng
```

### Enter the source directory

```sh
cd mattermost-load-test-ng
```

### Copy and modify needed configuration files

It's suggested to copy the required config files and edit them accordingly.

```sh
cp config/config.sample.json config/config.json
cp config/simplecontroller.sample.json config/simplecontroller.json
```

The load-test config file is documented [here](loadtest_config.md).  
The default [`UserController`](controllers.md) is the `SimpleController`. Its config file is documented [here](simplecontroller_config.md).  

### Run the initialization

```sh
go run ./cmd/ltagent init
```

Running this command will create initial teams and channels for the users to join on the target MM instance.

## Running a basic load-test

A new load-test can be started with the following command:

```sh
go run ./cmd/ltagent -n 10 -d 60
```

This will start a load-test running the specified number of users (10) for 60 seconds.

#### Note

Command line arguments take precedence over configuration settings.

## Running a load-test through the load-test agent API server

A more advanced way to run a load-test is to use the provided load-test agent API server.

### Start the API server

```sh
go run ./cmd/ltapi
```

This will start the server and expose the HTTP API on port 4000 (default).  
Using a different terminal it's possible to issue commands to create and run a load-test agent:

### Create a new load-test agent

To start a new load-test agent via API, the request structure should be in the form of:

```json
{
  "LoadTestConfig": {
    ...
  },
  "SimpleControllerConfig": {
    ...
  },
  "SimulControllerConfig": {
    ...
  }
}
```

The API will create controller specific configuration from the request. However, if there is no controller configuration or errors while reading it from the request, the server will read the default controller configuration locally.

```sh
curl -d "{\"LoadTestConfig\": $(cat config/config.json)}" http://localhost:4000/loadagent/create\?id\=lt0
```

### Start the load-test agent

```sh
curl -X POST http://localhost:4000/loadagent/lt0/run
```

### Add active users

```sh
curl -X POST http://localhost:4000/loadagent/lt0/addusers?amount=10
```

### Remove active users

```sh
curl -X POST http://localhost:4000/loadagent/lt0/removeusers?amount=10
```

### Stop the load-test agent

```sh
curl -X POST http://localhost:4000/loadagent/lt0/stop
```

### Destroy the load-test agent

```sh
curl -X DELETE http://localhost:4000/loadagent/lt0
```

## Running a load-test through the coordinator

An even more advanced way to run a load-test is through the use of the [`coordinator`](coordinator.md).  
This is especially needed when we need to figure out the maximum number of users the target instance supports.  
The [`coordinator`](coordinator.md) does also help running a load-test across a cluster of agents.

### Prerequisites 

In order to run the [`coordinator`](coordinator.md) a [Prometheus](https://prometheus.io/docs/introduction/overview/) server needs to be running and
correctly [configured](https://docs.mattermost.com/deployment/metrics.html) for the target Mattermost instance.  

### Start the load-test agent API server

The first step is having the server running.

```sh
go run ./cmd/ltapi
```

### Configure the coordinator

Before starting the [`coordinator`](coordinator.md), the default configuration file should be copied and modified accordingly.

```sh
cp config/coordinator.sample.json config/coordinator.json
```

Its documentation can be found [here](coordinator_config.md).

### Run the coordinator

From a different terminal we can then run the [`coordinator`](coordinator.md).

```sh
go run ./cmd/ltcoordinator
```

This will start running a load-test across the configured cluster of load-test agents.

### Run coordinator using the API server

Similar to what happens for the load-test agents, a coordinator (or more than
one) can be created and run using the API server.

### Start the API server

```sh
go run ./cmd/ltapi
```

### Create a coordinator

```sh
curl -d "{\"CoordinatorConfig\": $(cat config/coordinator.json), \"LoadTestConfig\": $(cat config/config.json)}" http://localhost:4000/coordinator/create\?id\=ltc0
```

### Run a coordinator

```sh
curl -X POST http://localhost:4000/coordinator/ltc0/run
```

### Stop a coordinator

```sh
curl -X POST http://localhost:4000/coordinator/ltc0/stop
```

### Destroy a coordinator

```sh
curl -X DELETE http://localhost:4000/coordinator/ltc0
```
