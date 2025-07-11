// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {FastifyBaseLogger, FastifyLoggerOptions} from 'fastify';
import path from 'path';
import {fileURLToPath} from 'url';
import {createRequire} from 'module';
import {TransportTargetOptions} from 'pino';
import pinoCaller from 'pino-caller';
import {PrettyOptions} from 'pino-pretty';

const require = createRequire(import.meta.url);
const pino = require('pino');

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

  const dirname = path.dirname(fileURLToPath(import.meta.url));
  const rootDir = path.resolve(dirname, '../../..');
  const filePath = path.join(rootDir, getFileLoggingLocation());

  const commonPinoPrettyOptions: PrettyOptions = {
    colorize: false,
    translateTime: 'SYS:yyyy-mm-dd HH:MM:ss.l p',
    singleLine: true,
    hideObject: false,
    levelFirst: true,
  };

  const fileLoggingTransport: TransportTargetOptions = {
    target: 'pino-pretty',
    level: getFileLoggingLevel(),
    options: {
      destination: filePath,
      ...commonPinoPrettyOptions,
    },
  };
  const consoleLoggingTransport: TransportTargetOptions = {
    target: 'pino-pretty',
    level: getConsoleLoggingLevel(),
    options: {
      destination: 1, // standard output to console
      ...commonPinoPrettyOptions,
    },
  };

  if (consoleLoggingEnabled && !fileLoggingEnabled) {
    return {
      stream: pino.transport({
        targets: [consoleLoggingTransport],
      }),
    };
  }

  if (!consoleLoggingEnabled && fileLoggingEnabled) {
    return {
      stream: pino.transport({
        targets: [fileLoggingTransport],
      }),
    };
  }

  return {
    stream: pino.transport({
      targets: [fileLoggingTransport, consoleLoggingTransport],
    }),
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

  // @ts-expect-error pino-caller is not ESM compatible yet
  const pinoWithCaller = pinoCaller(logger, {
    relativeTo: path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..'),
  });

  return {
    error: pinoWithCaller.error.bind(pinoWithCaller),
    warn: pinoWithCaller.warn.bind(pinoWithCaller),
    info: pinoWithCaller.info.bind(pinoWithCaller),
  };
}
