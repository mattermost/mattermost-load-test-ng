# Deployer Configuration

## AWSProfile

*string*

AWS profile to use for the deployment. Also used for all AWS CLI commands run locally. See the [AWS docs](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html) for more information.

## AWSRegion

*string*

AWS region to use for the deployment.  See the [AWS docs](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html) for more information.

## AWSAMI

*string*

AWS AMI to use for the deployment. This is the image used for all EC2 instances created by the loadtest tool. See the [AWS AMI](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html) docs for more information. We suggest Ubuntu 20.04 or 22.04. Note, the AMI could change between AWS Regions.



## ClusterName

*string*

The name of the cluster. This will be prefixed to all resources in AWS that are built with the configuration.

## ClusterVpcID

*string*

The ID of the VPC associated to the resources.

**Note**

This setting only affects load-test agent instances. It is meant for pre-deployed environments.

## ClusterSubnetID

*string*

The ID of the subnet associated to the resources.

**Note**

This setting only affects load-test agent instances. It is meant for pre-deployed environments.

## AppInstanceCount

*int*

The number of Mattermost application instances.
This value can be set to zero to enable a load-test agents only deployment.
When this value is greater than one, an S3 bucket is automatically created in the deployment and the server is configured to use it as a file store.

## AppInstanceType

*string*

The type of the EC2 instance of the application server. See type [here](https://aws.amazon.com/ec2/instance-types/). It is recommended to use c5 instances for consistent performance.

## AgentInstanceCount

*int*

The number of load-test agent instances. The first instance will also host the [coordinator](coordinator.md).

## AgentInstanceType

*string*

The type of the EC2 instance of the loadtest agent. See type [here](https://aws.amazon.com/ec2/instance-types/).

## ElasticSearchSettings

### InstanceCount

*int*

Number of ElasticSearch instances to be created. Right now, this config only supports the values `1` or `0`.

### InstanceType

*string*

The type of instance for the Elasticsearch service. See type [here](https://aws.amazon.com/ec2/instance-types/).

### Version

*float64*

Version of Elasticsearch to be deployed. See [here](https://aws.amazon.com/elasticsearch-service/faqs/?nc=sn&loc=6) the supported versions.

## VpcID

*string*

Id for the VPC that is going to be associated with the Elasticsearch created instance. You can get the VPC Id [here](https://console.aws.amazon.com/vpc/).

This ID is mandatory is you're going to instanciate an ES service in your cluster.

### CreateRole

*bool*

Elasticsearch depends on the `AWSServiceRoleForAmazonElasticsearchService` service-linked role. This role is unique and shared by all users of the account so if it's already created you can't create it again and you'll receive an error.

You can check if the role is already created [here](https://console.aws.amazon.com/iam/home#roles) and if it isn't created set this property to true.

## JobServerSettings

### InstanceCount

*int*

Number of instances to be created. Supported values are `1` or `0`. Once a job server is deployed, all of the periodic jobs will run on this instance.

### InstanceType

*string*

The type of EC2 instance for the Job Server. See type [here](https://aws.amazon.com/ec2/instance-types/).

## EnableAgentFullLogs

*bool*

Allows to log the agent service command output (`stdout` & `stderr`) to home directory.

## ProxyInstanceType

*string*

The type of the EC2 instance of the proxy server. See type [here](https://aws.amazon.com/ec2/instance-types/).

## SSHPublicKey

*string*

The path to the SSH public key, this key is used for establishing an SSH connection to the AWS instances.

## TerraformDBSettings

### InstanceCount

*int*

The number of database instances.

### InstanceType

*string*

The type of database instance. See types [here](https://aws.amazon.com/rds/instance-types/).

### InstanceEngine

*string*

The type of database backend. This can be either `aurora-mysql` or `aurora-postgresql`.

### UserName

*string*

The username to connect to the database.

### Password

*string*

The password to connect to the database.

### EnablePerformanceInsights

*bool*

If set to true enables performance insights for the created DB instances.

## ExternalDBSettings

### DriverName

*string*

The Mattermost driver to use to access to the external database.

### DataSource

*string*

The dsn of the external database.

### DataSourceReplicas

*[]string*

The list of dsn for external database read replicas

### DataSourceSearchReplicas

*[]string*

The list of dsn for external database search replicas

## MattermostDownloadURL

*string*

The URL from where to download Mattermost release. This can also point to a local binary path if the user wants to run a load-test on a custom server build.  
The path should be prefixed with `file://` and point to the binary of the server (e.g. `file:///home/user/go/src/github.com/mattermost/mattermost/server/bin/mattermost`).  
Only the binary gets replaced, and the rest of the build comes from the latest stable release.

## MattermostLicenseFile

*string*

The location of the Mattermost Enterprise Edition license file.

## MattermostConfigPatchFile

*string*

An optional path to a partial Mattermost config file to be applied as patch during app server deployment.

## AdminEmail

*string*

The e-mail that will be used when creating a sysadmin user during the deployment process.

## AdminUsername

*string*

The user name that will be used when creating a sysadmin user during the deployment process.

## AdminPassword

*string*

The password that will be used when creating a sysadmin user during the deployment process.

## LoadTestDownloadURL

*string*

The URL from where to download load-test-ng binaries. This can also point to a local package if the user wants to run a load-test with a custom version of load-test-ng binaries. The path should be prefixed with `file://` to use the local package. Either case the configuration files in the package will be updated in the deployment process.

## LogSettings

### EnableConsole

*bool*

If true, the server outputs log messages to the console based on the ConsoleLevel option.

### ConsoleLevel

*string*

Level of detail at which log events are written to the console.

### ConsoleJson

*bool*

When true, logged events are written in a machine-readable JSON format. Otherwise, they are printed as plain text.

### EnableFile

*bool*

When true, logged events are written to the file specified by the `FileLocation` setting.

### FileLevel

*string*

Level of detail at which log events are written to log files.

### FileJson

*bool*

When true, logged events are written in a machine-readable JSON format. Otherwise, they are printed as plain text.

### FileLocation

*string*

The location of the log file.

### EnableColor

*bool*

When true enables colored output.

## Report

### Label

*string*

The label to filter Prometheus queries.

### GraphQueries

*[]GraphQuery*

GraphQuery contains the query to be executed against a Prometheus instance to gather data for reports.

#### Name

*string*

A friendly name for the graph.

#### Query

*string*

The Prometheus query to run.

## TerraformStateDir

*string*

The directory under which Terraform-related files are stored. If the directory does not exist, it will be created when running the first command that needs it, defaulting to `/var/lib/mattermost-load-test-ng`. You'll need root permissions to create that specific directory, so you may want to change this setting to something like `/home/youruser/.loadtest`.

## S3BucketDumpURI

*string*

URI pointing to an S3 bucket: something of the form `s3://bucket-name/optional-subdir`.
The contents of this bucket will be copied to the bucket created in the deployment, using `aws s3 cp`. This command is ran locally, so having the AWS CLI installed is required.
If no bucket is created in the deployment (see [`AppInstanceCount`](#AppInstanceCount) for more information), this value is ignored.
If a bucket is created in the deployment but this value is empty, the created bucket will not be pre-populated with any data.

## S3BucketDumpURI

*string*

An optional URI to a MM server database dump file
to be loaded before running the load-test.
The file is expected to be gzip compressed.
This can also point to a local file if prefixed with "file://".
In such case, the dump file will be uploaded to the app servers.

## PermalinkIPsToReplace

*string*

An optional list of IPs present in the posts from the DB dump
that contain permalinks to other posts. These IPs are replaced,
when ingesting the dump into the database, in every post that
uses them with the public IP of the first app instance, so that
the permalinks are valid in the new deployment.
