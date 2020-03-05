# Mattermost loadtest configuration

## ConnectionConfiguration

### ServerURL

The URL to direct the load. Should be the public facing URL of the Mattermost instance.

### WebsocketURL

In most cases this will be the same URL as above with `http` replaced with `ws` or `https` replaced with `wss`.

### MaxIdleConns

The maximum number of idle connections held open from the loadtest agent to all servers.

### MaxIdleConnsPerHost

The maximum number of idle connections held open from the loadtest agent to any given server.

### IdleConnTimeoutMilliseconds

The number of milliseconds to leave an idle connection open between the loadtest agent an another server.

## LogSettings

### EnableConsole

If true, the server outputs log messages to the console based on ConsoleLevel option.

### ConsoleLevel

Level of detail at which log events are written to the console.

### ConsoleJson

When true, logged events are written in a machine readable JSON format. Otherwise they are printed as plain text.

### EnableFile

When true, logged events are written to the file specified by the `FileLocation` setting.

### FileLevel

Level of detail at which log events are written to log files.

### FileJson

When true, logged events are written in a machine readable JSON format. Otherwise they are printed as plain text.

### FileLocation

The location of the log file.
