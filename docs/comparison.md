# Running an automated load-test comparison

## Introduction

This document explains how to run an automated load-test comparison.

## Prerequisites

- [Terraform](https://learn.hashicorp.com/terraform/getting-started/install). The minimum supported version is 0.12.
- AWS credentials to be used as described [here](https://www.terraform.io/docs/providers/aws/index.html#authentication).
- A valid Mattermost E20 license, required to run the load-test through the [`coordinator`](coordinator.md).

## Configuration

### deployer.json

To start with, a deployment should be configured through `deployer.json` config file.  
This will serve as a template to create all the required deployments to run the comparison.

```
cp config/deployer.sample.json config/deployer.json
```

Detailed documentation for this config file can be found [here](deployer_config.md).

### comparison.json

Next step is to configure the comparison itself:

```
cp config/comparison.sample.json config/comparison.json
```

Detailed documentation for this config file can be found [here](comparison_config.md).

### config.json and coordinator.json

Optionally, it's possible to further configure the load-test characteristics by editing `config.json` and `coordinator.json`.
More information can be found in the [load-test guide](local_loadtest.md).

#### Note

When starting a comparison, required configuration files are automatically read from the `config/` directory. If no files are found, defaults will be used.

## Comparison

### Setup ssh-agent

For the automated deployment to work, a [ssh-agent](https://linux.die.net/man/1/ssh-agent) needs to be running and loaded with a private key.

```sh
eval $(ssh-agent -s)
ssh-add PATH_TO_KEY
```

`PATH_TO_KEY` should be replaced with the path to the matching private key for `SSHPublicKey`, as previously [configured](deployer_config.md).

### Run the comparison 

```
go run ./cmd/ltctl comparison run
```

This command will start a fully automated load-test comparison as configured.  
When done, the command will output some results. 

There are a few interesting flags that the `comparison run` command supports:

- `--archive`  To output all artifact (reports and graphs) into a `.zip` file.
- `--format` Defines the format of the final output. It can be either `plain`
    for a text based output, or `json`.
- `--output-dir` An optional output directory where to write the output files.

#### Note

Depending on how it was configured, the comparison process can take hours to complete.

## Destroy 

```
go run ./cmd/ltctl comparison destroy
```

This command should be used to permanently destroy all deployed resources associated with the comparison.
