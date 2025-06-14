// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {handlePreferenceCheckbox, performLogin} from './login.js';
import {postInChannel} from './post_in_channel.js';
import {scrollInChannel} from './scrolling_in_channel.js';
import type {BrowserInstance} from '../lib/browser_manager.js';

export async function scenario1({page, userId, password}: BrowserInstance) {
  if (!page) {
    throw new Error('Page is not initialized');
  }

  try {
    await page.goto('https://community.mattermost.com');
    await page.waitForNavigation();
    await handlePreferenceCheckbox(page);
    await performLogin({page, userId, password});
    await postInChannel({page});
    await scrollInChannel(page, 'sidebarItem_public-test-channel', 40, 400, 500);
  } catch (error) {
    throw error;
  }
}
