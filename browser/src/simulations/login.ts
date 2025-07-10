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

    log.info('pass--performLogin');
  } catch (error) {
    throw {error, testId: 'performLogin'};
  }
}
