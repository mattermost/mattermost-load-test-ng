// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import fastify, {FastifyInstance} from 'fastify';

import browserRoutes from './routes/browser.js';
import healthRoutes from './routes/health.js';
import {getServerLoggerConfig, createLoggerFunctions} from './config/logging.js';

export function createApp(options = {}): FastifyInstance {
  const serverOptions = {
    logger: getServerLoggerConfig(),
    trustProxy: true,
    ...options,
  };

  const app = fastify(serverOptions);

  app.register(healthRoutes);
  app.register(browserRoutes);

  return app;
}

export const app = createApp();

export const log = createLoggerFunctions(app.log);
