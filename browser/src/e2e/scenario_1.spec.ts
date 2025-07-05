// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {test} from '@playwright/test';

import {scenario1} from '../simulations/scenario_1.js';
import {getMattermostServerURL} from '../utils/config.js';
import type {BrowserInstance} from '../lib/browser_manager.js';

test('Scenario 1', async ({page}) => {
  const browserInstance = {
    page,
    userId: 'user1@example.com',
    password: 'Password-1!',
  } as BrowserInstance;

  const serverURL = getMattermostServerURL();

  await scenario1(browserInstance, serverURL, false);
});
