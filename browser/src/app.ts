// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import swagger from '@fastify/swagger';
import type {FastifyInstance, FastifyServerOptions} from 'fastify';
import fastify from 'fastify';
import {type Ajv} from 'ajv';

import browserRoutes from './routes/browser.js';
import healthRoutes from './routes/health.js';
import {isConsoleLoggingEnabled} from './utils/config_accessors.js';
import {getServerLoggerConfig, createLogger} from './utils/log.js';

export async function applyMiddleware(fastifyInstance: FastifyInstance) {
  const baseSchema = {
    openapi: {
      info: {
        title: 'LTBrowser API',
        description: 'API for managing browser instances in load testing',
        version: '0.1.0',
      },
    },
    hideUntagged: true,
    exposeRoute: true,
  };
  await fastifyInstance.register(swagger, baseSchema);

  await fastifyInstance.register(browserRoutes);
  await fastifyInstance.register(healthRoutes);
}

export function createApp(options?: FastifyServerOptions): FastifyInstance {
  const serverOptions = {
    logger: getServerLoggerConfig(),
    trustProxy: true,
    ajv: {
      plugins: [
        function (ajv: Ajv) {
          // This is used to whitelist the x-examples in the OpenAPI schema
          ajv.addKeyword({keyword: 'x-examples'});
          return ajv;
        },
      ],
    },
    ...options,
  };

  const app = fastify(serverOptions);
  app.register(applyMiddleware);

  return app;
}

export const app = createApp();

export const log = createLogger(app.log, isConsoleLoggingEnabled());

/**
 * This is used by the Fastify CLI to generate the OpenAPI schema. This needs to be exported as default.
 */
export default async function schema(fastifyInstance: FastifyInstance) {
  await applyMiddleware(fastifyInstance);
}
