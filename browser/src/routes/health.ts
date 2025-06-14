// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {FastifyInstance} from 'fastify';
import os from 'os';
import ms from 'ms';

// Server metadata
const serverStartTime = new Date();

export default async function healthRoutes(fastify: FastifyInstance) {
  fastify.get('/health', getHealth);
}

async function getHealth() {
  const uptime = Math.floor(Date.now() - serverStartTime.getTime());

  return {
    success: true,
    data: {
      startTime: serverStartTime.toISOString(),
      uptime: ms(uptime),
      hostname: os.hostname(),
      platform: process.platform,
    },
  };
}
