# LoadTest Configuration

## ConnectionConfiguration

### ServerURL

*string*

The URL to direct the load. Should be the public facing URL of the target Mattermost instance.

### WebsocketURL

*string*

The URL to the WebSocket endpoint the users will connect to.  
In most cases this will be the same as `ServerURL` with `http` replaced with `ws` or `https` replaced with `wss`.

### AdminEmail

*string*

The e-mail for the system admin of the target Mattermost instance.

### AdminPassword

*string*

The password for the system admin of the target Mattermost instance.

## UserControllerConfiguration

### Type

*string*

The type of [`UserController`](controllers.md) that will be used to drive the users.

Possible values:
- `simple` - to use [`SimpleController`](controllers.md#simplecontroller)
- `simulative`  - to use [`SimulController`](controllers.md#simulcontroller)
- `noop` - to use [`NoopController`](controllers.md#noopcontroller)
- `generative` - to use [`GenController`](controllers.md#gencontroller)

### RatesDistribution

*[]struct{
  Rate float64
  Percentage float64
}*

The distribution of action rates for running controllers.

Rate is a multiplier that will affect the speed at which user actions are executed by the `UserController`.

A rate < 1.0 will run actions at a faster pace.   
A rate == 1.0 will run actions at the default pace.    
A rate > 1.0 will run actions at a slower pace.  

Percentage is the percentage of controllers that should run with the specified rate.

### ServerVersion

*string*

An optional MM server version to use when running actions (e.g. `5.30.0`).
This value overrides the actual server version. If left empty, the one returned by the server is used instead.

## InstanceConfiguration

### NumTeams

*int*

The number of teams the target Mattermost instance should have.  
These will be created during the `init` process.

### NumChannels

*int*

The number of channels the target Mattermost instance should have.  
These will be created during the `init` process.

### NumPosts

*int*

The number of posts the target Mattermost instance should have.  
These will be created during the `init` process.

### NumReactions

*int*

The number of reactions the target Mattermost instance should have.  
These will be created during the `init` process.

### NumAdmins

*int*

The number of admins the target Mattermost instance should have.  
These will be created during the `init` process.
	
## UsersConfiguration

### InitialActiveUsers

*int*

The amount of active users to run when the load-test starts.

### UsersFilePath

*string*

The path to the file which contains a list of user email and passwords that will be used by the tool if set. Each line should be for a user containing an email and password separated by space. The number of lines in the file should be at least equal to MaxActiveUsers.

### MaxActiveUsers

*int*

The maximum amount of concurrently active users the load-test agent will run.

### MaxActiveBrowserUsers

*int*

The maximum amount of concurrently active browser users per instance the load-test agent will run.

### PercentOfUsersAreAdmin

*float*

The percentage of users generated that will be system admins.

## LogSettings

### EnableConsole

*bool*

If true, the server outputs log messages to the console based on ConsoleLevel option.

### ConsoleLevel

*string*

Level of detail at which log events are written to the console.

### ConsoleJson

*bool*

When true, logged events are written in a machine readable JSON format. Otherwise they are printed as plain text.

### EnableFile

*bool*

When true, logged events are written to the file specified by the `FileLocation` setting.

### FileLevel

*string*

Level of detail at which log events are written to log files.

### FileJson

*bool*

When true, logged events are written in a machine readable JSON format. Otherwise they are printed as plain text.

### FileLocation

*string*

The location of the log file.

### EnableColor

*bool*

When true enables colored output.

## BrowserLogSettings

### EnableConsole

*bool*

When true, the browser server outputs log messages to the console based on ConsoleLevel option.

### ConsoleLevel

*string*

Level of detail at which log events are written to the console.

Possible values (in order of decreasing verbosity, these are case sensitive):
- `trace`
- `debug`
- `info`
- `warn`
- `error`
- `fatal`

### EnableFile

*bool*

When true, the browser server outputs log messages to the file specified by the `FileLocation` setting.

### FileLevel

*string*

Level of detail at which log events are written to log files. Exactly same as `ConsoleLevel` as mentioned above.

Possible values (in order of decreasing verbosity, these are case sensitive):
- `trace`
- `debug`
- `info`
- `warn`
- `error`
- `fatal`

When both `EnableConsole` and `EnableFile` are true, the logs are written asynchronously to reduce overhead.

### FileLocation

*string*

The location of the log file.
