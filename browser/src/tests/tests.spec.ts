// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {test} from '@playwright/test';

import {handlePreferenceCheckbox, performLogin} from './login.js';
import {postInChannel} from './post_in_channel.js';
import {scrollInChannel} from './scrolling_in_channel.js';

test('Scenario 1', async ({page}) => {
  await page.goto('https://community.mattermost.com/core/channels/zubairloadtest1');
  await page.waitForNavigation();
  await handlePreferenceCheckbox(page);
  await performLogin({page, userId: 'zubair.loadtest3@maildrop.cc', password: 'Password-1!'});
  await postInChannel({page});
  await scrollInChannel(page, 'sidebarItem_public-test-channel', 100, 400, 100);
});
