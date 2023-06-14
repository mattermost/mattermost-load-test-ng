# Running a load-test with a Terraform deployment

This guide describes how to setup and run a load-test using a [Terraform](https://www.terraform.io/intro/index.html) deployment.
Following this process it is possible to create a complete [load-test system](loadtest_system.md) in a mostly automated form.
This is the recommended way to load-test a Mattermost instance for production.

## Prerequisites

- [Terraform](https://learn.hashicorp.com/terraform/getting-started/install). Version 1.3.3 or greater (as long as it's in the v1.x series) is required.
- AWS credentials to be used as described [here](https://www.terraform.io/docs/providers/aws/index.html#authentication-and-configuration). If you're a Mattermost staff member, please use [AWS Single Sign-On](https://aws.amazon.com/blogs/security/aws-single-sign-on-now-enables-command-line-interface-access-for-aws-accounts-using-corporate-credentials/) to generate API credentials for the [AWS credentials file](https://www.terraform.io/docs/providers/aws/index.html#shared-credentials-file). Credentials generated for mattermost-loadtest will remain active for 12 hours, at which point you will need to generate new credentials using the Single Sign-On portal. If you save the profile in the credentials file, make sure to use the name `mm-loadtest`.
- A valid Mattermost Enterprise license, required to run the load-test through the [`coordinator`](coordinator.md). If you're a Mattermost staff member, you can get a test license in the [~Team: Self-Serve](https://community.mattermost.com/private-core/channels/team-self-serve) channel.

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

Detailed documentation for the deployer's config can be found [here](config/deployer.md). At least, make sure to set the `SSHPublicKey` to the path of your public key, `MattermostLicenseFile` to the path of an enterprise license, and the `ClusterName` to a unique value within your AWS account. 

## Deployment

### Setup ssh-agent

For the deployer to work, an [ssh-agent](https://linux.die.net/man/1/ssh-agent) needs to be running and loaded with a private key.

```sh
eval $(ssh-agent -s)
ssh-add PATH_TO_PRIVATE_KEY
```

`PATH_TO_PRIVATE_KEY` should be replaced with the path to the matching private key for `SSHPublicKey`, as previously [configured](config/deployer.md).

### Create a new deployment

```sh
go run ./cmd/ltctl deployment create
```

This command can take several minutes to complete when creating a [full](loadtest_system.md) deployment. By default, the console will keep logging info messages. If it does not, then something's wrong.

Once done, it will output information about the entire cluster. Everything will be now ready to start a new load-test.

If you see an error when running `deployment create` mentioning a "resource already exists", it is most likely because your `ClusterName` is not a unique value within your AWS account. Run `go run ./cmd/ltctl deployment destroy` to clean up the half created deployment. Then change the `ClusterName` to something more unique for your loadtest and try again.

### Get information on the current deployment

```sh
go run ./cmd/ltctl deployment info
```

This will show information about the current deployment.

### Synchronize the current deployment

```sh
go run ./cmd/ltctl deployment sync
```

This command will synchronize any changes made manually to the AWS cluster.

### Optionally configure coordinator and load-test

When starting a load-test with the `ltctl` command, required configuration files are automatically uploaded to the instance hosting the [coordinator](coordinator.md).
If no files are found, defaults will be used.

#### Copy default config

To configure the [coordinator](coordinator.md), `config/coordinator.json` should be created and/or edited.

```sh
cp config/coordinator.sample.json config/coordinator.json
```

Its documentation can be found [here](config/coordinator.md).

#### Copy default config

To configure the load-test `config/config.json`, should be created and/or edited.

```sh
cp config/config.sample.json config/config.json
```

Its documentation can be found [here](config/config.md).

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

## Comparing results

To compare the results of your load tests, see [here](compare.md).

## Debugging

### SSH access to the terraformed hosts

To access one of the terraformed hosts via ssh, invoke `ltctl ssh` with the appropriate target:
* One of the instance names (invoke `ltctl ssh` to list them)
* `coordinator` to connect to the first agent doing double duty as the loadtest coordinator
* `proxy` to connect to the instance running Nginx
* `metrics`, `prometheus` or `grafana` to connect to the instance running all metrics related services
