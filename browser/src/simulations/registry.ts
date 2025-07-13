// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {postAndScrollScenario} from './post_and_scroll_scenario.js';

export enum SimulationIds {
  postAndScroll = 'postAndScroll',
  login = 'login',
}

type SimulationRegistryItem = {
  id: SimulationIds;
  name?: string;
  description?: string;
  scenario: unknown;
};

export const SimulationsRegistry: SimulationRegistryItem[] = [
  {
    id: SimulationIds.postAndScroll,
    name: 'Post and Scroll scenario',
    description: 'A basic scenario that posts and scrolls in a channel',
    scenario: postAndScrollScenario,
  },
];
