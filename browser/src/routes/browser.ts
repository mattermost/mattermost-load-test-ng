// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {FastifyInstance, FastifyRequest, FastifyReply} from 'fastify';

import {browserTestSessionManager} from '../lib/browser_manager.js';
import {IReply} from './types.js';
import {getMattermostServerURL} from '../utils/config.js';

export default async function browserRoutes(fastify: FastifyInstance) {
  // Register shutdown hook when routes are loaded
  fastify.addHook('onClose', closeBrowser);

  fastify.post<{Reply: IReply; Body: AddBrowserRequestBody}>('/browsers', addBrowser);
  fastify.delete<{Reply: IReply; Body: RemoveBrowserRequestBody}>('/browsers', removeBrowser);
  fastify.get<{Reply: IReply}>('/browsers', getBrowsers);
}

interface AddBrowserRequestBody {
  userId: string;
  password: string;
}

async function addBrowser(
  request: FastifyRequest<{Body: AddBrowserRequestBody}>,
  reply: FastifyReply,
): Promise<IReply> {
  const {userId, password} = request.body;
  console.log('addBrowser', userId, password);

  if (!userId) {
    return reply.code(400).send({
      success: false,
      error: {
        code: 'USER_ID_MISSING',
        message: 'userId is missing',
      },
    });
  }

  if (!password) {
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
        code: 'SERVER_URL_MISSING',
        message: 'serverURL is missing in config.json',
      },
    });
  }

  const createInstanceResult = await browserTestSessionManager.createBrowserSession(userId, password, serverURL);

  if (!createInstanceResult.isCreated) {
    return reply.code(400).send({
      success: false,
      error: {
        code: 'CREATE_SESSION_FAILED',
        message: createInstanceResult.message,
      },
    });
  }

  return reply.code(200).send({
    success: true,
    message: createInstanceResult.message,
  });
}

interface RemoveBrowserRequestBody {
  userId: string;
}

async function removeBrowser(
  request: FastifyRequest<{Body: RemoveBrowserRequestBody}>,
  reply: FastifyReply,
): Promise<IReply> {
  const {userId} = request.body;

  if (!userId) {
    return reply.code(400).send({
      success: false,
      error: {
        code: 'USER_ID_MISSING',
        message: 'userId is missing',
      },
    });
  }

  const removeInstanceResult = await browserTestSessionManager.removeBrowserSession(userId);

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
