# Packs directory of the browser loadtest

This directory stores local `.tgz` package files for plugins and other dependencies that are not available on npm. These are installed as `file:` dependencies by `browser/package.json`.

## Why this directory exists

Some packages used by the browser load test runner are not published to a public npm registry. This is by design: these packages are developed in tandem with this project, and maintaining versioned npm releases for every incremental change would introduce unnecessary overhead. Instead, they are built from their respective source repositories, packaged as tarballs via `npm pack`, and placed in this directory so that the browser runner can consume them locally without any dependency on an external registry.

A common use case is plugin load testing. When developing or iterating on plugin-specific simulation scripts, the plugin's load test package should be built and placed here. This allows any changes to the simulation scripts to be picked up immediately — simply rebuild and replace the tarball — without requiring a new npm release. For example:

```json
"mattermost-plugin-playbooks-loadtest-browser": "file:packs/mattermost-plugin-playbooks-loadtest-browser-2.4.3.tgz",
"some-internal-tool": "file:packs/some-internal-tool-1.0.0.tgz"
```

## Adding a new package

1. Build the package from its source repository and produce a `.tgz` via `npm pack`.
2. Place the `.tgz` file in this directory.
3. Add it as a dependency in `browser/package.json` using a `file:packs/<filename>.tgz` reference.
4. Import and register its `SimulationsRegistry` in `browser/src/registry.ts`.
