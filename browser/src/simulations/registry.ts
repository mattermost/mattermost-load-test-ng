// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {type SimulationRegistryItem} from '@mattermost/loadtest-browser';

import {postAndScrollScenario} from './post_and_scroll_scenario.js';

import {SimulationsRegistry as PlaybooksSimulationsRegistry} from 'mattermost-plugin-playbooks-loadtest-browser';

export const SimulationsRegistry: SimulationRegistryItem[] = [
  {
    id: 'postAndScroll',
    name: 'Post and Scroll scenario',
    description: 'A basic scenario that posts and scrolls in a channel',
    scenario: postAndScrollScenario,
  },
  ...PlaybooksSimulationsRegistry,
];
