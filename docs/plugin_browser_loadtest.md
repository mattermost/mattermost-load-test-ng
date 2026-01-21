# Browser load testing with Plugins

This document describes how to integrate a Mattermost plugin's browser load testing package into the Mattermost Load Test NG framework.

## Overview

The browser load testing framework supports plugin-specific simulations through a modular registry system. Plugin authors can create their own browser load testing packages that:

1. Export a `SimulationsRegistry` array containing plugin-specific scenarios.
1. Are added as dependencies to the `mattermost-load-test-ng/browser` package.

## Integration into mattermost-load-test-ng/browser

### Prerequisites

1. The package should be built to ESM modules
1. Each simulation must conform to the `SimulationRegistryItem` interface from `@mattermost/loadtest-browser-lib`.
1. Expose the `SimulationsRegistry` array as a named export.
    ```typescript
    // mattermost-plugin-<name>/loadtest/browser/src/registry.ts
    import {type SimulationRegistryItem} from '@mattermost/loadtest-browser-lib';

    import {myScenario} from './simulations/my_scenario.js';

    export const SimulationsRegistry: SimulationRegistryItem[] = [
      {
        id: 'pluginNameMyScenario',  // Prefix with plugin name to avoid collisions
        name: 'Plugin Name - My Scenario',
        description: 'Description of what this scenario does',
        scenario: myScenario,
      },
    ];
    ```

### Steps

1. Build the Plugin browser load testing package
    ```bash
    cd <plugin-repo>/loadtest/browser
    npm run build
    ```

1. Add the plugin's loadtest-browser package as a dependency to mattermost-load-test-ng/browser

    ```bash
    cd mattermost-load-test-ng/browser
    npm install --save mattermost-plugin-<name>-loadtest-browser
    ```

1. Update `mattermost-load-test-ng/browser/src/registry.ts` to import and spread the plugin's registry:

    ```typescript
    // 1. Import plugin simulation registries
    import {SimulationsRegistry as PluginsSimulationsRegistry} from 'mattermost-plugin-<name>-loadtest-browser';

    export const SimulationsRegistry: SimulationRegistryItem[] = [
      // ...Mattermost's simulations registry

      // 2. Spread the plugin's simulations registry
      ...PluginsSimulationsRegistry,
    ];
    ```

## Best Practices

1. Prefix simulation IDs with the plugin name to avoid collisions. For example, `playbooksCreateAndRun`, `callsCreateDmCall`, etc.
1. Use the common helpers provided by `@mattermost/loadtest-browser-lib` for browser load testing.

