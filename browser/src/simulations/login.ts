// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Page} from 'playwright';

import {log} from '../app.js';

export async function handlePreferenceCheckbox(page: Page) {
  log.info('run--handlePreferenceCheckbox');

  try {
    // Try to find the checkbox with a short timeout
    await page.waitForSelector('label.get-app__preference input.get-app__checkbox', {timeout: 2000});

    // If found, click it
    await page.click('label.get-app__preference input.get-app__checkbox');

    await page.waitForSelector('a.btn.btn-tertiary.btn-lg');
    await page.evaluate(() => {
      const buttons = Array.from(document.querySelectorAll('a.btn.btn-tertiary.btn-lg'));
      const viewButton = buttons.find((button) => button.textContent?.trim() === 'View in Browser');
      if (viewButton) {
        (viewButton as HTMLElement).click();
      }
    });

    log.info('pass--handlePreferenceCheckbox');
  } catch (error) {
    // If checkbox not found, log and skip
    log.info('skip--handlePreferenceCheckbox');
  }
}

export async function handleTeamSelection(page: Page): Promise<void> {
  log.info('run--handleTeamSelection');

  try {
    // Wait briefly to see if we land on team selection page
    await page.waitForSelector('.signup-team-all', {timeout: 3000});

    // If we found the team selection page, click the first team link
    const firstTeamLink = await page.locator('.signup-team-all a').first();
    if (await firstTeamLink.count() > 0) {
      await firstTeamLink.click();
      log.info('pass--handleTeamSelection--clicked-first-team');
    } else {
      log.info('skip--handleTeamSelection--no-team-links-found');
    }
  } catch (error) {
    // If team selection page not found, continue normally
    log.info('skip--handleTeamSelection--no-team-selection-page');
  }
}

export async function performLogin({
  page,
  userId,
  password,
}: {
  page: Page;
  userId: string;
  password: string;
}): Promise<void> {
  log.info('run--performLogin');

  try {
    await page.waitForSelector('#input_loginId');
    await page.type('#input_loginId', userId);
    await page.type('#input_password-input', password);
    await page.keyboard.press('Enter');

    // After login, check if we need to select a team
    await handleTeamSelection(page);

    log.info('pass--performLogin');
  } catch (error) {
    throw {error, testId: 'performLogin'};
  }
}
