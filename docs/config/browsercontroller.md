# BrowserController Configuration

This document describes the configuration options for the BrowserController [browsercontroller.json](../../config/browsercontroller.sample.json), which manages browser-based load test simulations using Playwright. 

## SimulationId

*string* (required)

The ID of the simulation to run. For a complete list of available simulations and their descriptions, see the [Browser Simulations Registry](../browser_simulations_registry.md).

Note: When `EnabledPlugins` is set to `true`, use the simulation ID as defined in the plugin's Simulations Registry instead.

## RunInHeadless

*boolean*

If set to `true`, browser simulations will run in a headless Chromium browser instance. If set to `false`, browser windows will be visible, which is helpful for debugging simulations.

**Default:** `true`

## SimulationTimeoutMs

*int*

The timeout in milliseconds for browser simulations. This value sets the default timeout for page interactions and navigation operations. The value must be greater than or equal to `0`. 

See [Playwright's setDefaultTimeout](https://playwright.dev/docs/api/class-page#page-set-default-timeout) for more details.

**Default:** `60000`

## EnabledPlugins

*boolean*

When set to `false`, the load test uses only the predefined simulations built into the browser controller. When set to `true`, the load test also discovers and runs simulations provided by plugins being load tested.

See [Plugin Browser Load Testing](../plugin_browser_loadtest.md) for more details.

**Default:** `false`

## LogSettings

### EnableConsole

*bool*

When true, the browser server outputs log messages to the console based on ConsoleLevel option.

### ConsoleLevel

*string*

Level of detail at which log events are written to the console.

Possible values (in order of decreasing verbosity, these are case-sensitive):
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

Possible values (in order of decreasing verbosity, these are case-sensitive):
- `trace`
- `debug`
- `info`
- `warn`
- `error`
- `fatal`

When both `EnableConsole` and `EnableFile` are true, the logs are written asynchronously to reduce overhead.

### FileLocation

*string*

The location of the log file. Must be a valid file path including the file name.
