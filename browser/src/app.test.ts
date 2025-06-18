// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {describe, test, expect, vi, beforeEach, afterEach} from 'vitest';

vi.mock('fastify', () => {
  const mockRegister = vi.fn();
  const mockFastify = vi.fn(() => ({
    register: mockRegister,
    server: {},
  }));
  mockFastify.mockReturnValue = mockRegister;

  return {default: mockFastify};
});

vi.mock('./routes/browser.js', () => ({
  default: 'browser-routes',
}));

vi.mock('./routes/health.js', () => ({
  default: 'health-routes',
}));

describe('src/app', () => {
  const originalEnv = {...process.env};

  beforeEach(() => {
    vi.resetModules();
    process.env = {...originalEnv};
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  test('should create app instance', async () => {
    const {default: fastify} = await import('fastify');
    await import('./app.js');

    expect(fastify).toHaveBeenCalled();
  });

  test('should register health routes', async () => {
    const {app} = await import('./app.js');
    const {default: healthRoutes} = await import('./routes/health.js');

    expect(app.register).toHaveBeenCalledWith(healthRoutes);
  });

  test('should register browser routes', async () => {
    const {app} = await import('./app.js');
    const {default: browserRoutes} = await import('./routes/browser.js');

    expect(app.register).toHaveBeenCalledWith(browserRoutes);
  });
});
