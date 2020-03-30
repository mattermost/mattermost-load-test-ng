# Deployer Configuration

## ClusterName

*string*

Name of the cluster. This will be prefixed to all instances in AWS that built with the configuration.

## AppInstanceCount

*int*

The number of Mattermost application instances.

## AgentCount

*int*

The number of load-test agent instances. The first instance will also host the [coordinator](coordinator.md).

## SSHPublicKey

*string*

The path the SSH public key, this key is used for establishing an SSH connection to the AWS instances.

## DBInstanceCount

*int*

The number of Database instances.

## DBInstanceClass

*string*

Type of the Database instance. See types [here](https://aws.amazon.com/rds/instance-types/).

## DBInstanceEngine

*string*

Type of the database backend. This can be either Postgres or MySQL.

## DBUserName

*string*

Username to connect to the database.

## DBPassword

*string*

The password to connect to the database.

## MattermostDownloadURL

*string*

URL from where to download Mattermost release. This can also point to a local binary path if the user wants to run a load-test on a custom build. The path should be prefixed with `file://`. In that case, only the binary gets replaced, and the rest of the build comes from the latest stable release.

## MattermostLicenseFile

*string*

Path to the Mattermost Enterprise Edition license file.

## AdminEmail

*string*

Mattermost instance sysadmin e-mail.

## AdminUsername

*string*

Mattermost instance sysadmin user name.

## AdminPassword

*string*

Mattermost instance sysadmin password.

## GoVersion

*string*

Go version to download for compiling load-test source.

## SourceCodeRef

*string*

load-test-ng head reference.

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

*boolean*

When true, logged events are written in a machine-readable JSON format. Otherwise, they are printed as plain text.

### FileLocation

*string*

The location of the log file.
