// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {FastifyBaseLogger, FastifyLoggerOptions} from 'fastify';
import {configJson} from './config.js';

const isConsoleLoggingEnabled = configJson.BrowserLogSettings.EnableConsole;
const consoleLoggingLevel = configJson.BrowserLogSettings.ConsoleLevel;

export const serverLoggerConfig: FastifyLoggerOptions = {
  level: isConsoleLoggingEnabled ? consoleLoggingLevel : 'silent',
};

export function getServerLoggerConfig(): FastifyLoggerOptions | boolean {
  if (!isConsoleLoggingEnabled) {
    return false;
  }

  return {
    level: consoleLoggingLevel,
  };
}

export function createLoggerFunctions(logger: FastifyBaseLogger) {
  if (!isConsoleLoggingEnabled) {
    return {
      error: () => {},
      warn: () => {},
      info: () => {},
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
