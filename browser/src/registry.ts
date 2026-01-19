// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {type SimulationRegistryItem} from '@mattermost/loadtest-browser-lib';

import {postAndScrollScenario} from './simulations/post_and_scroll_scenario.js';

export const SimulationsRegistry: SimulationRegistryItem[] = [
  {
    id: 'postAndScroll',
    name: 'Post and Scroll scenario',
    description: 'A basic scenario that posts and scrolls in a channel',
    scenario: postAndScrollScenario,
  },
  // Here goes the plugins simulations registry
  // after it is imported from the plugin's loadtest-browser package
  // ...PluginsSimulationsRegistry,
];
