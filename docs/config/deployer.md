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

The type of instance for the Elasticsearch service. Only AWS OpenSearch instances are allowed. Instances that do not support EBS storage volumes are not allowed. Check [AWS documentation](https://docs.aws.amazon.com/opensearch-service/latest/developerguide/supported-instance-types.html) for the full list of instance types.

### Version

*string*

Version of Elasticsearch to be deployed. Deployments only support AWS OpenSearch versions compatible with ElasticSearch, up to and including ElasticSearch v7.10.0; i.e., the ones prefixed by `Elasticsearch_.`. Check [AWS documentation](https://aws.amazon.com/opensearch-service/faqs/) to learn more about the versions and the [`aws` Terraform provider documentation](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/opensearch_domain#engine_version) to learn more about the specific string used.

## VpcID

*string*

Id for the VPC that is going to be associated with the Elasticsearch created instance. You can get the VPC Id [here](https://console.aws.amazon.com/vpc/).

This ID is mandatory is you're going to instantiate an ES service in your cluster.

### CreateRole

*bool*

Elasticsearch depends on the `AWSServiceRoleForAmazonElasticsearchService` service-linked role. This role is unique and shared by all users of the account so if it's already created you can't create it again and you'll receive an error.

You can check if the role is already created [here](https://console.aws.amazon.com/iam/home#roles) and if it isn't created set this property to true.

### SnapshotRepository

*string*

If you want to deploy an already indexed database, you need to provide both the name of the repository where the snapshot lives and the snapshot's name. `SnapshotRepository` is the name of the repository.

### SnapshotName

*string*

If you want to deploy an already indexed database, you need to provide both the name of the repository where the snapshot lives and the snapshot's name. `SnapshotName` is the name of the snapshot itself.

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

### ClusterIdentifier

*string*

The name of the existing cluster to attach to. If this is set, then the `DBDumpURI` does not have any effect. This string should be a restored AWS Aurora backup cluster.

### DBName

*string*

The name of the database. This is meant to be used in conjunction with `ClusterIdentifier`, and its value should be the name of the database in such cluster.

However, it can be used by itself to hardcode the name of the database that will otherwise be created. If `DBName` is not specified, and a brand new database is created for the deployment, its name will equal `${ClusterName}db`.

### DBParameters

*[]DBParameter*

A list of DB specific parameters to use for the created instance.
Detailed information on these values can be found [here](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_WorkingWithParamGroups.html).

Example:

```
    "DBParameters": [
      {
        "Name": "innodb_buffer_pool_size",
        "Value": "2147483648",
        "ApplyMethod": "pending-reboot"
      }
    ]
```

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

## ExternalBucketSettings

### AmazonS3AccessKeyId

*string*

The access key id of the external bucket.

### AmazonS3SecretAccessKey

*string*

The secret access key of the external bucket.

### AmazonS3Bucket

*string*

The bucket name.

### AmazonS3PathPrefix

*string*

The path prefix.

### AmazonS3Region

*string*

The AWS region.

### AmazonS3Endpoint

*string*

The S3 endpoint.

### AmazonS3SSL

*bool*

Whether to use SSL or not.

### AmazonS3SignV2

*bool*

Whether to use the v2 protocol while signing or not.

### AmazonS3SSE

*bool*

Whether to use SSE or not.

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

## DBDumpURI

*string*

An optional URI to a MM server database dump file
to be loaded before running the load-test.
The file is expected to be gzip compressed.
This can also point to a local file if prefixed with "file://".
In such case, the dump file will be uploaded to the app servers.

## SiteURL

*string*

The name of a host that will be used for two purposes:
- It will override the server's site URL.
- It will populate a new entry in the /etc/hosts file of the app nodes, so that it points to the proxy private IP or, if there's no proxy, to the current app node.
This config is used for tests that require an existing database dump that contains permalinks. These permalinks point to a specific hostname. Without this setting, that hostname is not known by the nodes of a new deployment and the permalinks cannot be resolved.

## UsersFilePath

*string*

The path to a file containing a list of credentials for the controllers to use. If present, it is used to automatically upload it to the agents and override the agent's config's own [`UsersFilePath`](config.md/#UsersFilePath).

## PyroscopeSettings

### EnableAppProfiling

*bool*

Enable continuous profiling of all the app instances.

### EnableAgentProfiling

*bool*

Enable continuous profiling of all the agent instances.
