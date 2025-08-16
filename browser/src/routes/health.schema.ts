// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export const getSchema = {
  tags: ['health'],
  summary: 'Health check endpoint',
  description: 'Returns server health status, uptime, and system information',
  response: {
    200: {
      type: 'object',
      properties: {
        success: {type: 'boolean', default: true},
        data: {
          type: 'object',
          properties: {
            startTime: {
              type: 'string',
              format: 'date-time',
              description: 'Server start time in ISO format',
            },
            uptime: {
              type: 'string',
              description: 'Server uptime in human-readable format',
            },
            hostname: {
              type: 'string',
              description: 'Server hostname',
            },
            platform: {
              type: 'string',
              description: 'Operating system platform',
            },
          },
        },
      },
      example: {
        success: true,
        data: {
          startTime: '2025-07-14T07:46:11.922Z',
          uptime: '4m',
          hostname: 'mm-macbook-pro.local',
          platform: 'darwin',
        },
      },
    },
  },
};
