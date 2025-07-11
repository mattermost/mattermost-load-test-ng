// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {test} from '@playwright/test';

import {noopScenario} from '../simulations/noop_scenario.js';
import {getMattermostServerURL} from '../utils/config.js';
import type {BrowserInstance} from '../lib/browser_manager.js';

test('Noop Scenario', async ({page}) => {
  const browserInstance = {
    page,
    userId: 'user1@example.com',
    password: 'Password-1!',
  } as BrowserInstance;

  const serverURL = getMattermostServerURL();

  await noopScenario(browserInstance, serverURL, false);
});
