# BrowserController Configuration

## RunInHeadless

*boolean*

If set to `true`, browser simulations will run in a headless Chromium browser instance. If set to `false`, browser windows will be visible, which is helpful for debugging simulations.

## SimulationTimeoutMs

*int*

The timeout in milliseconds for browser simulations. This value sets the default timeout for page interactions and navigation operations. The value must be greater than or equal to `0`. See [Playwright's setDefaultTimeout](https://playwright.dev/docs/api/class-page#page-set-default-timeout) for more details.

**Default:** `60000`
