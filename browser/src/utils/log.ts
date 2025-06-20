// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {FastifyBaseLogger, FastifyLoggerOptions} from 'fastify';
import path from 'path';
import {fileURLToPath} from 'url';

import {
  isConsoleLoggingEnabled,
  getConsoleLoggingLevel,
  getFileLoggingLocation,
  isFileLoggingEnabled,
} from './config.js';

export function getServerLoggerConfig(): FastifyLoggerOptions | boolean {
  if (!isConsoleLoggingEnabled()) {
    return false;
  }

  const loggerConfig: FastifyLoggerOptions = {
    level: getConsoleLoggingLevel(),
  };

  if (isFileLoggingEnabled()) {
    const dirname = path.dirname(fileURLToPath(import.meta.url));
    const rootDir = path.resolve(dirname, '../../..');
    const filePath = path.join(rootDir, getFileLoggingLocation());

    loggerConfig.file = filePath;
  }

  return loggerConfig;
}

export function createLogger(logger?: FastifyBaseLogger, isEnabled = true) {
  if (!isEnabled) {
    return {
      error: () => {},
      warn: () => {},
      info: () => {},
    };
  }

  if (!logger) {
    return {
      error: console.error,
      warn: console.warn,
      info: console.info,
    };
  }

  return {
    error: (message: string) => {
      return logger.error(message);
    },

    warn: (message: string) => {
      return logger.warn(message);
    },

    info: (message: string) => {
      return logger.info(message);
    },
  };
}
