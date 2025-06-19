// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {FastifyBaseLogger, FastifyLoggerOptions} from 'fastify';
import {configJson} from './config.js';

// Use functions to get config values to make testing easier
export function isConsoleLoggingEnabled(): boolean {
  return configJson.BrowserLogSettings.EnableConsole;
}

export function getConsoleLoggingLevel(): string {
  return configJson.BrowserLogSettings.ConsoleLevel;
}

export const serverLoggerConfig: FastifyLoggerOptions = {
  level: isConsoleLoggingEnabled() ? getConsoleLoggingLevel() : 'silent',
};

export function getServerLoggerConfig(): FastifyLoggerOptions | boolean {
  if (!isConsoleLoggingEnabled()) {
    return false;
  }

  return {
    level: getConsoleLoggingLevel(),
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
