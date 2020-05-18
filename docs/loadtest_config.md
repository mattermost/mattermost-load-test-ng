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

## InstanceConfiguration

### NumTeams

*int*

The number of teams the target Mattermost instance should have.  
These will be created during the `init` process.

## UsersConfiguration

### InitialActiveUsers

*int*

The amount of active users to run when the load-test starts.

### MaxActiveUsers

*int*

The maximum amount of concurrently active users the load-test agent will run.

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
