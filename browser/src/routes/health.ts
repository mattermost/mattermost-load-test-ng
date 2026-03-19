// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import os from 'os';

import type {FastifyInstance, FastifyReply, FastifyRequest} from 'fastify';
import ms from 'ms';

import {getSchema} from './health.schema.js';
import {type IReply} from './types.js';

interface HealthDataResponse {
  startTime: string;
  uptime: string;
  hostname: string;
  platform: string;
}

// Server metadata
const serverStartTime = new Date();
const hostname = os.hostname();
const platform = process.platform;

export default async function healthRoutes(fastify: FastifyInstance) {
  fastify.get<{Reply: IReply}>('/health', {schema: getSchema}, getHealth);
}

export async function getHealth(_: FastifyRequest, reply: FastifyReply): Promise<IReply> {
  const uptime = Math.floor(Date.now() - serverStartTime.getTime());
  const healthData: HealthDataResponse = {
    startTime: serverStartTime.toISOString(),
    uptime: ms(uptime),
    hostname,
    platform,
  };

  return reply.code(200).send({
    success: true,
    data: healthData,
  });
}
