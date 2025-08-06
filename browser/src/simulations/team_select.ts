// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Page} from 'playwright';
import {log} from '../app.js';

export async function handleTeamSelection(page: Page): Promise<void> {
  log.info('run--handleTeamSelection');

  try {
    const isTeamSelectionPage = await page
      .waitForURL((url) => url.pathname.includes('/select_team'))
      .then(() => true)
      .catch(() => false);
    if (!isTeamSelectionPage) {
      throw new Error('Not on team selection page');
    }

    await page.waitForSelector('.signup-team-dir a');
    const teamElement = page.locator('.signup-team-dir a').first();
    await teamElement.click();

    await page.waitForURL((url) => !url.pathname.includes('/select_team'));

    log.info('pass--handleTeamSelection');
  } catch (error) {
    log.info('skip--handleTeamSelection');
  }
}
