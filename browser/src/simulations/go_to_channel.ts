// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Page} from 'playwright';

import {log} from '../app.js';

export async function goToChannel(page: Page, channelId: string): Promise<void> {
  log.info('run--goToChannel');

  try {
    const channel = page.locator(`#sidebarItem_${channelId}`);
    await channel.waitFor({state: 'visible'});
    await channel.click();

    log.info('pass--goToChannel');
  } catch (error) {
    throw {error, testId: 'goToChannel'};
  }
}
