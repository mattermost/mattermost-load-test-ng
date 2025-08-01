// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {FastifyInstance, FastifyRequest, FastifyReply} from 'fastify';

import {browserTestSessionManager} from '../lib/browser_manager.js';
import {IReply} from './types.js';
import {postSchema, deleteSchema, getSchema} from './browser.schema.js';
import {SimulationIds} from '../simulations/registry.js';

export default async function browserRoutes(fastify: FastifyInstance) {
  // Register shutdown hook when routes are loaded
  fastify.addHook('onClose', closeBrowser);

  fastify.post<{Reply: IReply; Body: AddBrowserRequestBody}>('/browsers', {schema: postSchema}, addBrowser);
  fastify.delete<{Reply: IReply; Querystring: RemoveBrowserRequestQuery}>(
    '/browsers',
    {schema: deleteSchema},
    removeBrowser,
  );
  fastify.get<{Reply: IReply}>('/browsers', {schema: getSchema}, getBrowsers);
}

interface AddBrowserRequestBody {
  /**
   * user is the username or email of the user to create a browser session for
   */
  user: string;
  password: string;
  server_url: string;
}

async function addBrowser(
  request: FastifyRequest<{Body: AddBrowserRequestBody}>,
  reply: FastifyReply,
): Promise<IReply> {
  const {user, password} = request.body;

  if (!user || user.length === 0) {
    const errorMessage = 'username or email is missing';
    request.log.error(`${request.method}:${request.url} - ${errorMessage}`);
    return reply.code(400).send({
      success: false,
      error: {
        code: 'USER_MISSING',
        message: errorMessage,
      },
    });
  }

  if (!password || password.length === 0) {
    const errorMessage = 'password is missing';
    request.log.error(`${request.method}:${request.url} - ${errorMessage}`);
    return reply.code(400).send({
      success: false,
      error: {
        code: 'PASSWORD_MISSING',
        message: errorMessage,
      },
    });
  }

  // TODO: server url should always be used from the config.json instead of the request body
  // TODO: this is a temporary fix for now
  // const serverURL = getMattermostServerURL();

  const serverURL = request.body.server_url;
  if (!serverURL) {
    const errorMessage = 'Mattermost server URL is empty';
    request.log.error(`${request.method}:${request.url} - ${errorMessage}`);
    return reply.code(400).send({
      success: false,
      error: {
        code: 'MM_SERVER_URL_MISSING',
        message: errorMessage,
      },
    });
  }

  // TODO: make this configurable
  const simulationId = SimulationIds.postAndScroll;

  const createInstanceResult = await browserTestSessionManager.createBrowserSession(
    user,
    password,
    serverURL,
    simulationId,
  );

  if (!createInstanceResult.isCreated) {
    request.log.error(`${request.method}:${request.url} - ${createInstanceResult.message}`);
    return reply.code(400).send({
      success: false,
      error: {
        code: 'CREATE_SESSION_FAILED',
        message: createInstanceResult.message,
      },
    });
  }

  request.log.info(`${request.method}:${request.url} - ${createInstanceResult.message}`);
  return reply.code(201).send({
    success: true,
    message: createInstanceResult.message,
  });
}

interface RemoveBrowserRequestQuery {
  /**
   * user is the username or email of the user to remove the browser session
   */
  user: string;
}

async function removeBrowser(
  request: FastifyRequest<{Querystring: RemoveBrowserRequestQuery}>,
  reply: FastifyReply,
): Promise<IReply> {
  const {user} = request.query;

  if (!user) {
    const errorMessage = 'userId is missing';
    request.log.error(`${request.method}:${request.url} - ${errorMessage}`);
    return reply.code(400).send({
      success: false,
      error: {
        code: 'USER_ID_MISSING',
        message: errorMessage,
      },
    });
  }

  const removeInstanceResult = await browserTestSessionManager.removeBrowserSession(user);

  if (!removeInstanceResult.isRemoved) {
    request.log.error(`${request.method}:${request.url} - ${removeInstanceResult.message}`);
    return reply.code(400).send({
      success: false,
      error: {
        code: 'REMOVE_SESSION_FAILED',
        message: removeInstanceResult.message,
      },
    });
  }

  request.log.info(`${request.method}:${request.url} - ${removeInstanceResult.message}`);
  return reply.code(200).send({
    success: true,
    message: removeInstanceResult.message,
  });
}

async function getBrowsers(request: FastifyRequest, reply: FastifyReply): Promise<IReply> {
  const activeSessions = browserTestSessionManager.getActiveBrowserSessions();

  request.log.info(`${request.method}:${request.url} - ${activeSessions.length} active browser sessions`);
  return reply.code(200).send({
    success: true,
    data: {
      sessions: activeSessions,
      count: activeSessions.length,
    },
  });
}

async function closeBrowser() {
  await browserTestSessionManager.shutdown();
}
