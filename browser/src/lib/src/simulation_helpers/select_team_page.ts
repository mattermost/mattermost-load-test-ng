// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Page} from 'playwright';

import type {Logger} from '@mattermost/loadtest-browser-lib';

type ExtraArgs = {
  /**
   * If left empty, the first team in the list will be selected.
   */
  teamName?: string;
};

export async function performTeamSelection(page: Page, log: Logger, {teamName}: ExtraArgs): Promise<void> {
  log.info('run--performTeamSelection');

  try {
    const isTeamSelectionPage = await page
      .waitForURL((url) => url.pathname.includes('/select_team'))
      .then(() => true)
      .catch(() => false);
    if (!isTeamSelectionPage) {
      throw new Error('Not on team selection page');
    }

    await page.waitForSelector('.signup-team-dir a');

    let teamIconElement;
    if (teamName) {
      teamIconElement = page.locator('.signup-team-dir a').filter({
        has: page.locator('.signup-team-dir__name', {hasText: teamName}),
      });
    } else {
      teamIconElement = page.locator('.signup-team-dir a').first();
    }

    await teamIconElement.click();

    await page.waitForURL((url) => !url.pathname.includes('/select_team'));

    log.info('pass--performTeamSelection');
  } catch {
    log.info('skip--performTeamSelection');
  }
}
