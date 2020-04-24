# Deployer Configuration

## ClusterName

*string*

The name of the cluster. This will be prefixed to all resources in AWS that are built with the configuration.

## AppInstanceCount

*int*

The number of Mattermost application instances.

## AppInstanceType

*string*

The type of the EC2 instance of the application server. See type [here](https://aws.amazon.com/ec2/instance-types/). It is recommended to use c5 instances for consistent performance.

## AgentInstanceCount

*int*

The number of load-test agent instances. The first instance will also host the [coordinator](coordinator.md).

## AgentInstanceType

*string*

The type of the EC2 instance of the loadtest agent. See type [here](https://aws.amazon.com/ec2/instance-types/).

## EnableAgentLogs

*bool*

Allows to log the agent service command output (`stdout` & `stderr`) to home directory.

## ProxyInstanceType

*string*

The type of the EC2 instance of the proxy server. See type [here](https://aws.amazon.com/ec2/instance-types/).

## SSHPublicKey

*string*

The path to the SSH public key, this key is used for establishing an SSH connection to the AWS instances.

## DBInstanceCount

*int*

The number of database instances.

## DBInstanceType

*string*

The type of database instance. See types [here](https://aws.amazon.com/rds/instance-types/).

## DBInstanceEngine

*string*

The type of database backend. This can be either `aurora-mysql` or `aurora-postgresql`.

## DBUserName

*string*

The username to connect to the database.

## DBPassword

*string*

The password to connect to the database.

## MattermostDownloadURL

*string*

The URL from where to download Mattermost release. This can also point to a local binary path if the user wants to run a load-test on a custom build. The path should be prefixed with `file://`. In that case, only the binary gets replaced, and the rest of the build comes from the latest stable release.

## MattermostLicenseFile

*string*

The location of the Mattermost Enterprise Edition license file.

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
