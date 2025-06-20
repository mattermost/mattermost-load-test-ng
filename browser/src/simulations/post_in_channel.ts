// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Page} from '@playwright/test';

import {log} from '../app.js';

export async function postInChannel({page}: {page: Page}): Promise<void> {
  log.info('[simulation][run][postInChannel]');

  try {
    await page.waitForSelector('#post_textbox');
    await page.type('#post_textbox', `Hello, world! ${new Date().toISOString()}`, {delay: 100});
    await page.keyboard.press('Enter');

    log.info('[simulation][pass][postInChannel]');
  } catch (error) {
    throw {error, testId: 'postInChannel'};
  }
}
