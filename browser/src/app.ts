// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import fastify, { FastifyInstance } from 'fastify';
import cors from '@fastify/cors';
import ratelimit from '@fastify/rate-limit';

import browserRoutes from './routes/browser.js';
import healthRoutes from './routes/health.js';

export function createApp(options = {}): FastifyInstance {
  const serverOptions = {
    logger: process.env.DEBUG_LOGS === 'true',
    trustProxy: true,
    ...options
  };

  const app = fastify(serverOptions);

  app.register(cors, {
    origin: false,
  });

  app.register(ratelimit, {
    global: true,
    max: Number(process.env.RATE_LIMIT_MAX) || 100,
    timeWindow: process.env.RATE_LIMIT_TIME_WINDOW || '2 minutes',
  });

  app.register(healthRoutes);
  app.register(browserRoutes);

  return app;
}

export const app = createApp();
