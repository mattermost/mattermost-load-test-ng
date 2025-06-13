import {FastifyInstance, FastifyRequest, FastifyReply} from 'fastify';

import {browserManager} from '../lib/browser_manager.js';

interface BrowserAddBody {
  userId: string;
  password: string;
}

export default async function browserRoutes(fastify: FastifyInstance) {
  fastify.post('/browsers', addBrowser);
  fastify.delete('/browsers', removeBrowser);
  fastify.get('/browsers', getBrowsers);
}

async function addBrowser(request: FastifyRequest<{Body: BrowserAddBody}>, reply: FastifyReply) {
  const {userId, password} = request.body;

  if (!userId) {
    return reply.code(400).send({
      success: false,
      error: 'userId is missing',
    });
  }

  if (!password) {
    return reply.code(400).send({
      success: false,
      error: 'password is missing',
    });
  }

  const createInstanceResult = await browserManager.createInstance(userId, password);

  if (!createInstanceResult.isCreated) {
    return reply.code(400).send({
      success: false,
      error: createInstanceResult.message,
    });
  }

  return reply.code(200).send({
    success: true,
    message: createInstanceResult.message,
  });
}

async function removeBrowser() {}

async function getBrowsers(_: FastifyRequest, reply: FastifyReply) {
  return reply.code(200).send({
    success: true,
    instances: browserManager.getAllInstances(),
    count: browserManager.getInstancesCount(),
  });
}
