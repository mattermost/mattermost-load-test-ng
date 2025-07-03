// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {handlePreferenceCheckbox, performLogin} from './login.js';
import {postInChannel} from './post_in_channel.js';
import {scrollInChannel} from './scrolling_in_channel.js';
import type {BrowserInstance} from '../lib/browser_manager.js';

export async function scenario1({page, userId, password}: BrowserInstance, serverURL: string, simulationMode = true) {
  if (!page) {
    throw new Error('Page is not initialized');
  }

  try {
    await page.goto(serverURL);
    await handlePreferenceCheckbox(page);
    await performLogin({page, userId, password});

    // Runs the simulation atleast once and then runs it in continuous loop if simulationMode is true
    do {
      const scrollCount = simulationMode ? 40 : 3;
      await postInChannel({page});
      await scrollInChannel(page, 'sidebarItem_off-topic', scrollCount, 400, 500);
      await scrollInChannel(page, 'sidebarItem_town-square', scrollCount, 400, 500);
    } while (simulationMode);
  } catch (error: any) {
    throw {error: error?.error, testId: error?.testId};
  }
}
