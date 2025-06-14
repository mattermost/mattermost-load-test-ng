// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import chalk from 'chalk';
import type {Page} from '@playwright/test';

export async function postInChannel({page}: {page: Page}): Promise<void> {
  console.log(chalk.blue('[test-log][postInChannel]-start'));

  try {
    await page.waitForSelector('#post_textbox');
    await page.type('#post_textbox', 'Hello, world!', {delay: 100});
    await page.keyboard.press('Enter');

    console.log(chalk.green('[test-log][postInChannel]-ok'));
  } catch (error) {
    console.log(chalk.red('[test-log][postInChannel]-fail'), error);
  }
}
