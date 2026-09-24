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

## AccessControlConfiguration

Settings to load-test channel read and write access with Attribute-Based Access Control (ABAC) permission policies. See [Load-testing channel access policies](../abac_loadtest.md) for an overview.

The target instance needs an Enterprise Advanced license, the `SessionAttributes` feature flag and `AccessControlSettings.EnableAttributeBasedAccessControl` enabled. The deployer does this when [`AccessControlSettings.Enable`](deployer.md#accesscontrolsettings) is set.

### Enable

*bool*

When true, every agent sets up the configured attributes and policies on the target instance when it's created (the setup is idempotent), assigns the user attribute values to the simulated users when they log in, and makes the simulated users send their session attributes on every request.

It has no effect on the `generative` controller, so the data generated by `ltagent init` is not restricted by the policies.

### UserAgent

*string*

The `User-Agent` header the simulated users send when `Enable` is true. The server only accepts client-provided session attributes from the desktop and mobile apps, which it recognizes by their `User-Agent`. The default is a desktop app one.

### UserAttributes

*[]struct{
  Name string
  Type string
  Values []struct{
    Value string
    Weight float64
  }
}*

The user attributes (custom profile attributes) to create. They are created as admin managed so that they can be used in policies. At most 20 are supported.

- `Name`: the name referenced in policy expressions as `user.attributes.<Name>`. Must be a valid identifier.
- `Type`: `select` (default) or `text`. The values of a `select` attribute become the options of the field; missing options are added to an existing field.
- `Values`: the values users are assigned, each with the probability (`Weight`) of being assigned. Weights must sum to 1.

Each user gets the same values regardless of the agent simulating it, since they are picked deterministically from the user's email.

### SessionAttributes

*[]struct{
  Name string
  TTLSeconds int
  GracePeriodSeconds int
  Values []struct{
    Value string
    Weight float64
  }
}*

The client-provided session attributes to enable on the server and send through the `X-MM-Session-Attributes` header. Server-computed attributes (`ip_address` and `user_agent_*`) are not supported, since the clients cannot set them.

- `Name`: the name referenced in policy expressions as `user.session.<Name>`. One of `client_ip_address`, `network_interface_type`, `vpn_active`, `ssid`, `mdm_enrolled`, `jailbreak_detected`, `os_platform`, `os_version`, `client_version`, `client_device_id`, `hardware_id`, `server_fqdn` and `client_fqdn`. The server only accepts `jailbreak_detected` and `client_device_id` from mobile clients, and `hardware_id` and `client_fqdn` from desktop clients.
- `TTLSeconds`, `GracePeriodSeconds`: when greater than zero, override how long the server keeps the attribute after the last request carrying it. Only WebSocket event delivery can be affected by an attribute expiring, since every HTTP request refreshes them.
- `Values`: the values the simulated clients send, each with the probability (`Weight`) of being picked. Weights must sum to 1. For the select attributes (`network_interface_type`, `vpn_active`, `mdm_enrolled`, `jailbreak_detected` and `os_platform`), values must be one of their options.

### Policies

*[]struct{
  Name string
  Action string
  Role string
  UserAttribute string
  UserAttributeValues []string
  Operator string
  SessionAttribute string
  SessionAttributeValues []string
}*

The permission policies to create. A permission policy governs its action on every public and private channel. All the policies of a role and action must grant access, and the server requires `channel_read_access` to grant `channel_write_access`. At most 10 are supported, as the server only evaluates the first 10 permission policies. Policies created by earlier runs that are no longer configured get deleted.

Each policy combines one user attribute condition and one session attribute condition:

```
user.attributes.<UserAttribute> in [<UserAttributeValues>] && (or ||) user.session.<SessionAttribute> in [<SessionAttributeValues>]
```

- `Name`: the unique name of the policy.
- `Action`: `channel_read_access` or `channel_write_access`.
- `Role`: `system_user` (default), `system_admin` or `system_guest`. System admins without a dedicated policy fall back to the `system_user` ones.
- `UserAttribute`, `UserAttributeValues`: the user attribute to check, and its values that grant access.
- `Operator`: `and` (default) or `or`.
- `SessionAttribute`, `SessionAttributeValues`: the session attribute to check, and its values that grant access.

The expected share of users granted read and write access, given the weights of the values, is logged when an agent sets up the policies.
