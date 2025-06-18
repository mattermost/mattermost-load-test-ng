// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Page} from 'playwright';
import {log} from '../app.js';

export async function postInChannel(page: Page) {
  log.info('[test-log][postInChannel]-start');

  try {
    await page.waitForSelector('#post_textbox');
    await page.type('#post_textbox', 'Hello, world!', {delay: 100});
    await page.keyboard.press('Enter');

    log.info('[test-log][postInChannel]-ok');
  } catch (error) {
    throw error;
  }
}
