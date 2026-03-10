# Packs directory of the browser loadtest

This directory stores local `.tgz` package files that are installed as `file:` dependencies by `browser/package.json`. This directory is tracked by git. The `.tgz` files should be committed so that the browser runner can be built without requiring access to external registries.

## Why this directory exists

Some packages used by the browser load test runner cannot be published to a public npm registry — either because they are internal tools or because they contain plugin-specific load test simulations. Instead, they are built locally and placed here as tarballs, then referenced directly in `package.json` using `file:` paths:

```json
"@mattermost/loadtest-browser-lib": "file:packs/loadtest-browser-lib.tgz",
"mattermost-plugin-playbooks-loadtest-browser": "file:packs/mattermost-plugin-playbooks-loadtest-browser-2.4.3.tgz"
```

## Adding a new package

To add a new local package:

1. Build the package from its source repository and produce a `.tgz` via `npm pack`.
2. Place the `.tgz` file in this directory.
3. Add it as a dependency in `browser/package.json` using a `file:packs/<filename>.tgz` reference.
4. Import and register its `SimulationsRegistry` in `browser/src/registry.ts`.
