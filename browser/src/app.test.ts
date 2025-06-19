// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {describe, test, expect, vi, beforeEach} from 'vitest';

const mockFastifyRegister = vi.fn();
const mockFastifyInstance = {
  register: mockFastifyRegister,
  server: {},
  log: {
    error: vi.fn(),
    warn: vi.fn(),
    info: vi.fn(),
  },
};
const mockFastify = vi.fn(() => mockFastifyInstance);

vi.mock('fastify', () => ({
  default: mockFastify,
}));

vi.mock('./config/log.js', () => ({
  getServerLoggerConfig: vi.fn().mockReturnValue({}),
  createLogger: vi.fn().mockReturnValue({
    error: vi.fn(),
    warn: vi.fn(),
    info: vi.fn(),
  }),
  isConsoleLoggingEnabled: vi.fn().mockReturnValue(true),
}));

vi.mock('./routes/browser.js', () => ({
  default: 'browser-routes',
}));

vi.mock('./routes/health.js', () => ({
  default: 'health-routes',
}));

describe('src/app', () => {
  beforeEach(() => {
    vi.resetModules();

    mockFastify.mockClear();
    mockFastifyRegister.mockClear();
  });

  test('should create app instance', async () => {
    await import('./app.js');

    expect(mockFastify).toHaveBeenCalled();
  });

  test('should register health routes', async () => {
    await import('./app.js');
    const {default: healthRoutes} = await import('./routes/health.js');

    expect(mockFastifyRegister).toHaveBeenCalledWith(healthRoutes);
  });

  test('should register browser routes', async () => {
    await import('./app.js');
    const {default: browserRoutes} = await import('./routes/browser.js');

    expect(mockFastifyRegister).toHaveBeenCalledWith(browserRoutes);
  });
});
