// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Page} from '@playwright/test';
import {log} from '../app.js';

export async function scrollInChannel(
  page: Page,
  channelId: string,
  scrollCount: number,
  scrollStep: number,
  pauseBetweenScrolls: number,
): Promise<void> {
  log.info(`[simulation][run][scrollInChannel]`);

  try {
    // Navigate to the specified channel
    await page.evaluate((id) => {
      const element = document.getElementById(id);
      if (element) {
        element.click();
      } else {
        throw new Error(`Channel with id ${id} not found`);
      }
    }, channelId);

    for (let i = 0; i < scrollCount; i++) {
      // Scroll up by scrollStep pixels - passing params as a single object
      await page.evaluate(
        (params) => {
          const container = document.querySelector(params.selector) as HTMLElement;
          if (container) {
            // Negative value scrolls UP
            container.scrollBy({top: -params.step, behavior: 'smooth'});
          }
        },
        {selector: '.post-list__dynamic', step: scrollStep},
      );

      await page.waitForTimeout(pauseBetweenScrolls);
    }

    log.info('[simulation][pass][scrollInChannel]');
  } catch (error) {
    throw {error, testId: 'scrollInChannel'};
  }
}
