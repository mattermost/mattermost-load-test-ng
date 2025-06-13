import {FastifyInstance} from 'fastify';
import os from 'os';

// Server metadata
const serverStartTime = new Date();

export default async function healthRoutes(fastify: FastifyInstance) {
  fastify.get('/health', getHealth);
}

async function getHealth() {
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
}
