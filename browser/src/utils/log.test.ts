// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {vi, describe, it, expect, beforeEach} from 'vitest';
import {FastifyBaseLogger} from 'fastify';

// Mock all the dependencies with simple implementations
vi.mock('./config.js', () => ({
  isConsoleLoggingEnabled: vi.fn(() => false),
  getConsoleLoggingLevel: vi.fn(() => 'info'),
  isFileLoggingEnabled: vi.fn(() => false),
  getFileLoggingLevel: vi.fn(() => 'debug'),
  getFileLoggingLocation: vi.fn(() => 'logs/browser.log'),
}));

vi.mock('path', () => ({
  default: {
    dirname: vi.fn(() => '/mock/path/to'),
    resolve: vi.fn(() => '/mock/root'),
    join: vi.fn(() => '/mock/root/logs/browser.log'),
  },
}));

vi.mock('url', () => ({
  fileURLToPath: vi.fn(() => '/mock/path/to/file.js'),
}));

const {mockTransport, mockPino} = vi.hoisted(() => {
  const mockTransport = vi.fn();
  const mockPino = vi.fn();
  // @ts-expect-error pino-caller is not ESM compatible yet
  mockPino.transport = mockTransport;
  return {mockTransport, mockPino};
});

vi.mock('module', () => {
  const mockRequire = vi.fn((moduleName) => {
    if (moduleName === 'pino') {
      return mockPino;
    }
    return {};
  });

  return {
    createRequire: vi.fn(() => mockRequire),
  };
});

vi.mock('pino-caller', () => ({
  default: vi.fn((logger) => logger),
}));

import {createLogger, getServerLoggerConfig} from './log.js';
import {
  isConsoleLoggingEnabled,
  getConsoleLoggingLevel,
  isFileLoggingEnabled,
  getFileLoggingLevel,
  getFileLoggingLocation,
} from './config.js';

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

    errorSpy.mockRestore();
    warnSpy.mockRestore();
    infoSpy.mockRestore();
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
    mockTransport.mockReturnValue({stream: 'mock'});
  });

  it('should return false when both console and file logging are disabled', () => {
    vi.mocked(isConsoleLoggingEnabled).mockReturnValue(false);
    vi.mocked(isFileLoggingEnabled).mockReturnValue(false);

    const result = getServerLoggerConfig();

    expect(result).toEqual({
      stream: {stream: 'mock'},
      serializers: {},
    });
    expect(mockTransport).toHaveBeenCalledWith({
      targets: [],
    });
  });

  it('should return transport config when only console logging is enabled', () => {
    vi.mocked(isConsoleLoggingEnabled).mockReturnValue(true);
    vi.mocked(isFileLoggingEnabled).mockReturnValue(false);
    vi.mocked(getConsoleLoggingLevel).mockReturnValue('info');

    const result = getServerLoggerConfig();

    expect(result).toEqual({
      stream: {stream: 'mock'},
      serializers: expect.any(Object),
    });
    expect(mockTransport).toHaveBeenCalledWith({
      targets: expect.arrayContaining([
        expect.objectContaining({
          target: 'pino-pretty',
          level: 'info',
        }),
      ]),
    });
  });

  it('should return transport config when only file logging is enabled', () => {
    vi.mocked(isConsoleLoggingEnabled).mockReturnValue(false);
    vi.mocked(isFileLoggingEnabled).mockReturnValue(true);
    vi.mocked(getFileLoggingLevel).mockReturnValue('debug');
    vi.mocked(getFileLoggingLocation).mockReturnValue('logs/browser.log');

    const result = getServerLoggerConfig();

    expect(result).toEqual({
      stream: {stream: 'mock'},
      serializers: expect.any(Object),
    });
    expect(mockTransport).toHaveBeenCalledWith({
      targets: expect.arrayContaining([
        expect.objectContaining({
          target: 'pino-pretty',
          level: 'debug',
        }),
      ]),
    });
  });

  it('should return transport config when both console and file logging are enabled', () => {
    vi.mocked(isConsoleLoggingEnabled).mockReturnValue(true);
    vi.mocked(isFileLoggingEnabled).mockReturnValue(true);
    vi.mocked(getConsoleLoggingLevel).mockReturnValue('info');
    vi.mocked(getFileLoggingLevel).mockReturnValue('debug');
    vi.mocked(getFileLoggingLocation).mockReturnValue('logs/browser.log');

    const result = getServerLoggerConfig();

    expect(result).toEqual({
      stream: {stream: 'mock'},
      serializers: expect.any(Object),
    });
    expect(mockTransport).toHaveBeenCalledWith({
      targets: expect.arrayContaining([
        expect.objectContaining({
          target: 'pino-pretty',
          level: 'debug',
        }),
        expect.objectContaining({
          target: 'pino-pretty',
          level: 'info',
        }),
      ]),
    });
  });
});
