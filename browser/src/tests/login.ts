// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import chalk from 'chalk';
import type {Page} from '@playwright/test';
import ora from 'ora';

export async function handlePreferenceCheckbox(page: Page) {
  const spinner = ora({
    text: chalk.blue('[test-log][handlePreferenceCheckbox]-Running'),
    color: 'blue',
  }).start();

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

    spinner.succeed(chalk.green('[test-log][handlePreferenceCheckbox]-OK'));
  } catch (error) {
    // If checkbox not found, log and skip
    spinner.info(chalk.yellow('[test-log][handlePreferenceCheckbox]-Skipped'));
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
  const spinner = ora({
    text: chalk.blue('[test-log][performLogin]-Running'),
    color: 'blue',
  }).start();

  try {
    await page.waitForSelector('#input_loginId');
    await page.type('#input_loginId', userId);
    await page.type('#input_password-input', password);
    await page.keyboard.press('Enter');

    // Ensure the spinner has time to render before succeeding
    spinner.succeed(chalk.green('[test-log][performLogin]-OK'));
  } catch (error) {
    spinner.fail(chalk.red('[test-log][performLogin]-Failed'));
    console.error(error);
  }
}
