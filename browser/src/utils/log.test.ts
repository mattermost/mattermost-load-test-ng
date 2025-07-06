// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {vi, describe, it, expect, beforeEach} from 'vitest';
import {FastifyBaseLogger} from 'fastify';
import {createLogger, getServerLoggerConfig} from './log.js';

vi.mock('./config.js', () => ({
  isConsoleLoggingEnabled: vi.fn(),
  getConsoleLoggingLevel: vi.fn(),
  isFileLoggingEnabled: vi.fn(),
  getFileLoggingLevel: vi.fn(),
  getFileLoggingLocation: vi.fn(),
}));

vi.mock('pino', () => ({
  default: {
    transport: vi.fn(),
  },
}));

vi.mock('path', () => ({
  default: {
    dirname: vi.fn(),
    resolve: vi.fn(),
    join: vi.fn(),
  },
}));

vi.mock('url', () => ({
  fileURLToPath: vi.fn(),
}));

import {
  isConsoleLoggingEnabled,
  getConsoleLoggingLevel,
  isFileLoggingEnabled,
  getFileLoggingLevel,
  getFileLoggingLocation,
} from './config.js';
import pino from 'pino';
import path from 'path';
import {fileURLToPath} from 'url';

describe('createLogger', () => {
  const mockLogger = {
    error: vi.fn(),
    warn: vi.fn(),
    info: vi.fn(),
  } as unknown as FastifyBaseLogger;

  beforeEach(() => {
    vi.resetAllMocks();
  });

  it('should return no-op functions when console logging is disabled', () => {
    const logger = createLogger(mockLogger, false);

    logger.error('test error');
    logger.warn('test warning');
    logger.info('test info');

    expect(mockLogger.error).not.toHaveBeenCalled();
    expect(mockLogger.warn).not.toHaveBeenCalled();
    expect(mockLogger.info).not.toHaveBeenCalled();
  });

  it('should use console functions when logger is not provided', () => {
    const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const infoSpy = vi.spyOn(console, 'log').mockImplementation(() => {});

    const logger = createLogger(undefined, true);

    logger.error('test error');
    logger.warn('test warning');
    logger.info('test info');

    expect(errorSpy).toHaveBeenCalledWith('test error');
    expect(warnSpy).toHaveBeenCalledWith('test warning');
    expect(infoSpy).toHaveBeenCalledWith('test info');
  });

  it('should use provided logger when available', () => {
    const logger = createLogger(mockLogger, true);

    logger.error('test error');
    logger.warn('test warning');
    logger.info('test info');

    expect(mockLogger.error).toHaveBeenCalledWith('test error');
    expect(mockLogger.warn).toHaveBeenCalledWith('test warning');
    expect(mockLogger.info).toHaveBeenCalledWith('test info');
  });
});

describe('getServerLoggerConfig', () => {
  beforeEach(() => {
    vi.resetAllMocks();

    // Setup default path mocks
    vi.mocked(fileURLToPath).mockReturnValue('/mock/path/to/file.js');
    vi.mocked(path.dirname).mockReturnValue('/mock/path/to');
    vi.mocked(path.resolve).mockReturnValue('/mock/root');
    vi.mocked(path.join).mockReturnValue('/mock/root/logs/browser.log');
  });

  it('should return false when both console and file logging are disabled', () => {
    vi.mocked(isConsoleLoggingEnabled).mockReturnValue(false);
    vi.mocked(isFileLoggingEnabled).mockReturnValue(false);

    const result = getServerLoggerConfig();

    expect(result).toBe(false);
  });

  it('should return console-only config when only console logging is enabled', () => {
    vi.mocked(isConsoleLoggingEnabled).mockReturnValue(true);
    vi.mocked(isFileLoggingEnabled).mockReturnValue(false);
    vi.mocked(getConsoleLoggingLevel).mockReturnValue('info');

    const result = getServerLoggerConfig();

    expect(result).toEqual({
      level: 'info',
    });
  });

  it('should return file-only config when only file logging is enabled', () => {
    vi.mocked(isConsoleLoggingEnabled).mockReturnValue(false);
    vi.mocked(isFileLoggingEnabled).mockReturnValue(true);
    vi.mocked(getFileLoggingLevel).mockReturnValue('debug');
    vi.mocked(getFileLoggingLocation).mockReturnValue('logs/browser.log');

    const result = getServerLoggerConfig();

    expect(result).toEqual({
      level: 'debug',
      file: '/mock/root/logs/browser.log',
    });
  });

  it('should return transport config when both console and file logging are enabled', () => {
    const mockTransport = {mockTransport: true};

    vi.mocked(isConsoleLoggingEnabled).mockReturnValue(true);
    vi.mocked(isFileLoggingEnabled).mockReturnValue(true);
    vi.mocked(getConsoleLoggingLevel).mockReturnValue('info');
    vi.mocked(getFileLoggingLevel).mockReturnValue('debug');
    vi.mocked(getFileLoggingLocation).mockReturnValue('logs/browser.log');
    vi.mocked(pino.transport).mockReturnValue(mockTransport);

    const result = getServerLoggerConfig();

    expect(result).toEqual({
      stream: mockTransport,
    });

    expect(pino.transport).toHaveBeenCalledWith({
      targets: [
        {
          target: 'pino/file',
          level: 'info',
          options: {
            destination: 1,
          },
        },
        {
          target: 'pino/file',
          level: 'debug',
          options: {
            destination: '/mock/root/logs/browser.log',
          },
        },
      ],
    });
  });
});
