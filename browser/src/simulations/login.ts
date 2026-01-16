// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Page} from 'playwright';

import {log} from '../app.js';

export async function handlePreferenceCheckbox(page: Page) {
  log.info('run--handlePreferenceCheckbox');

  try {
    const isLandingPage = await page
      .waitForURL((url) => url.pathname.includes('/landing'))
      .then(() => true)
      .catch(() => false);
    if (!isLandingPage) {
      throw new Error('Not on landing page');
    }

    await page.waitForSelector('label.get-app__preference input.get-app__checkbox');
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
  } catch (_error) {
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
    const inputLoginId = page.getByTestId('login-id-input');
    await inputLoginId.waitFor({state: 'visible'});
    await inputLoginId.fill(userId);

    const inputPassword = page.locator('#input_password-input');
    await inputPassword.waitFor({state: 'visible'});
    await inputPassword.fill(password);

    const saveButton = page.getByTestId('saveSetting');
    await saveButton.waitFor({state: 'visible'});
    await saveButton.click();

    await page.waitForURL((url) => !url.pathname.includes('/login'));

    log.info('pass--performLogin');
  } catch (error) {
    throw {error, testId: 'performLogin'};
  }
}
