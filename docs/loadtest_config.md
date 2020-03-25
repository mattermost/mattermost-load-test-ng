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

### MaxIdleConns

*integer*

The maximum number of idle connections held open from the load-test agent to all servers.

### MaxIdleConnsPerHost

*integer*

The maximum number of idle connections held open from the load-test agent to any given server.

### IdleConnTimeoutMilliseconds

*integer*

The number of milliseconds to leave an idle connection open between the load-test agent an another server.

## UserControllerConfiguration

### Type

*string*

The type of [`UserController`](controllers.md) that will be used to drive the users.

Possible values:
- `simple` - to use [`SimpleController`](controllers.md#simplecontroller)
- `simulative`  - to use [`SimulController`](controllers.md#simulcontroller)
- `noop` - to use [`NoopController`](controllers.md#noopcontroller)

### Rate

*number*

A rate multiplier that will affect the speed at which user actions are executed by the `UserController`.

A rate < 1.0 will run actions at a faster pace.  
A rate == 1.0 will run actions at the default pace.  
A rate > 1.0 will run actions at a slower pace.  

## InstanceConfiguration

### NumTeams

*integer*

The number of teams the target Mattermost instance should have.
These will be created during the `init` process.

## UsersConfiguration

### InitialActiveUsers

*integer*

The amount of active users to run when the load-test starts.

### MaxActiveUsers

*integer*

The maximum amount of concurrently active users the load-test agent will run.

## LogSettings

### EnableConsole

*boolean*

If true, the server outputs log messages to the console based on ConsoleLevel option.

### ConsoleLevel

*string*

Level of detail at which log events are written to the console.

### ConsoleJson

*boolean*

When true, logged events are written in a machine readable JSON format. Otherwise they are printed as plain text.

### EnableFile

*boolean*

When true, logged events are written to the file specified by the `FileLocation` setting.

### FileLevel

*string*

Level of detail at which log events are written to log files.

### FileJson

*boolean*

When true, logged events are written in a machine readable JSON format. Otherwise they are printed as plain text.

### FileLocation

*string*

The location of the log file.
