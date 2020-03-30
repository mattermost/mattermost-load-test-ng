# Running a load-test through a Terraform deployment

This guide describes how to setup and run a load-test using a [Terraform](https://www.terraform.io/intro/index.html) deployment.  
This is the recommended way to load-test a system for production. Following this process it is possible to automatically setup a complete [load-test system](loadtest_system.md).

## Prerequisites

- [Terraform](https://learn.hashicorp.com/terraform/getting-started/install).
- AWS credentials to be used as described [here](https://www.terraform.io/docs/providers/aws/index.html#authentication).
- A Mattermost E20 license is required to run the load-test through the [`coordinator`](coordinator.md).

### Clone the repository

```sh
git clone https://github.com/mattermost/mattermost-load-test-ng
```

### Copy and modify required config

In order to start the deployment process it is required to [configure](deployer_config.md) the deployer appropriately.

```sh
cp config/deployer.default.json config/deployer.conf
```

## Create the deployment

### Setup ssh-agent

For the deployer to work, a [ssh-agent](https://linux.die.net/man/1/ssh-agent) needs to be running and a private key added.

```sh
eval `ssh-agent -s`
ssh-add PATH_TO_KEY
```

`PATH_TO_KEY` should be the path to the matching private key for `SSHPublicKey`, as previosly [configured](deployer_config.md).

### Create a new deployment

```sh
go run ./cmd/deployer create
```

This command can take several minutes to complete for a full deployment.

### Run the coordinator

Connect through SSH to the instance hosting the [coordinator](coordinator.md).  
Optionally [configure](coordinator_config.md) it by editing `config/coordinator.json`.

```sh
go run ./cmd/coordinator
```

This will begin to run the load-test across the load-test agent cluster.

### Destroy the current deployment

```sh
go run ./cmd/deployer destroy
```

This will permanently destroy all resources for the current deployment.
