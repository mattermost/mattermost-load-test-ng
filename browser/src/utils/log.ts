// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {FastifyBaseLogger, FastifyLoggerOptions} from 'fastify';
import path from 'path';
import {fileURLToPath} from 'url';
import pino from 'pino';

import {
  isConsoleLoggingEnabled,
  getConsoleLoggingLevel,
  isFileLoggingEnabled,
  getFileLoggingLevel,
  getFileLoggingLocation,
} from './config.js';

export function getServerLoggerConfig(): FastifyLoggerOptions | boolean {
  const consoleLoggingEnabled = isConsoleLoggingEnabled();
  const fileLoggingEnabled = isFileLoggingEnabled();

  if (!consoleLoggingEnabled && !fileLoggingEnabled) {
    return false;
  }

  if (consoleLoggingEnabled && !fileLoggingEnabled) {
    return {
      level: getConsoleLoggingLevel(),
    };
  }

  const dirname = path.dirname(fileURLToPath(import.meta.url));
  const rootDir = path.resolve(dirname, '../../..');
  const filePath = path.join(rootDir, getFileLoggingLocation());

  if (!consoleLoggingEnabled && fileLoggingEnabled) {
    return {
      level: getFileLoggingLevel(),
      file: filePath,
    };
  }

  // When both console and file logging are enabled, create transport stream
  // and use it for both console and file logging simultaneously
  const transport = pino.transport({
    targets: [
      {
        target: 'pino/file',
        level: getConsoleLoggingLevel(),
        options: {
          destination: 1, // standard output to console
        },
      },
      {
        target: 'pino/file',
        level: getFileLoggingLevel(),
        options: {
          destination: filePath,
        },
      },
    ],
  });

  return {
    stream: transport,
  };
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
      info: console.log,
    };
  }

  return {
    error: logger.error.bind(logger),
    warn: logger.warn.bind(logger),
    info: logger.info.bind(logger),
  };
}
