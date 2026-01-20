// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Page} from '@playwright/test';
import type {Logger} from '../types/log.js';
import {LandingLoginPage} from '@mattermost/playwright-lib';

export async function handleLandingPage(page: Page, log: Logger): Promise<void> {
  log.info('run--handleLandingPage');

  try {
    const isLandingPage = await page
      .waitForURL((url) => url.pathname.includes('/landing'))
      .then(() => true)
      .catch(() => false);
    if (!isLandingPage) {
      throw new Error('Not on landing page');
    }

    const landingLoginPage = new LandingLoginPage(page);
    await landingLoginPage.toBeVisible();

    await landingLoginPage.viewInBrowserButton.click();

    log.info('pass--handleLandingPage');
  } catch {
    // If checkbox not found, log and skip
    log.info('skip--handleLandingPage');
  }
}
