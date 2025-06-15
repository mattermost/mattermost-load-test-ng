// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {describe, expect, test} from 'vitest';
import {BrowserTestSessionManager} from './browser_manager.js';

describe('BrowserManager', () => {
  test('should create a instance of BrowserTestSessionManager', () => {
    const browserManager = BrowserTestSessionManager.getInstance();
    expect(browserManager).toBeDefined();
  });
});
