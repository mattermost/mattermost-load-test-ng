# Generating data

When running a load-test, there are many variables that can affect its performance. One of them is the shape and size of the data the system manages. Your analysis may require to keep that dataset constant for all of your tests, or maybe you actually want to fix all other variables and only change the initial state of the database. In both cases, you need a process you can follow to generate some data reliably.

This document explains such a process, enabled by one of the controllers implemented in the tool: [`gencontroller`](controllers.md#gencontroller).

## Prerequisites

Before starting, make sure you have read and understood [how to run a load-test locally](local_loadtest.md).

## Process

The process can be divided in the following phases:

1. Setup your target system.
2. Configure the load-test tool.
3. Run the test to generate the data.
4. Dump and archive the data.

### Target system setup

While the load-test will send the needed requests to generate the data configured, you first need a system that can listen to those requests.

How to perform this preliminary step will depend on where you want to generate the data:

- Local setup: if you want to run everything locally, make sure to follow the server documentation to setup a server in your laptop (you can use the usual developer workflow, or run a released version, up to you).
- Existing server: if you want to use an existing server, then this step is already done :) Beware that this process will create potentially lots of data in that system, so you should never target a production system here.
- Ad-hoc AWS setup: you can use the load-test deployer to setup a new cluster in AWS. Read the [Deployment section in the Terraform guide](terraform_loadtest.md#deployment) to learn how to do it. In this step, you can also deploy an agent to its own instance, from which you will start the test later.

In any case, you'll end up with a running Mattermost server. It's very important that this server satisfies two conditions:
1. It has a sysadmin user whose credentials you can pass to the load-test agent.
2. It is reachable by the machine from where you plan to run the agent.

The first condition is easy to satisfy: just note the username and password of any sysadmin, or create a new one it if it does not exist. For the second, it depends on how your Mattermost is deployed: if it's a local setup and you plan to run the agent from the same machine, then there's no problem; if it's an ad-hoc AWS setup, then the deployer has already taken care of this for you; if it's an existing deployment, simply make sure that the machine with the agent and the machine with the server can see each other.

### Configuration

Now that you have a working Mattermost server ready to react to the agent's requests, we need to configure the agent itself. Go to the machine where the agent will run, and follow these steps:

1. First of all, make sure that the load-test is pointing to the target system. You can do so by changing the [`ConnectionConfiguration` in the `config/config.json` file](config/config.md#connectionconfiguration). This step is already taken care of if you used the load-test deployer to setup an AWS cluster.
2. Then, verify that the [`UserController` in `config/config.json`](config/config.md#usercontrollerconfiguration) is set to `generative`. Also, set all absolute numbers under [`InstanceConfiguration` in that same file](config/config.md#usercontrollerconfiguration) to 0, since some commands do run an `init` process that populates the database with the data configured there. Since we are generating the data ourselves, we don't want anything done for us in that phase.
3. Then, create or modify the gencontroller configuration file in `config/gencontroller.json`. You can use the sample file as a starting point:
```sh
cp config/gencontroller.sample.json config/gencontroller.json
```
4. Now tweak everything you need in the controller configuration. All the settings are [documented](config/gencontroller.md), but here are some recommendations:
  - Make sure that the number of DMs is possible with the number of users you will run (more on simulated users later). If you run 10 users, the maximum number of different DMs is 45 so don't use a number larger than that. The formula for the maximum number of DMs is $n(n-1)2$, with $n$ the number of users.
  - All the settings follow specific distributions in real servers, so make sure to run some numbers in the server you want to mimic to make an educated decision here. As an example, the vast majority of channels in a normal server are DMs and GMs, accounting for 75% and 15% respectively, while private channels represent a 7% and public ones only a 3%.
  
You don't need to modify any of the other configuration files, since they are not considered when using a gencontroller.


### Data generation

With everything setup and configured, we get to the main step: generating the data.

There are two more variables that you need to choose:
- The number of simulated users that will run the actions to generate the data.
- The rate of each simulated user; i.e., the speed at which it will run the actions.

There is no golden rule on how to choose these numbers. The best way to decide is to run a couple of initial tests and monitor  how the system behaves.
- The more users you run, the faster the data will be generated.
- The larger the rate is, the slower the data will be generated (yes, this rate is more of a period, not a rate, we'll change the name at some point).

You probably want the data to be generated as quickly as possible, but be cautious, since this controller can quickly become a DDoS tool if configured with too many users or a quick enough loop. If you deployed a cluster to AWS, you should have Grafana and Prometheus automatically configured, so use them to monitor if the system can handle the load you're generating.

In any case, to both experiment with these numbers and to run your final tests, go to your agent's machine and run the following:

```sh
go run ./cmd/ltagent -n 10 -r 0.5
```

Those two flags are the ones you will want to tweak:
- `-n` is the number of simulated users.
- `-r` is the rate at which the user performs the actions (a number of 0.5 will halve the speed of the actions, while 2 will double it).

The command will exit once all the configured data has been successfully created. You can also stop the process early with <kbd>Ctrl-C</kbd>.

#### Batching

If you plan on running high number of users it might be unfeasible to generate data for all of those at the same time. In such a case you can split the generation process in batches by providing some useful flags:

```sh
go run ./cmd/ltagent -n 100 -r 1.0 --user-offset 100 --user-prefix ltuser
```

- `-n 100` specifies we want to run 100 users.
- `--user-prefix ltuser` specifies that all users will have a username starting with `ltuser-`.
- `--user-offset 100` specifies that the first user simulated will be `ltuser-100`.

In short, this command will run users `ltuser-100`, `ltuser-101`, ..., `ltuser-199`.

For example, you could use this to build a loop like the following, which will run 1000 users in batches of 100 each time:

```sh
for i in `seq 0 100 1000`; do
    go run ./cmd/ltagent -n 100 -r 1.0 --user-offset %d --user-prefix ltuser
done
```

### Data archival

When the data is finally generated, you will need to dump and archive it. How this process is performed is really up to you, but there are a few considerations:

- You can simply use the export tool in Mattermost, although it lacks some features to do a complete backup of the data.
- The best way is to simply dump the database. This will depend on whether the server is using MySQL or PostgreSQL, but it will look something like the following:
```sh
# For MySQL:
mysqldump --set-gtid-purged --column-statistics=0 -h <host> -u <user> <dbname> -p<password> | gzip > mysqldump.sql.gz

# For PostgreSQL:
PG_PASSWORD=<password> pg_dump -h <host> -p <port> <db_name> | gzip > postgresdump.sql.gz
```
- The gencontroller also uploads some sample files, so make sure to backup your file backend as well. If you deployed an AWS cluster with the tool, and the cluster is using an S3 bucket, then make a copy of it before destroying.

That's it, upload your data somewhere safe and use it in your future tests as a starting point. If you want to know how, check the following settings in the comparison configuration:
- [`DBDumpURL`](config/comparison.md/#DBDumpURL)
- [`S3BucketDumpURI`](config/comparison.md#S3BucketDumpURI)

