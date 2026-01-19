# BrowserController Configuration

## RunInHeadless

*boolean*

If set to `true`, browser simulations will run in a headless Chromium browser instance. If set to `false`, browser windows will be visible, which is helpful for debugging simulations.

## SimulationTimeoutMs

*int*

The timeout in milliseconds for browser simulations. This value sets the default timeout for page interactions and navigation operations. The value must be greater than or equal to `0`. See [Playwright's setDefaultTimeout](https://playwright.dev/docs/api/class-page#page-set-default-timeout) for more details.

**Default:** `60000`

## EnabledPlugins

*boolean*

The default value is `false`, so the load test will use only the predefined simulations built into the browser controller. If set to `true`, the load test will also look for and run simulations provided by plugins.

**Default:** `false`

## SimulationId

*string*

The ID of the simulation to run. For a complete list and description of available simulations, see the [Browser Simulations Registry](../browser_simulations_registry.md). If `EnabledPlugins` is set to `true`, specify the simulation ID as defined in the plugin’s Simulations Registry.
