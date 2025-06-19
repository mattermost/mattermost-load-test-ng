// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Page} from '@playwright/test';

import {log} from '../app.js';

export async function handlePreferenceCheckbox(page: Page) {
  log.info('[simulation][handlePreferenceCheckbox]-start');

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

    log.info('[simulation][handlePreferenceCheckbox]-ok');
  } catch (error) {
    // If checkbox not found, log and skip
    log.info('[simulation][handlePreferenceCheckbox]-skipped');
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
  log.info('[simulation][performLogin]-start');

  try {
    await page.waitForSelector('#input_loginId');
    await page.type('#input_loginId', userId);
    await page.type('#input_password-input', password);
    await page.keyboard.press('Enter');

    // Ensure the spinner has time to render before succeeding
    log.info('[simulation][performLogin]-OK');
  } catch (error) {
    throw error;
  }
}
