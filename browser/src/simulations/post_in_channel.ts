// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Page} from 'playwright';

import {log} from '../app.js';

export async function postInChannel({page}: {page: Page}): Promise<void> {
  log.info('run--postInChannel');

  try {
    const textbox = page.getByTestId('post_textbox');
    await textbox.waitFor({state: 'visible'});
    await textbox.click();
    const nowDate = new Date();
    await textbox.fill(`Hello, world! ${nowDate.toLocaleDateString()} ${nowDate.toLocaleTimeString()}`);

    const postButton = page.getByRole('button', {name: 'Send Now'});
    await postButton.waitFor({state: 'visible'});
    await postButton.click();

    // Wait until textbox is cleared
    await textbox.evaluate((el) => (el as HTMLTextAreaElement).value === '');

    log.info('pass--postInChannel');
  } catch (error) {
    throw {error, testId: 'postInChannel'};
  }
}
