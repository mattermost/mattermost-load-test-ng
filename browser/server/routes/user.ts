import {FastifyInstance} from 'fastify';

interface BrowserAddBody {
  userId: string;
  password: string;
}

/**
 * Register browser management routes with the Fastify instance
 */
export default async function browserRoutes(fastify: FastifyInstance) {
  // Start a browser instance for a user
  fastify.post<{Body: BrowserAddBody}>('/browser/add', async (request, reply) => {
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

    // TODO: Add logic to add a browser instance for a user
  });

  // Stop a browser instance for a user
  fastify.post('/browser/remove', async (request, reply) => {
    // TODO: Remove the oldest added browser instance
  });

  fastify.get('/browser', async (request, reply) => {
    // TODO: Get all browser instances and more
  });
}
