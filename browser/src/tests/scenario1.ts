// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {handlePreferenceCheckbox, performLogin} from './login.js';
import {postInChannel} from './post_in_channel.js';
import {scrollInChannel} from './scrolling_in_channel.js';
import type {BrowserInstance} from '../lib/browser_manager.js';

export async function scenario1({page, userId, password}: BrowserInstance, serverURL: string) {
  if (!page) {
    throw new Error('Page is not initialized');
  }

  try {
    await page.goto(serverURL);
    await page.waitForNavigation();
    await handlePreferenceCheckbox(page);
    await performLogin({page, userId, password});

    while (true) {
      await postInChannel({page});
      await scrollInChannel(page, 'sidebarItem_off-topic', 40, 400, 500);
      await scrollInChannel(page, 'sidebarItem_town-square', 40, 400, 500);
    }
  } catch (error) {
    throw error;
  }
}
