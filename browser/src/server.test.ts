// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {describe, test, expect, vi, beforeEach, afterEach, type Mock} from 'vitest';

vi.mock('./app.js', () => {
  const mockApp = {
    listen: vi.fn().mockResolvedValue(undefined),
    close: vi.fn().mockImplementation(() => {
      console.log('[server] Server stopped');
      return Promise.resolve();
    }),
    server: {
      address: vi.fn().mockImplementation(() => {
        return {port: Number(process.env.PORT) || 8080};
      }),
    },
  };
  return {
    app: mockApp,
    createApp: vi.fn().mockReturnValue(mockApp),
  };
});

vi.mock('./config/env.js', () => ({
  loadEnvironmentVariables: vi.fn(),
}));

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
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  test('should start server with default port and host', async () => {
    delete process.env.PORT;
    delete process.env.HOST;

    await import('./server.js');
    const {app} = await import('./app.js');

    expect(app.listen).toHaveBeenCalledWith({port: 8080, host: '127.0.0.1'});
    expect(mockConsoleLog).toHaveBeenCalledWith(
      expect.stringContaining('Server successfully started at 127.0.0.1:8080'),
    );
  });

  test('should start server with environment variables port and host', async () => {
    process.env.PORT = '3000';
    process.env.HOST = '18.212.128.100';

    await import('./server.js');
    const {app} = await import('./app.js');

    expect(app.listen).toHaveBeenCalledWith({port: 3000, host: '18.212.128.100'});
    expect(mockConsoleLog).toHaveBeenCalledWith(
      expect.stringContaining('Server successfully started at 18.212.128.100:3000'),
    );
  });

  test('should handle server start error', async () => {
    const {app} = await import('./app.js');

    (app.listen as Mock).mockRejectedValueOnce(new Error('Failed to start'));

    await import('./server.js');

    expect(mockConsoleError).toHaveBeenCalledWith(expect.stringContaining('Server failed to start'), expect.any(Error));
  });

  test('should handle server shutdown on SIGTERM', async () => {
    await import('./server.js');
    const {app} = await import('./app.js');

    process.emit('SIGTERM', 'SIGTERM');

    expect(mockConsoleLog).toHaveBeenCalledWith(expect.stringContaining('Received SIGTERM, Server stopping'));
    expect(app.close).toHaveBeenCalled();
  });

  test('should handle server shutdown on SIGINT', async () => {
    await import('./server.js');
    const {app} = await import('./app.js');

    process.emit('SIGINT', 'SIGINT');

    expect(mockConsoleLog).toHaveBeenCalledWith(expect.stringContaining('Received SIGINT, Server stopping'));
    expect(app.close).toHaveBeenCalled();
  });

  test('should handle server shutdown error', async () => {
    await import('./server.js');
    const {app} = await import('./app.js');

    (app.close as Mock).mockRejectedValueOnce(new Error('Failed to close'));

    process.emit('SIGTERM', 'SIGTERM');

    await vi.waitFor(() =>
      expect(mockConsoleError).toHaveBeenCalledWith(
        expect.stringContaining('Error during shutdown:'),
        expect.any(Error),
      ),
    );
  });

  test('should handle uncaught exceptions while server is running', async () => {
    await import('./server.js');

    process.emit('uncaughtException', new Error('Uncaught error'));

    await vi.waitFor(() => {
      expect(mockConsoleError).toHaveBeenCalledWith(expect.stringContaining('Uncaught exception:'), expect.any(Error));
    });
  });

  test('should handle unhandled rejections while server is running', async () => {
    await import('./server.js');

    const mockPromise = Promise.reject(new Error('Unhandled rejection'));
    const mockReason = new Error('Unhandled rejection');

    process.emit('unhandledRejection', mockReason, mockPromise);

    await vi.waitFor(() => {
      expect(mockConsoleError).toHaveBeenCalledWith(
        expect.stringContaining('Unhandled rejection at:'),
        mockPromise,
        expect.stringContaining('reason:'),
        mockReason,
      );
    });
  });
});
