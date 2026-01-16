// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {createRequire} from 'module';
import path from 'path';
import {fileURLToPath} from 'url';

import type {FastifyBaseLogger, FastifyLoggerOptions, FastifyRequest, FastifyReply, RawServerBase} from 'fastify';
import type {ResSerializerReply} from 'fastify/types/logger.js';
import type {TransportTargetOptions} from 'pino';
import pinoCaller from 'pino-caller';
import type {PrettyOptions} from 'pino-pretty';

// We need to use the require function to import pino and pino-pretty
// because they are not ESM compatible yet.
const require = createRequire(import.meta.url);
import {
  isConsoleLoggingEnabled,
  getConsoleLoggingLevel,
  isFileLoggingEnabled,
  getFileLoggingLevel,
  getFileLoggingLocation,
} from './config_accessors.js';

const pino = require('pino');

export function getServerLoggerConfig(): FastifyLoggerOptions {
  const consoleLoggingEnabled = isConsoleLoggingEnabled();
  const fileLoggingEnabled = isFileLoggingEnabled();

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

  const targets: TransportTargetOptions[] = [];
  if (consoleLoggingEnabled) {
    targets.push(consoleLoggingTransport);
  }
  if (fileLoggingEnabled) {
    targets.push(fileLoggingTransport);
  }

  let serializers: FastifyLoggerOptions['serializers'] = {};
  if (consoleLoggingEnabled || fileLoggingEnabled) {
    serializers = {
      req: (request: FastifyRequest) => {
        return {
          method: request.method,
          url: request.url,
          path: request.routeOptions.url,
          host: request.hostname,
          remoteAddress: request.ip,
          remotePort: request.socket.remotePort,
          parameters: request.params,
          query: request.query,
        };
      },
      res: (reply: ResSerializerReply<RawServerBase, FastifyReply>) => {
        return {
          statusCode: reply.statusCode,
        };
      },
    };
  }

  return {
    stream: pino.transport({targets}),
    serializers,
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
