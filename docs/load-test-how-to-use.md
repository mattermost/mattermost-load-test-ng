# Load test notes

## Prerequisites for load-testing a new feature

 - Build and test a feature in a mattermost-server branch.
 - Scan through markdown docs in [mattermost-load-test-ng](https://github.com/mattermost/mattermost-load-test-ng). Make sure you get an idea of what a coordinator, agent, controller, bounded and unbounded load-test is, and why do we need metrics collection(a deployment of prometheus) in this setup.

## Steps to load test the feature

### Brief summary of tasks

This would include
 - Writing new load testing actions to mattermost-load-test-ng
 - Testing the changes locally.
 - Testing the changes in terraform: its purpose is to load-test under a heavier load/dataset.
 - Analyse load-test results
 - Getting the changes merged to load-test repository, so the release-manager can test the same changes for unexpected behaviors during the upcoming releases.

### Detailed summary of tasks

#### Writing new load testing code

 - Go through [coverage.md](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/coverage.md#implementation-overview), make a list of changes needed to load-test the new feature.
 - Optionally check out [this video walkthrough](https://drive.google.com/file/d/1l462zMdANwCRXUtj7nnHv2CX_6BiINHl/view) by @claudio.costa
 - Make the necessary changes in `loadtest/`

#### Testing changes locally

 - Go through [local_loadtest.md](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/local_loadtest.md)
 - Some additional information on the above document
    - In step where configs are copied from samples, `simplecontroller.json` doesn't have to be generated since by default the `config.json` uses a `simulative` controller. <!-- docfix -->
    - updates to `config.json`
        - Make sure to change `ConnectionConfiguration` section in config.json according to the local deployment of mattermost-server.
        - `InstanceConfiguration` refers to [doc](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/loadtest_config.md#instanceconfiguration), this is used by `init` command later to initially populate mattermost database.
    - increase frequency of the new action, so its easier to debug while running locally. [sample](https://github.com/mattermost/mattermost-load-test-ng/blob/8faa4dfb485dace3bd65908c0d3d98979b7dfd17/loadtest/control/simulcontroller/controller.go#L227)
    - Expect some failures in [running-a-basic-loadtest](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/local_loadtest.md#running-a-basic-load-test), if one sees > 10-15 error logs, there's a [troubleshooting guide](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/faq.md#troubleshooting).
    - check `ltagent.log` in `mattermost-load-test-ng` directory, and server logs for details on errors, if any.
    - Further sections of the document i.e [using load-test agent API server](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/local_loadtest.md#running-a-load-test-through-the-load-test-agent-api-server) and [using load-test coordinator](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/local_loadtest.md#running-a-load-test-through-the-coordinator) are highly recommended to go through, since the Terraform deployment uses the latter method to execute the load-tests and it'd be easier to debug the setup if the developer understands these underlying principles.


#### Testing changes in terraform

Even without the framework, a general load-test workflow in the cloud will be similar to the following.

 - Create a database (to be used by mattermost servers)
 - Create deployments of mattermost servers - let's call them app servers for convenience.
 - Create 'agent' machines to ping app-servers using [controllers](https://github.com/mattermost/mattermost-load-test-ng/tree/61c44f35224b76d3098199b0cd2b67db2222b549/loadtest/control)
 - Populate database, and start app, agents (i.e loadtest)
 - Collect metrics from app, and agent deployments to either,
    - manage an unbounded load test
    - analyse api performance after the load-test completes.

Loadtest instances created with this framework achieve the same goals as mentioned above, only some of the things like creating a deployment, running a loadtest, etc. are automated.

Following are some steps to load-test a new feature in production, after testing new actions locally.

 - Go through [terraform_loadtest.md](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/terraform_loadtest.md)
 - Some additional information on the above document
    - AWS credentials are to be fetched from [onelogin](https://mattermost.onelogin.com/) and [added to terraform config](https://registry.terraform.io/providers/hashicorp/aws/latest/docs). <!-- docfix -->
    - Enterprise license is to be fetched from [Developers:private](https://community.mattermost.com/private-core/channels/core-developers) <!-- docfix;  https://github.com/mattermost/enterprise/pull/1208 -->
    - When performing [this](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/terraform_loadtest.md#copy-and-modify-the-required-configuration) step, edit `MattermostLicenseFile` value to the path containing the license.
        - The fields `MattermostDownloadURL` and `LoadTestDownloadURL` point to the latest mattermost-server, and load-test package respectively to be used in the load-test.
        - That's the default option, when there's unmerged changes to `mattermost-server`, or `mattermost-load-test-ng`
            - `make build-linux` in `mattermost-server` directory, change `MattermostDownloadURL` value to the path containing mattermost executable. For example `file:///somepath/mattermost-server/bin/linux_amd64/mattermost`
            - `make package` in `mattermost-load-test-ng` directory, change `LoadTestDownloadURL` value to the path containing gzip of load-test package. For example `file:///somepath/mattermost-load-test-ng/dist/v1.5.0-8-gd4f18cf/mattermost-load-test-ng-v1.5.0-8-gd4f18cf-linux-amd64.tar.gz`
    - Edit `SSHPublicKey` in deployer.json after setting up ssh.
    - `go run ./cmd/ltctl deployment create`
        <!-- devfix; anything to do with deployment spawns zombie processes if the command is cancelled with a shell interruption. Further ltctl deployment commands don't work, so one has to restart the computer before starting again. Haven't tried to kill the processes manually.-->
        - Limit operations of `deployment` to a single shell window.
        - If the deployment gets stuck, check for `ps -ef | grep terraform`, if there are running processes, restart the computer and start again.
        - [Terraform actions are idempotent](https://community.mattermost.com/core/pl/jtebkneah3futd1y7pj8y9nrqy), so one would rarely have to [destroy](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/terraform_loadtest.md#destroy-the-current-deployment) the deployment, if things go wrong while creating resources.
    - Once the deployment gets created successfully, stdout will have information on server addresses for app, agent, coordinator, and Grafana deployments.
        - Open the mattermost URL in browser to check if the app is working as required.
        - At this point, one might check the server logs by [ssh-ing](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/terraform_loadtest.md#ssh-access-to-the-terraformed-hosts) into the app instance. Once in there, open `/opt/mattermost/logs/mattermost.log`.
    - Gearing up to start the load-test
        - Use `agents'` URL and Prometheus URL in the `coordinator.json` file generated [here](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/terraform_loadtest.md#copy-default-config)
        - Change `ConnectionConfiguration` in the `config.json` generated [here](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/terraform_loadtest.md#copy-default-config-1)
        - Configure `InstanceConfiguration` in the same `config.json` (which as mentioned earlier, creates the seeds mm-server's database with required data for the loadtest). Note that a heavier config with `NumPosts` would take a very long time get seeded. Please refer to the [NB](/#nb) section below to manually seed the database from a backup, in order to bypass 'data-generation.
    
    - Start the load test with `go run ./cmd/ltctl loadtest start`
        - Once a loadtest is running, its status can be checked with `go run ./cmd/ltctl loadtest status`.
        - `ssh` into one of the agent machines, and `cat ~/mattermost-load-test-ng/logs/ltagent.log` to verify the load-test is working without errors.
        - Open the Grafana deployment with the URL from `go run ./cmd/ltctl deployment info`. 
        - It takes some time for the deployment to stabilize, i.e while the loadtest tool connects `MaxActiveUsers` count of users to app, there might be a big count of HTTP 4xx errors in this duration.
            
            A sample error count vs time graph: ![error-vs-time](https://i.imgur.com/RSH1Szl.png)
    
    - "My load test is running, now what?"
        - If it's a [bounded](https://github.com/mattermost/mattermost-load-test-ng/blob/497554e376ef23d548947bf331c8bdce6ce453d6/docs/faq.md#what-is-a-bounded-load-test) loadtest, it has to be manually stopped with `go run ./cmd/ltctl loadtest stop` [after an hour](https://community.mattermost.com/core/pl/45woi49ru7yrj8r8upzaqhog3a)
        - If it's an [unbounded](https://github.com/mattermost/mattermost-load-test-ng/blob/497554e376ef23d548947bf331c8bdce6ce453d6/docs/faq.md#what-is-an-unbounded-load-test) loadtest, the load-test will finish with a stdout listing the maximum concurrent users the deployment supports. The load-test status check command will say status as `Done` when it's complete.

#### Analysing load-test results

"My load tests ran successfully, what to make of it?"
    - In case of unbounded load-tests, when they finish, `go run ./cmd/ltctl loadtest status` would give you a count of maximum concurrent users which is a metric to compare the performance of that version of mattermost-server.
    - In case of two bounded loadtests with same `MaxConcurrentUsers` count, one can [generate a report](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/compare.md) comparing performance of various server metrics. 
    <!-- devfix: should we allow comparision of specific store and api functions? -->
    - In case of both bounded and unbounded loadtests, one can create grafana dashboards to analyze performances of the new features by filtering api metrics relevant to the new handler. Here's an example.
        ![sample-dashboard-creation](https://i.imgur.com/zzRfh8b.png)


#### Creating a PR

After all the code changes,
 - [Add tests](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/coverage.md#testing) if required. 
 - `make check-style`
 - `go get -u github.com/mattermost/mattermost-server/v6@<commit-hash-in-master>` && `go mod tidy`
 - Make the PR

#### NB:

 - **For seeding the database manually** :
    - [InstanceConfiguration](https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/loadtest_config.md#instanceconfiguration) section would be as minimal as possible to reduce `db init` time.
    - Message in [Developers:Performance](https://community.mattermost.com/core/channels/developers-performance) for a migration file.
    - `ssh` into the app machine, and `psql` into the connected database.
    - Drop all tables, log out of psql. Run the migration, which might take a while. <!-- This might not be necessary, I faced some primary id collision issue while restoring from sql. If someone can confirm it's redundance, we'll delete this instruction.-->
    - Now, the app service needs to be restarted so the server can run the necessary migrations.
    - `ssh` into app-instance(s) and run `sudo systemctl restart mattermost && until $(curl -sSf http://localhost:8065 --output /dev/null); do sleep 1; done;`
 - **If the feature is behind a feature flag**: [link to Claudio's message to add environment variables to app-service](https://community.mattermost.com/core/pl/honr5se45f8etpwexgmi9qbe5a)