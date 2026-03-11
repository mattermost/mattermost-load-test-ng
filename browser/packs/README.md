# Packs directory of the browser loadtest

This directory stores local `.tgz` package files for plugins and other dependencies that are not available on npm. These are installed as `file:` dependencies by `browser/package.json`.

## Why this directory exists

Some packages used by the browser load test runner are not published to a public npm registry — for example, plugin-specific load test simulations or internal tools. These are built from their respective source repositories, packed as tarballs, and placed here so the browser runner can install them without requiring access to external registries. For example:

```json
"mattermost-plugin-playbooks-loadtest-browser": "file:packs/mattermost-plugin-playbooks-loadtest-browser-2.4.3.tgz",
"some-internal-tool": "file:packs/some-internal-tool-1.0.0.tgz"
```

## Adding a new package

1. Build the package from its source repository and produce a `.tgz` via `npm pack`.
2. Place the `.tgz` file in this directory.
3. Add it as a dependency in `browser/package.json` using a `file:packs/<filename>.tgz` reference.
4. Import and register its `SimulationsRegistry` in `browser/src/registry.ts`.
