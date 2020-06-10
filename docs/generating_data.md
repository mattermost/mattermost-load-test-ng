# Generating data

## Introduction

This guide explains how to populate an instance with some realistic data
using [`GenController`](controllers.md#gencontroller).

## Prerequisites

- Before starting, make sure you read and understood [how to run a load-test locally](local_loadtest.md).

## Configuration

First of all, the type of `UserController` to be used should be set to
`generative` as explained [here](loadtest_config.md#usercontrollerconfiguration).

### Copy and modify needed configuration file

```sh
cp config/gencontroller.default.json config/gencontroller.json
```

The `GenController` config file is documented [here](gencontroller_config.md).  

### Run the load-test agent

```sh
go run ./cmd/ltagent -n 10 -r 0.5
```

This will run 10 users at double than normal speed.  
The command will exit once all the configured data has been successfully
created.

### Batching

If you plan on running high number of users it might be unfeasible to generate
data for all of those at the same time.  
In such a case you can batch the generation process by providing some useful flags:

```sh
go run ./cmd/ltagent -n 100 -r 1.0 --user-offset 100 --user-prefix ltuser
```

This will run users `ltuser-100` to `ltuser-199`.

