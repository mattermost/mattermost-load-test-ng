// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {describe, expect, test, vi} from 'vitest';

describe('BrowserRoutes', () => {
  test('should register browser routes', async () => {
    const fastify = {
      get: vi.fn(),
      post: vi.fn(),
      delete: vi.fn(),
      addHook: vi.fn(),
    };

    const {default: browserRoutes} = await import('./browser.js');

    await browserRoutes(fastify as any);

    expect(fastify.get).toHaveBeenCalledWith('/browsers', expect.any(Function));
    expect(fastify.post).toHaveBeenCalledWith('/browsers', expect.any(Function));
    expect(fastify.delete).toHaveBeenCalledWith('/browsers', expect.any(Function));
    expect(fastify.addHook).toHaveBeenCalledWith('onClose', expect.any(Function));
  });
});
