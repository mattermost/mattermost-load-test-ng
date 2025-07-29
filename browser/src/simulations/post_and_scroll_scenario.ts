// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {handlePreferenceCheckbox, performLogin} from './login.js';
import {postInChannel} from './post_in_channel.js';
import {scrollInChannel} from './scrolling_in_channel.js';
import type {BrowserInstance} from '../lib/browser_manager.js';

export async function postAndScrollScenario(
  {page, userId, password}: BrowserInstance,
  serverURL: string,
  runInLoop = true,
) {
  if (!page) {
    throw new Error('Page is not initialized');
  }

  await page.goto(serverURL);
  await handlePreferenceCheckbox(page);
  await performLogin({page, userId, password});

  // Runs the simulation at least once and then runs it in a continuous loop if runInLoop is true
  // which is true by default
  do {
    const scrollCount = runInLoop ? 40 : 3;
    await postInChannel({page});
    await scrollInChannel(page, 'sidebarItem_off-topic', scrollCount, 400, 500);
    await scrollInChannel(page, 'sidebarItem_town-square', scrollCount, 400, 500);
  } while (runInLoop);
}
