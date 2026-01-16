// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {FastifyInstance} from 'fastify';
import supertest from 'supertest';
import {describe, test, expect, vi, beforeEach, afterEach} from 'vitest';

import {createApp} from '../app.js';

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

    expect(mockFastify.get).toHaveBeenCalledWith(
      '/health',
      expect.objectContaining({schema: expect.any(Object)}),
      expect.any(Function),
    );
  });
});

describe('API /health', () => {
  const MIN_PORT = 10000;
  const MAX_PORT = 65000;
  let appInstance: FastifyInstance;
  let port: number;

  beforeEach(async () => {
    appInstance = createApp({logger: false});
    port = Math.floor(Math.random() * (MAX_PORT - MIN_PORT + 1)) + MIN_PORT;
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
      success: true,
      data: {
        startTime: expect.any(String),
        uptime: expect.any(String),
        hostname: expect.any(String),
        platform: expect.any(String),
      },
    });
  });
});
