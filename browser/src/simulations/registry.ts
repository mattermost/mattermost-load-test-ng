// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {BrowserInstance} from '../lib/browser_manager.js';
import {postAndScrollScenario} from './post_and_scroll_scenario.js';

export enum SimulationIds {
  postAndScroll = 'postAndScroll',
  login = 'login',
}

export type SimulationRegistryItem = {
  id: SimulationIds;
  name?: string;
  description?: string;
  scenario: (browserInstance: BrowserInstance, serverURL: string) => Promise<void>;
};

export const SimulationsRegistry: SimulationRegistryItem[] = [
  {
    id: SimulationIds.postAndScroll,
    name: 'Post and Scroll scenario',
    description: 'A basic scenario that posts and scrolls in a channel',
    scenario: postAndScrollScenario,
  },
];
