// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {describe, test, expect, vi, beforeEach} from 'vitest';

const mockAppListen = vi.fn().mockResolvedValue(undefined);
const mockAppClose = vi.fn().mockImplementation(() => {
  return Promise.resolve();
});

const mockLogError = vi.fn();
const mockLogInfo = vi.fn();

const mockApp = {
  listen: mockAppListen,
  close: mockAppClose,
  address: vi.fn(),
  log: {
    error: vi.fn(),
    warn: vi.fn(),
    info: vi.fn(),
  },
};

vi.mock('./app.js', () => {
  return {
    app: mockApp,
    createApp: vi.fn().mockReturnValue(mockApp),
    log: {
      error: mockLogError,
      info: mockLogInfo,
    },
  };
});

describe('src/server', () => {
  const mockExit = vi.spyOn(process, 'exit').mockImplementation((() => undefined) as any);
  const originalEnv = {...process.env};
  const mockConsoleLog = vi.spyOn(console, 'log').mockImplementation(() => undefined);
  const mockConsoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);

  beforeEach(() => {
    vi.resetModules();
    process.env = {...originalEnv};
    mockConsoleLog.mockClear();
    mockConsoleError.mockClear();
    mockExit.mockClear();
    mockAppListen.mockClear();
    mockAppClose.mockClear();
    mockLogError.mockClear();
    mockLogInfo.mockClear();
  });

  test('should handle server start error', async () => {
    mockAppListen.mockRejectedValueOnce(new Error('Failed to start'));

    await import('./server.js');

    expect(mockLogError).toHaveBeenCalledWith(expect.stringContaining('LTBrowser server failed to start:'));
  });

  test('should handle server shutdown on SIGTERM', async () => {
    process.env.BROWSER_AGENT_API_URL = 'http://localhost:8080';
    await import('./server.js');

    process.emit('SIGTERM', 'SIGTERM');

    expect(mockLogInfo).toHaveBeenCalledWith(expect.stringContaining('Received SIGTERM, LTBrowser server stopping'));
    expect(mockAppClose).toHaveBeenCalled();
  });

  test('should handle server shutdown on SIGINT', async () => {
    process.env.BROWSER_AGENT_API_URL = 'http://localhost:8080';
    await import('./server.js');

    process.emit('SIGINT', 'SIGINT');

    expect(mockLogInfo).toHaveBeenCalledWith(expect.stringContaining('Received SIGINT, LTBrowser server stopping'));
    expect(mockAppClose).toHaveBeenCalled();
  });

  test('should handle server shutdown error', async () => {
    process.env.BROWSER_AGENT_API_URL = 'http://localhost:8080';
    mockAppClose.mockRejectedValueOnce(new Error('Failed to close'));

    await import('./server.js');
    process.emit('SIGTERM', 'SIGTERM');

    await vi.waitFor(() =>
      expect(mockLogError).toHaveBeenCalledWith(
        expect.stringContaining('LTBrowser server encountered error during shutdown:'),
      ),
    );
  });

  test('should handle uncaught exceptions while server is running', async () => {
    process.env.BROWSER_AGENT_API_URL = 'http://localhost:8080';
    await import('./server.js');

    process.emit('uncaughtException', new Error('Uncaught error'));

    await vi.waitFor(() => {
      expect(mockLogError).toHaveBeenCalledWith(expect.stringContaining('Uncaught exception:'));
    });
  });

  test('should handle unhandled rejections while server is running', async () => {
    process.env.BROWSER_AGENT_API_URL = 'http://localhost:8080';
    await import('./server.js');

    const mockPromise = Promise.reject(new Error('Unhandled rejection'));
    const mockReason = new Error('Unhandled rejection');

    process.emit('unhandledRejection', mockReason, mockPromise);

    await vi.waitFor(() => {
      expect(mockLogError).toHaveBeenCalledWith(expect.stringContaining('Unhandled rejection'));
    });
  });
});
