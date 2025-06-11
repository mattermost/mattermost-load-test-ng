import {FastifyInstance} from 'fastify';
import os from 'os';

// Server metadata
const serverStartTime = new Date();

async function healthRoutes(fastify: FastifyInstance) {
  fastify.get('/health', async () => {
    const uptime = Math.floor((Date.now() - serverStartTime.getTime()) / 1000);

    return {
      status: 'ok',
      serverInfo: {
        startTime: serverStartTime.toISOString(),
        uptime: `${uptime} seconds`,
        hostname: os.hostname(),
        platform: process.platform,
      },
    };
  });
}

export default healthRoutes;
