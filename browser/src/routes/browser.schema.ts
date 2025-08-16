// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export const postSchema = {
  tags: ['browser'],
  summary: 'Add a new browser instance',
  description: 'Creates a new browser instance for load testing with the provided user credentials',
  body: {
    type: 'object',
    required: ['user', 'password'],
    properties: {
      user: {
        type: 'string',
        description: 'User ID or email address of the user to create a browser session for',
      },
      password: {
        type: 'string',
        description: 'Password of the user',
      },
      server_url: {
        type: 'string',
        description: 'URL of the Mattermost server',
      },
    },
  },
  response: {
    201: {
      type: 'object',
      properties: {
        success: {type: 'boolean', default: true},
        message: {type: 'string'},
      },
      example: {
        success: true,
        message: 'Browser instance created for user bilalcall',
      },
    },
    400: {
      type: 'object',
      properties: {
        success: {type: 'boolean', default: false},
        error: {
          type: 'object',
          properties: {
            code: {type: 'string'},
            message: {type: 'string'},
          },
        },
      },
      example: {
        success: false,
        error: {
          code: 'CREATE_SESSION_FAILED',
          message: 'Failed to create context for user bilalcall',
        },
      },
    },
  },
};

export const deleteSchema = {
  tags: ['browser'],
  summary: 'Remove a browser instance',
  description: 'Removes an existing browser instance by user',
  querystring: {
    type: 'object',
    required: ['user'],
    properties: {
      user: {
        type: 'string',
        description: 'Username or email of the user to remove the browser session',
      },
    },
  },
  response: {
    200: {
      type: 'object',
      properties: {
        success: {type: 'boolean', default: true},
        message: {type: 'string'},
      },
      example: {
        success: true,
        message: 'Browser instance scheduled for removal for user bilalcall',
      },
    },
    400: {
      type: 'object',
      properties: {
        success: {type: 'boolean', default: false},
        error: {
          type: 'object',
          properties: {
            code: {type: 'string'},
            message: {type: 'string'},
          },
        },
      },
      example: {
        success: false,
        error: {
          code: 'REMOVE_SESSION_FAILED',
          message: 'Browser instance does not exist for user bilalcall',
        },
      },
    },
  },
};

export const getSchema = {
  tags: ['browser'],
  summary: 'Get all active browser instances',
  description:
    'Retrieves information about all active browser instances which might or might not be running simulation tests. It also includes the states each of the browser instances are in.',
  response: {
    200: {
      type: 'object',
      properties: {
        success: {type: 'boolean'},
        data: {
          type: 'object',
          properties: {
            sessions: {
              type: 'array',
              items: {
                type: 'object',
                properties: {
                  userId: {type: 'string'},
                  createdAt: {type: 'string'},
                  state: {type: 'string'},
                },
              },
            },
            count: {type: 'number'},
          },
        },
      },
      'x-examples': {
        noSessions: {
          summary: 'No active sessions',
          description: 'Example showing when no browser sessions are currently active',
          value: {
            success: true,
            data: {
              count: 0,
              sessions: [],
            },
          },
        },
        activeSessions: {
          summary: 'Multiple active sessions',
          description: 'Example showing when there are multiple browser sessions in different states',
          value: {
            success: true,
            data: {
              count: 2,
              sessions: [
                {userId: 'bilalcall', createdAt: '2021-01-01', state: 'creation_failed'},
                {userId: 'salimword', createdAt: '2021-01-01', state: 'started'},
              ],
            },
          },
        },
      },
    },
  },
};
