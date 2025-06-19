// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {describe, test, expect, vi, beforeEach} from 'vitest';

const mockAppListen = vi.fn().mockResolvedValue(undefined);
const mockAppClose = vi.fn().mockImplementation(() => {
  return Promise.resolve();
});
const mockAppServer = {
  address: vi.fn().mockImplementation(() => {
    return {port: Number(process.env.PORT) || 8080};
  }),
};
const mockLogError = vi.fn();
const mockLogInfo = vi.fn();
const mockLogWarn = vi.fn();

const mockApp = {
  listen: mockAppListen,
  close: mockAppClose,
  server: mockAppServer,
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
      warn: mockLogWarn,
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
    mockLogWarn.mockClear();
  });

  test('should start server with default port and host', async () => {
    delete process.env.PORT;
    delete process.env.HOST;

    await import('./server.js');

    expect(mockAppListen).toHaveBeenCalledWith({port: 8080, host: '127.0.0.1'});
  });

  test('should start server with environment variables port and host', async () => {
    process.env.PORT = '3000';
    process.env.HOST = '18.212.128.100';

    await import('./server.js');

    expect(mockAppListen).toHaveBeenCalledWith({port: 3000, host: '18.212.128.100'});
  });

  test('should handle server start error', async () => {
    mockAppListen.mockRejectedValueOnce(new Error('Failed to start'));

    await import('./server.js');

    expect(mockLogError).toHaveBeenCalledWith(
      expect.stringContaining('[server] Server failed to start'),
      expect.any(Error),
    );
  });

  test('should handle server shutdown on SIGTERM', async () => {
    await import('./server.js');

    process.emit('SIGTERM', 'SIGTERM');

    expect(mockLogInfo).toHaveBeenCalledWith(expect.stringContaining('[server] Received SIGTERM, Server stopping'));
    expect(mockAppClose).toHaveBeenCalled();
  });

  test('should handle server shutdown on SIGINT', async () => {
    await import('./server.js');

    process.emit('SIGINT', 'SIGINT');

    expect(mockLogInfo).toHaveBeenCalledWith(expect.stringContaining('[server] Received SIGINT, Server stopping'));
    expect(mockAppClose).toHaveBeenCalled();
  });

  test('should handle server shutdown error', async () => {
    mockAppClose.mockRejectedValueOnce(new Error('Failed to close'));

    await import('./server.js');
    process.emit('SIGTERM', 'SIGTERM');

    await vi.waitFor(() =>
      expect(mockLogError).toHaveBeenCalledWith(
        expect.stringContaining('[server] Error during shutdown:'),
        expect.any(Error),
      ),
    );
  });

  test('should handle uncaught exceptions while server is running', async () => {
    await import('./server.js');

    process.emit('uncaughtException', new Error('Uncaught error'));

    await vi.waitFor(() => {
      expect(mockLogError).toHaveBeenCalledWith(
        expect.stringContaining('[server] Uncaught exception:'),
        expect.any(Error),
      );
    });
  });

  test('should handle unhandled rejections while server is running', async () => {
    await import('./server.js');

    const mockPromise = Promise.reject(new Error('Unhandled rejection'));
    const mockReason = new Error('Unhandled rejection');

    process.emit('unhandledRejection', mockReason, mockPromise);

    await vi.waitFor(() => {
      expect(mockLogError).toHaveBeenCalledWith(
        expect.stringContaining('[server] Unhandled rejection at:'),
        mockPromise,
        expect.stringContaining('reason:'),
        mockReason,
      );
    });
  });
});
