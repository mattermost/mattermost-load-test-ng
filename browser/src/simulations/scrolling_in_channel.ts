// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Page} from 'playwright';

import type {Logger} from '@mattermost/loadtest-browser-lib';

export async function scrollInChannel(
  page: Page,
  scrollCount: number,
  scrollStep: number,
  pauseBetweenScrolls: number,
  log: Logger,
): Promise<void> {
  log.info('run--scrollInChannel');

  try {
    await page.locator('.post-list__dynamic').waitFor({state: 'visible'});

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

    log.info('pass--scrollInChannel');
  } catch (error) {
    throw {error, testId: 'scrollInChannel'};
  }
}
