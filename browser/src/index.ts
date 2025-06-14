// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import fastify from 'fastify';
import cors from '@fastify/cors';
import ratelimit from '@fastify/rate-limit';

import browserRoutes from './routes/browser.js';
import healthRoutes from './routes/health.js';
import {loadEnvironmentVariables} from './utils/env.js';

loadEnvironmentVariables();

const serverOptions = {
  logger: process.env.DEBUG_LOGS === 'true',
  trustProxy: true,
};

const server = fastify(serverOptions);

await server.register(cors, {
  origin: false,
});

await server.register(ratelimit, {
  global: true,
  max: Number(process.env.RATE_LIMIT_MAX) || 100,
  timeWindow: process.env.RATE_LIMIT_TIME_WINDOW || '2 minutes',
});

server.register(healthRoutes);
server.register(browserRoutes);

async function startServer() {
  try {
    const portNumber = Number(process.env.PORT) || 8080;
    const host = process.env.HOST || '127.0.0.1';
    await server.listen({port: portNumber, host});

    const address = server.server.address();
    const port = typeof address === 'string' ? address : address?.port;

    console.log(`Server started at ${host}:${port}`);
  } catch (err) {
    console.error('Error starting server', err);
    process.exit(1);
  }
}

startServer();
