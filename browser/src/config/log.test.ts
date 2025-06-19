// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {vi, describe, it, expect, beforeEach} from 'vitest';
import {FastifyBaseLogger} from 'fastify';
import {createLogger} from './log.js';

describe('log', () => {
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
      const infoSpy = vi.spyOn(console, 'info').mockImplementation(() => {});

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
});
