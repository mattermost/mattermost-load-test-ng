// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {type BrowserInstance} from '@mattermost/loadtest-browser-lib';

import {goToChannel} from './go_to_channel.js';
import {handlePreferenceCheckbox, performLogin} from './login.js';
import {postInChannel} from './post_in_channel.js';
import {scrollInChannel} from './scrolling_in_channel.js';
import {handleTeamSelection} from './team_select.js';

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

  await handleTeamSelection(page);

  // Runs the simulation at least once and then runs it in a continuous loop if runInLoop is true
  // which is true by default
  const scrollCount = runInLoop ? 40 : 3;
  do {
    await goToChannel(page, 'town-square');
    await postInChannel({page});
    await scrollInChannel(page, scrollCount, 400, 500);

    await goToChannel(page, 'off-topic');
    await postInChannel({page});
    await scrollInChannel(page, scrollCount, 400, 500);
  } while (runInLoop);
}
