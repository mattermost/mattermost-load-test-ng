# Load test notes

## Prerequisites for load-testing a new feature

 - Build and test a feature in a mattermost-server branch.
 - Scan through markdown docs in [mattermost-load-test-ng](https://github.com/mattermost/mattermost-load-test-ng/tree/master/docs). Make sure you get an idea of what a coordinator, agent, and controller are, what bounded and unbounded load-tests are, and why metrics collection (a deployment of prometheus) is needed in this setup.

## Steps to load test the feature

### Brief summary of tasks

The steps to load test a feature include:
 - Writing new load testing actions to mattermost-load-test-ng.
 - Testing the changes locally.
 - Testing the changes in terraform: its purpose is to load-test with a larger dataset.
 - Analyse load-test results.
 - Getting the changes merged to the load-test repository.

### Detailed summary of tasks

#### Writing new load testing code

 - Go through [coverage.md](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/coverage.md#implementation-overview), and make a list of changes needed to load-test the new feature.
 - Optionally check out [this video walkthrough](https://drive.google.com/file/d/1l462zMdANwCRXUtj7nnHv2CX_6BiINHl/view) by @streamer45 
 - Make the necessary changes in `loadtest/`.

#### Testing changes locally

 - Go through [local_loadtest.md](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/local_loadtest.md).
 - Some additional information on the above document
    - Updates to `config.json`
        - Make sure to change `ConnectionConfiguration` section in `config.json` according to the local deployment of mattermost-server.
        - For `InstanceConfiguration`, see [this doc](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/loadtest_config.md#instanceconfiguration). This configuration setting is used by the `init` command to initially populate the mattermost database.
    - Increase the frequency of the new action, so it's easier to debug while running locally. [sample](https://github.com/mattermost/mattermost-load-test-ng/blob/8faa4dfb485dace3bd65908c0d3d98979b7dfd17/loadtest/control/simulcontroller/controller.go#L227)
    - If you see errors, there's a [troubleshooting guide](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/faq.md#troubleshooting) you can reference to resolve the issues.
    - Check `ltagent.log` located in the `mattermost-load-test-ng` directory, and the server logs for details on errors, if any.
    - We highly recommend becoming familiar with additional sections of documentation, including [using load-test agent API server](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/local_loadtest.md#running-a-load-test-through-the-load-test-agent-api-server) and [using load-test coordinator](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/local_loadtest.md#running-a-load-test-through-the-coordinator), since the Terraform deployment uses the latter method to execute the load-tests, and it'll be easier to debug the setup if you understand these underlying concepts and principles.


#### Testing changes in terraform

Loadtest instances created with this framework achieve the same goals as mentioned above, only some of the things like creating a deployment, running a loadtest, etc. are automated.

The steps to load-test a new feature in production, after testing new actions locally, includes the following:

 - Review the [terraform_loadtest.md](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/terraform_loadtest.md) documentation.
 - Some additional information on the above document
    - When performing [this](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/terraform_loadtest.md#copy-and-modify-the-required-configuration) step, edit `MattermostLicenseFile` value to the path containing the license.
        - The fields `MattermostDownloadURL` and `LoadTestDownloadURL` point to the latest `mattermost-server`, and `load-test package`, to be used in the load-test. These are the default options.
        - When there's unmerged changes to `mattermost-server`, or `mattermost-load-test-ng`, you need to update these values:
            - Run `make build-linux` in the `mattermost-server` directory, change the `MattermostDownloadURL` value to the path containing the Mattermost executable. For example `file:///somepath/mattermost-server/bin/linux_amd64/mattermost`.
            - Run `make package` in the `mattermost-load-test-ng` directory, change `LoadTestDownloadURL` value to the path containing the gzip of the load-test package. For example `file:///somepath/mattermost-load-test-ng/dist/v1.5.0-8-gd4f18cf/mattermost-load-test-ng-v1.5.0-8-gd4f18cf-linux-amd64.tar.gz`.
    - Edit `SSHPublicKey` in `deployer.json` after setting up ssh.
    - `go run ./cmd/ltctl deployment create`
        - If the deployment gets stuck, [check if AWS credentials are expired, and add new ones.](https://community.mattermost.com/core/pl/weau31yyp38btddryjuxbsnh1r).
        - [Terraform actions are idempotent](https://community.mattermost.com/core/pl/jtebkneah3futd1y7pj8y9nrqy), so you rarely have to [destroy](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/terraform_loadtest.md#destroy-the-current-deployment) the deployment, if things go wrong while creating resources.
    - Once the deployment is successful, stdout will contain information on server addresses for app server, agent, coordinator, and Grafana deployments.
        - Open the Mattermost URL in browser to confirm the app is working.
        - At this point, you can check the server logs by [ssh-ing](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/terraform_loadtest.md#ssh-access-to-the-terraformed-hosts) into the app instance. Once there, open `/opt/mattermost/logs/mattermost.log`.
    - Gearing up to start the load-test:
        - Use `agents'` URL and Prometheus URL in the `coordinator.json` file generated [here](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/terraform_loadtest.md#copy-default-config).
        - Change `ConnectionConfiguration` in the `config.json` generated [here](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/terraform_loadtest.md#copy-default-config-1).
        - Configure `InstanceConfiguration` in the same `config.json` file (which, as mentioned earlier, populates the Mattermost server's database with required data for the load test). Note that a heavier config with `NumPosts` would take a very long time get populated. Please refer to the [NB](/#nb) section below to manually populate the database from a backup, in order to bypass 'data-generation`.
    
    - Start the load test with `go run ./cmd/ltctl loadtest start`.
        - Once a loadtest is running, its status can be checked with `go run ./cmd/ltctl loadtest status`.
        - `ssh` into one of the agent machines, and `cat ~/mattermost-load-test-ng/logs/ltagent.log` to verify the load-test is working without errors.
        - Open the Grafana deployment with the URL from `go run ./cmd/ltctl deployment info`. 
        - It takes some time for the deployment to stabilize. Until the loadtest tool connects the `MaxActiveUsers` count of users to the app, there might be a big count of HTTP 4xx errors during this time.
            
            A sample error count vs time graph: ![error-vs-time](https://i.imgur.com/RSH1Szl.png)
    
    - "My load test is running, now what?"
        - If it's a [bounded](https://github.com/mattermost/mattermost-load-test-ng/blob/497554e376ef23d548947bf331c8bdce6ce453d6/docs/faq.md#what-is-a-bounded-load-test) loadtest, it has to be manually stopped with `go run ./cmd/ltctl loadtest stop` [one hour after the number of users connected stabilized](https://community.mattermost.com/core/pl/45woi49ru7yrj8r8upzaqhog3a).
        - If it's an [unbounded](https://github.com/mattermost/mattermost-load-test-ng/blob/497554e376ef23d548947bf331c8bdce6ce453d6/docs/faq.md#what-is-an-unbounded-load-test) loadtest, the load-test will finish with a stdout listing the maximum concurrent users the deployment supports. The load-test status check command will say status as `Done` when it's complete.

#### Analysing load-test results

"My load tests ran successfully, what to make of it?"
    - In case of unbounded load-tests, when they finish, `go run ./cmd/ltctl loadtest status` would give you a count of maximum concurrent users which is a metric to compare the performance of that version of mattermost-server.
    - In case of two bounded loadtests with same `MaxConcurrentUsers` count, one can [generate a report](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/compare.md) comparing performance of various server metrics. 
    - In case of both bounded and unbounded loadtests, one can create Grafana dashboards to analyze the performance of the new features by filtering API metrics to only what's relevant to the new handler. Here's an example.
        ![sample-dashboard-creation](https://i.imgur.com/zzRfh8b.png)


#### Testing changes in cloud without terraform
Even without the framework, a general load-test workflow in the cloud will be similar to the following:

 - Create a database (to be used by Mattermost servers).
 - Deploy Mattermost servers - let's call them app servers for convenience.
 - Create 'agent' machines to ping app servers using [controllers](https://github.com/mattermost/mattermost-load-test-ng/tree/61c44f35224b76d3098199b0cd2b67db2222b549/loadtest/control).
 - Populate a database, then start app servers and agents (i.e., loadtest).
 - Collect metrics from app servers and agent deployments to either manage an unbounded load test or analyze API performance after the load-test completes.


#### Creating a PR

After all the code changes:
 - [Add tests](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/coverage.md#testing) if required. 
 - Run `make check-style`.
 - `go get -u github.com/mattermost/mattermost-server/v6@<commit-hash-in-master>` && `go mod tidy`
 - Create the PR.

#### Note:

 - **For populating the database manually** :
    - [InstanceConfiguration](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/loadtest_config.md#instanceconfiguration) section would be as minimal as possible to reduce `db init` time.
    - Post a message in the [Developers:Performance](https://community.mattermost.com/core/channels/developers-performance) channel to request a migration file.
    - If you're using [comparison](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/comparison.md), there is [an option to load db from a backup](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/comparison_config.md#dbdumpurl). If not,
        - `ssh` into the app machine, and `psql` into the connected database. (Link to the database can be obtained with `ltctl deployment info`)
        - Drop and recreate the target database. Restore backup data with `zcat <backupfile> | psql <dsn>`.
        - Now, the app service needs to be restarted so the server can run the necessary migrations.
        - Run `sudo systemctl restart mattermost && until $(curl -sSf http://localhost:8065 --output /dev/null); do sleep 1; done;`
 - **If the feature is behind a feature flag**: [see Claudio's message to add environment variables to app-service](https://community.mattermost.com/core/pl/honr5se45f8etpwexgmi9qbe5a).