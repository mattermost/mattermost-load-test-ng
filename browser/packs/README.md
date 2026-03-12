# Packs directory of the browser loadtest

This directory stores local `.tgz` package files used as `file:` dependencies by `browser/package.json`. It serves two purposes: providing packages that are not published to any public npm registry, and overriding npm-published packages with local builds during active development.

## Why this directory exists

This directory covers two distinct use cases:

**Packages not on npm** — Some packages, such as plugin-specific load test simulation scripts, are never published to a public registry. They are developed in tandem with this project, and maintaining versioned npm releases for every incremental change would introduce unnecessary overhead. Instead, they are built from their respective source repositories, packaged as tarballs via `npm pack`, and placed here so the browser runner can consume them without any dependency on an external registry.

**Local development overrides** — Packages that are published on npm, such as `@mattermost/loadtest-browser-lib`, can also be placed here to override the registry version during local development. This allows changes to be tested end-to-end immediately — simply rebuild the package and replace the tarball — without requiring a new npm release.

A common example of both cases is plugin load testing: when developing or iterating on plugin-specific simulation scripts, the plugin's load test package is built locally and placed here so that changes are picked up immediately. For example:

```json
"mattermost-plugin-playbooks-loadtest-browser": "file:packs/mattermost-plugin-playbooks-loadtest-browser-2.4.3.tgz",
"some-internal-tool": "file:packs/some-internal-tool-1.0.0.tgz"
```

## Adding a new package

1. Build the package from its source repository and produce a `.tgz` via `npm pack`.
2. Place the `.tgz` file in this directory.
3. Add it as a dependency in `browser/package.json` using a `file:packs/<filename>.tgz` reference.
4. Import and register its `SimulationsRegistry` in `browser/src/registry.ts`.
