// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {describe, test, expect, vi, beforeEach, afterEach} from 'vitest';
import supertest from 'supertest';
import {FastifyInstance} from 'fastify';

import {createApp} from '../app.js';
import {getRandomPortForTests} from '../utils/config.js';

vi.mock('os', () => {
  return {
    default: {
      hostname: vi.fn().mockReturnValue('test-host'),
    },
  };
});

vi.mock('ms', () => {
  return {
    default: vi.fn((ms) => `${ms}ms`),
  };
});

describe('API /health', () => {
  const mockFastify = {
    get: vi.fn(),
  };

  beforeEach(() => {
    vi.resetModules();
    mockFastify.get.mockClear();
  });

  test('should register health route', async () => {
    const {default: healthRoutes} = await import('./health.js');

    await healthRoutes(mockFastify as any);

    expect(mockFastify.get).toHaveBeenCalledWith('/health', expect.any(Function));
  });
});

describe('API /health', () => {
  let appInstance: FastifyInstance;
  let port: number;

  beforeEach(async () => {
    appInstance = createApp({logger: false});
    port = getRandomPortForTests();
  });

  afterEach(async () => {
    if (appInstance) {
      await appInstance.close();
    }
  });

  test('GET /health should return health data', async () => {
    await appInstance.listen({port});

    const response = await supertest(`http://localhost:${port}`)
      .get('/health')
      .expect(200)
      .expect('Content-Type', /json/);

    expect(response.body).toEqual({
      200: {
        success: true,
        data: {
          startTime: expect.any(String),
          uptime: expect.any(String),
          hostname: expect.any(String),
          platform: expect.any(String),
        },
      },
    });
  });
});
