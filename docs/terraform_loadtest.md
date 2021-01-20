# Running a load-test with a Terraform deployment

This guide describes how to setup and run a load-test using a [Terraform](https://www.terraform.io/intro/index.html) deployment.  
Following this process it is possible to create a complete [load-test system](loadtest_system.md) in a mostly automated form.  
This is the recommended way to load-test a Mattermost instance for production.

## Prerequisites

- [Terraform](https://learn.hashicorp.com/terraform/getting-started/install). Version 0.12 is required.
- AWS credentials to be used as described [here](https://www.terraform.io/docs/providers/aws/index.html#authentication).
- A valid Mattermost E20 license, required to run the load-test through the [`coordinator`](coordinator.md).

**Note**

If authenticating using the [AWS credentials file](https://www.terraform.io/docs/providers/aws/index.html#shared-credentials-file), the profile to use is `mm-loadtest`.

### Clone the repository

```sh
git clone https://github.com/mattermost/mattermost-load-test-ng
```

### Enter the source directory

```sh
cd mattermost-load-test-ng
```

### Copy and modify the required configuration

In order to start the deployment process, it is required to configure the deployer appropriately.

```sh
cp config/deployer.sample.json config/deployer.json
```

Detailed documentation for the deployer's config can be found [here](deployer_config.md).

## Deployment

### Setup ssh-agent

For the deployer to work, a [ssh-agent](https://linux.die.net/man/1/ssh-agent) needs to be running and loaded with a private key.

```sh
eval $(ssh-agent -s)
ssh-add PATH_TO_KEY
```

`PATH_TO_KEY` should be replaced with the path to the matching private key for `SSHPublicKey`, as previously [configured](deployer_config.md).

### Create a new deployment

```sh
go run ./cmd/ltctl deployment create
```

This command can take several minutes to complete when creating a [full](loadtest_system.md) deployment.  
Once done, it will output information about the entire cluster. Everything will be now ready to start a new load-test.

### Get information on the current deployment

```sh
go run ./cmd/ltctl deployment info
```

This will show information about the current deployment.

### Optionally configure coordinator and load-test

When starting a load-test with the `ltctl` command, required configuration files are automatically uploaded to the instance hosting the [coordinator](coordinator.md). 
If no files are found, defaults will be used.

#### Copy default config

To configure the [coordinator](coordinator.md), `config/coordinator.json` should be created and/or edited. 

```sh
cp config/coordinator.sample.json config/coordinator.json
```

Its documentation can be found [here](coordinator_config.md).

#### Copy default config

To configure the load-test `config/config.json`, should be created and/or edited.

```sh
cp config/config.sample.json config/config.json
```

Its documentation can be found [here](loadtest_config.md).

### Start a load-test

```sh
go run ./cmd/ltctl loadtest start
```

This will begin to run the load-test across the whole cluster of load-test agents.

### Show the load-test status

```sh
go run ./cmd/ltctl loadtest status
```

This will print information about the status of the current load-test.

### Stop the running load-test

```sh
go run ./cmd/ltctl loadtest stop
```

This will stop the currently running load-test.

### Re-initialize a load-test

```sh
go run ./cmd/ltctl loadtest reset
```

This will completely erase data on the target instance's database and will run again the init process.

### Destroy the current deployment

When done with a deployment, it's suggested to run:

```sh
go run ./cmd/ltctl deployment destroy
```

This will permanently destroy all resources for the current deployment.

## Debugging

### SSH access to the terraformed hosts

To access one of the terraformed hosts via ssh, invoke `ltctl ssh` with the appropriate target:
* One of the instance names (invoke `ltctl ssh` to list them)
* `coordinator` to connect to the first agent doing double duty as the loadtest coordinator
* `proxy` to connect to the instance running Nginx
* `metrics`, `prometheus` or `grafana` to connect to the instance running all metrics related services
