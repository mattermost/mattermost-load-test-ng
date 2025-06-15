// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {FastifyInstance, FastifyRequest, FastifyReply} from 'fastify';

import {browserTestSessionManager} from '../lib/browser_manager.js';

export default async function browserRoutes(fastify: FastifyInstance) {
  // Register shutdown hook when routes are loaded
  fastify.addHook('onClose', closeBrowser);

  fastify.post('/browsers', addBrowser);
  fastify.delete('/browsers', removeBrowser);
  fastify.get('/browsers', getBrowsers);
}

interface AddBrowserRequestBody {
  userId: string;
  password: string;
}

async function addBrowser(request: FastifyRequest<{Body: AddBrowserRequestBody}>, reply: FastifyReply) {
  const {userId, password} = request.body;

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

  const createInstanceResult = await browserTestSessionManager.createBrowserSession(userId, password);

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

async function removeBrowser(request: FastifyRequest<{Body: RemoveBrowserRequestBody}>, reply: FastifyReply) {
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

async function getBrowsers(_: FastifyRequest, reply: FastifyReply) {
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
