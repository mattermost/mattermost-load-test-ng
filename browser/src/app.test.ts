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

vi.mock('./utils/log.js', () => ({
  getServerLoggerConfig: vi.fn().mockReturnValue({}),
  createLogger: vi.fn().mockReturnValue({
    error: vi.fn(),
    warn: vi.fn(),
    info: vi.fn(),
  }),
}));

vi.mock('./utils/config.js', () => ({
  isConsoleLoggingEnabled: vi.fn().mockReturnValue(true),
}));

// Mock the route modules to return functions
vi.mock('./routes/browser.js', () => ({
  default: vi.fn(),
}));

vi.mock('./routes/health.js', () => ({
  default: vi.fn(),
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

    expect(mockFastifyRegister).toHaveBeenCalledWith(expect.any(Function));
  });

  test('should register browser routes', async () => {
    await import('./app.js');

    expect(mockFastifyRegister).toHaveBeenCalledWith(expect.any(Function));
  });
});
