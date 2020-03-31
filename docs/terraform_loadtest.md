# Running a load-test with a Terraform deployment

This guide describes how to setup and run a load-test using a [Terraform](https://www.terraform.io/intro/index.html) deployment.  
Following this process it is possible to create a complete [load-test system](loadtest_system.md) in a mostly automated form.  
This is the recommended way to load-test a Mattermost instance for production.

## Prerequisites

- [Terraform](https://learn.hashicorp.com/terraform/getting-started/install)
- AWS credentials to be used as described [here](https://www.terraform.io/docs/providers/aws/index.html#authentication)
- A Mattermost E20 license, required to run the load-test through the [`coordinator`](coordinator.md)

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

### Copy and modify required config

In order to start the deployment process it is required to configure the deployer appropriately.

```sh
cp config/deployer.default.json config/deployer.json
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
go run ./cmd/deployer create
```

This command can take several minutes to complete when creating a [full](loadtest_system.md) deployment.  
Once done, it will output information about the entire cluster. Everything will be now ready to start a new load-test.

### Run the coordinator

Connect via SSH to the instance hosting the [coordinator](coordinator.md).  
Optionally configure the [coordinator](coordinator_config.md) by editing `config/coordinator.json` and the [load-test agents](loadtest_config.md) by editing `config/config.json`.

```sh
go run ./cmd/coordinator
```

This will begin to run the load-test across the whole load-test agent cluster.

### Destroy the current deployment

When done with a deployment, it's suggested to run:

```sh
go run ./cmd/deployer destroy
```

This will permanently destroy all resources for the current deployment.
