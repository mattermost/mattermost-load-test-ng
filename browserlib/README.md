# @mattermost/loadtest-browser-lib

This library provides shared types, utilities, and helper functions for browser-based load testing of Mattermost using Playwright.

## Installation

The library is published to npm as `@mattermost/loadtest-browser-lib`. See `package.json` for the latest version.

## Development

When actively developing changes to this library and testing them in the `browser/` app:

1. Build and pack the library locally:
   ```bash
   make build-dev
   ```
   This compiles the code, generates type declarations, and produces a `.tgz` file in `browser/packs/`.

2. Temporarily update `browser/package.json` to reference the local pack:
   ```json
   "@mattermost/loadtest-browser-lib": "file:packs/loadtest-browser-lib.tgz"
   ```

3. Run `npm install` in `browser/` to pick up the local changes.

Note: Once the changes are verified, publish the library to npm and revert `browser/package.json` back to the npm version reference.

## Publishing

1. Bump the version in `package.json`.
2. Build the library:
   ```bash
   make build
   ```
3. To publish to the npm registry, contact the Mattermost staff.
