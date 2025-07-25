// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {test} from '@playwright/test';

import {postAndScrollScenario} from '../simulations/post_and_scroll_scenario.js';
import {getMattermostServerURL} from '../utils/config.js';
import type {BrowserInstance} from '../lib/browser_manager.js';

test('Post and Scroll Scenario', async ({page}) => {
  const browserInstance = {
    page,
    userId: 'user1@example.com',
    password: 'Password-1!',
  } as BrowserInstance;

  const serverURL = getMattermostServerURL();

  await postAndScrollScenario(browserInstance, serverURL, false);
});
