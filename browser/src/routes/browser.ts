// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {FastifyInstance, FastifyRequest, FastifyReply} from 'fastify';

import {browserTestSessionManager} from '../lib/browser_manager.js';
import {IReply} from './types.js';
import {getMattermostServerURL} from '../utils/config.js';
import {postSchema, deleteSchema, getSchema} from './browser.schema.js';

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
}

async function addBrowser(
  request: FastifyRequest<{Body: AddBrowserRequestBody}>,
  reply: FastifyReply,
): Promise<IReply> {
  const {user, password} = request.body;

  if (!user || user.length === 0) {
    return reply.code(400).send({
      success: false,
      error: {
        code: 'USER_MISSING',
        message: 'username or email is missing',
      },
    });
  }

  if (!password || password.length === 0) {
    return reply.code(400).send({
      success: false,
      error: {
        code: 'PASSWORD_MISSING',
        message: 'password is missing',
      },
    });
  }

  const serverURL = getMattermostServerURL();
  if (!serverURL) {
    return reply.code(400).send({
      success: false,
      error: {
        code: 'MM_SERVER_URL_MISSING',
        message: 'Mattermost server URL is missing in config.json',
      },
    });
  }

  const createInstanceResult = await browserTestSessionManager.createBrowserSession(user, password, serverURL);

  if (!createInstanceResult.isCreated) {
    return reply.code(400).send({
      success: false,
      error: {
        code: 'CREATE_SESSION_FAILED',
        message: createInstanceResult.message,
      },
    });
  }

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
    return reply.code(400).send({
      success: false,
      error: {
        code: 'USER_ID_MISSING',
        message: 'userId is missing',
      },
    });
  }

  const removeInstanceResult = await browserTestSessionManager.removeBrowserSession(user);

  if (!removeInstanceResult.isRemoved) {
    return reply.code(400).send({
      success: false,
      error: {
        code: 'REMOVE_SESSION_FAILED',
        message: removeInstanceResult.message,
      },
    });
  }

  return reply.code(200).send({
    success: true,
    message: removeInstanceResult.message,
  });
}

async function getBrowsers(_: FastifyRequest, reply: FastifyReply): Promise<IReply> {
  const activeSessions = browserTestSessionManager.getActiveBrowserSessions();

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
