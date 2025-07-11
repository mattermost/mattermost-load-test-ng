// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {describe, expect, test, beforeEach, afterEach, vi} from 'vitest';
import supertest from 'supertest';
import {FastifyInstance} from 'fastify';

import {createApp} from '../app.js';

vi.mock('../utils/config.js', async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as any),
    getMattermostServerURL: vi.fn().mockReturnValue('http://localhost:8065'),
    isBrowserHeadless: vi.fn().mockReturnValue(true),
  };
});

vi.mock('playwright', () => {
  const mockPageClose = vi.fn().mockResolvedValue(undefined);
  const mockPage = {
    close: mockPageClose,
    goto: vi.fn().mockResolvedValue(undefined),
    fill: vi.fn().mockResolvedValue(undefined),
    click: vi.fn().mockResolvedValue(undefined),
    waitForSelector: vi.fn().mockResolvedValue(undefined),
    waitForTimeout: vi.fn().mockResolvedValue(undefined),
    waitForLoadState: vi.fn().mockResolvedValue(undefined),
    locator: vi.fn().mockReturnValue({
      click: vi.fn().mockResolvedValue(undefined),
      fill: vi.fn().mockResolvedValue(undefined),
      waitFor: vi.fn().mockResolvedValue(undefined),
      isVisible: vi.fn().mockResolvedValue(true),
    }),
    getByRole: vi.fn().mockReturnValue({
      click: vi.fn().mockResolvedValue(undefined),
      fill: vi.fn().mockResolvedValue(undefined),
      waitFor: vi.fn().mockResolvedValue(undefined),
    }),
    getByText: vi.fn().mockReturnValue({
      click: vi.fn().mockResolvedValue(undefined),
      waitFor: vi.fn().mockResolvedValue(undefined),
    }),
    getByTestId: vi.fn().mockReturnValue({
      click: vi.fn().mockResolvedValue(undefined),
      fill: vi.fn().mockResolvedValue(undefined),
      waitFor: vi.fn().mockResolvedValue(undefined),
    }),
    url: vi.fn().mockReturnValue('http://localhost:8065'),
    title: vi.fn().mockResolvedValue('Mattermost'),
  };

  const mockContextNewPage = vi.fn().mockResolvedValue(mockPage);
  const mockContextClose = vi.fn().mockResolvedValue(undefined);
  const mockContext = {
    newPage: mockContextNewPage,
    close: mockContextClose,
  };

  const mockBrowserNewContext = vi.fn().mockResolvedValue(mockContext);
  const mockBrowserClose = vi.fn().mockResolvedValue(undefined);
  const mockBrowser = {
    newContext: mockBrowserNewContext,
    close: mockBrowserClose,
  };

  const mockChromiumLaunch = vi.fn().mockResolvedValue(mockBrowser);

  return {
    chromium: {
      launch: mockChromiumLaunch,
    },
  };
});

// Mock the test scenarios to prevent background scenario execution errors
vi.mock('../simulations/noop_scenario.js', () => ({
  noopScenario: vi.fn().mockResolvedValue(undefined),
}));

describe('API /browsers', () => {
  const MIN_PORT = 10000;
  const MAX_PORT = 65000;
  let app: FastifyInstance;
  let port: number;

  beforeEach(async () => {
    // Create app instance with real browser manager
    app = createApp({logger: false});
    port = Math.floor(Math.random() * (MAX_PORT - MIN_PORT + 1)) + MIN_PORT;
  });

  afterEach(async () => {
    if (app) {
      await app.close();
    }
  });

  test('should register browser routes', async () => {
    const fastify = {
      get: vi.fn(),
      post: vi.fn(),
      delete: vi.fn(),
      addHook: vi.fn(),
    };

    const {default: browserRoutes} = await import('./browser.js');

    await browserRoutes(fastify as any);

    expect(fastify.get).toHaveBeenCalledWith('/browsers', expect.any(Function));
    expect(fastify.post).toHaveBeenCalledWith('/browsers', expect.any(Function));
    expect(fastify.delete).toHaveBeenCalledWith('/browsers', expect.any(Function));
    expect(fastify.addHook).toHaveBeenCalledWith('onClose', expect.any(Function));
  });

  describe('POST /browsers', () => {
    test('should successfully create a browser session with real browser manager', async () => {
      await app.listen({port});

      const response = await supertest(`http://localhost:${port}`)
        .post('/browsers')
        .send({
          userId: 'integration-test-user',
          password: 'testpassword',
        })
        .expect(200)
        .expect('Content-Type', /json/);

      expect(response.body).toEqual({
        success: true,
        message: expect.stringContaining('Browser instance created for user integration-test-user'),
      });
    });

    test('should return 400 when userId is missing', async () => {
      await app.listen({port});

      const response = await supertest(`http://localhost:${port}`)
        .post('/browsers')
        .send({
          password: 'testpassword',
        })
        .expect(400)
        .expect('Content-Type', /json/);

      expect(response.body).toEqual({
        success: false,
        error: {
          code: 'USER_ID_MISSING',
          message: 'userId is missing',
        },
      });
    });

    test('should return 400 when userId is empty string', async () => {
      await app.listen({port});

      const response = await supertest(`http://localhost:${port}`)
        .post('/browsers')
        .send({
          userId: '',
          password: 'testpassword',
        })
        .expect(400)
        .expect('Content-Type', /json/);

      expect(response.body).toEqual({
        success: false,
        error: {
          code: 'USER_ID_MISSING',
          message: 'userId is missing',
        },
      });
    });

    test('should return 400 when password is missing', async () => {
      await app.listen({port});

      const response = await supertest(`http://localhost:${port}`)
        .post('/browsers')
        .send({
          userId: 'testuser',
        })
        .expect(400)
        .expect('Content-Type', /json/);

      expect(response.body).toEqual({
        success: false,
        error: {
          code: 'PASSWORD_MISSING',
          message: 'password is missing',
        },
      });
    });

    test('should return 400 when password is empty string', async () => {
      await app.listen({port});

      const response = await supertest(`http://localhost:${port}`)
        .post('/browsers')
        .send({
          userId: 'testuser',
          password: '',
        })
        .expect(400)
        .expect('Content-Type', /json/);

      expect(response.body).toEqual({
        success: false,
        error: {
          code: 'PASSWORD_MISSING',
          message: 'password is missing',
        },
      });
    });

    test('should return 400 when both userId and password are missing', async () => {
      await app.listen({port});

      const response = await supertest(`http://localhost:${port}`)
        .post('/browsers')
        .send({})
        .expect(400)
        .expect('Content-Type', /json/);

      expect(response.body).toEqual({
        success: false,
        error: {
          code: 'USER_ID_MISSING',
          message: 'userId is missing',
        },
      });
    });

    test('should return 400 when server URL is missing in config', async () => {
      // Mock getMattermostServerURL to return empty string
      const {getMattermostServerURL} = await import('../utils/config.js');
      vi.mocked(getMattermostServerURL).mockReturnValueOnce('');

      await app.listen({port});

      const response = await supertest(`http://localhost:${port}`)
        .post('/browsers')
        .send({
          userId: 'testuser',
          password: 'testpassword',
        })
        .expect(400)
        .expect('Content-Type', /json/);

      expect(response.body).toEqual({
        success: false,
        error: {
          code: 'MM_SERVER_URL_MISSING',
          message: 'Mattermost server URL is missing in config.json',
        },
      });
    });

    test('should reject duplicate session creation for same userId', async () => {
      await app.listen({port});

      // First request should succeed
      const response1 = await supertest(`http://localhost:${port}`)
        .post('/browsers')
        .send({
          userId: 'duplicate-test-user',
          password: 'testpassword',
        })
        .expect(200);

      expect(response1.body.success).toBe(true);

      // Second request with same userId should fail
      const response2 = await supertest(`http://localhost:${port}`)
        .post('/browsers')
        .send({
          userId: 'duplicate-test-user',
          password: 'testpassword',
        })
        .expect(400);

      expect(response2.body).toEqual({
        success: false,
        error: {
          code: 'CREATE_SESSION_FAILED',
          message: expect.stringContaining('already exists'),
        },
      });
    });

    test('should handle concurrent requests for same userId correctly', async () => {
      await app.listen({port});

      // Send two requests with same userId simultaneously
      const [response1, response2] = await Promise.allSettled([
        supertest(`http://localhost:${port}`).post('/browsers').send({
          userId: 'concurrent-test-user',
          password: 'testpassword',
        }),
        supertest(`http://localhost:${port}`).post('/browsers').send({
          userId: 'concurrent-test-user',
          password: 'testpassword',
        }),
      ]);

      // One should succeed, one should fail
      const responses = [response1, response2]
        .map((result) => (result.status === 'fulfilled' ? result.value : null))
        .filter(Boolean);

      const successResponses = responses.filter((r) => r!.status === 200);
      const failureResponses = responses.filter((r) => r!.status === 400);

      expect(successResponses).toHaveLength(1);
      expect(failureResponses).toHaveLength(1);

      expect(successResponses[0]!.body.success).toBe(true);
      expect(failureResponses[0]!.body.success).toBe(false);
      expect(failureResponses[0]!.body.error.code).toBe('CREATE_SESSION_FAILED');
    });

    test('should handle rapid successive requests for same userId (race condition test)', async () => {
      await app.listen({port});

      const userId = 'race-condition-user';
      const numRequests = 5;

      // Send multiple requests in rapid succession
      const requestPromises = Array.from({length: numRequests}, () =>
        supertest(`http://localhost:${port}`).post('/browsers').send({
          userId,
          password: 'testpassword',
        }),
      );

      const responses = await Promise.allSettled(requestPromises);
      const fulfilledResponses = responses
        .filter((result) => result.status === 'fulfilled')
        .map((result) => (result as PromiseFulfilledResult<any>).value);

      // Count success and failure responses
      const successCount = fulfilledResponses.filter((r) => r.status === 200).length;
      const failureCount = fulfilledResponses.filter((r) => r.status === 400).length;

      // Exactly one should succeed, the rest should fail
      expect(successCount).toBe(1);
      expect(failureCount).toBe(numRequests - 1);

      // Verify the success response
      const successResponse = fulfilledResponses.find((r) => r.status === 200);
      expect(successResponse!.body.success).toBe(true);
      expect(successResponse!.body.message).toContain(userId);

      // Verify all failure responses have the correct error
      const failureResponses = fulfilledResponses.filter((r) => r.status === 400);
      failureResponses.forEach((response) => {
        expect(response.body.success).toBe(false);
        expect(response.body.error.code).toBe('CREATE_SESSION_FAILED');
        expect(response.body.error.message).toContain('already exists');
      });

      // Verify only one session exists in the system
      const listResponse = await supertest(`http://localhost:${port}`).get('/browsers').expect(200);

      expect(listResponse.body.data.count).toBe(1);
      expect(listResponse.body.data.sessions[0].userId).toBe(userId);
    });

    test('should create sessions for different userIds simultaneously', async () => {
      await app.listen({port});

      // Send requests for different users simultaneously
      const responses = await Promise.all([
        supertest(`http://localhost:${port}`).post('/browsers').send({
          userId: 'user1',
          password: 'password1',
        }),
        supertest(`http://localhost:${port}`).post('/browsers').send({
          userId: 'user2',
          password: 'password2',
        }),
        supertest(`http://localhost:${port}`).post('/browsers').send({
          userId: 'user3',
          password: 'password3',
        }),
      ]);

      // All should succeed
      responses.forEach((response, index) => {
        expect(response.status).toBe(200);
        expect(response.body.success).toBe(true);
        expect(response.body.message).toContain(`user${index + 1}`);
      });
    });
  });

  describe('GET /browsers', () => {
    test('should return empty list initially', async () => {
      await app.listen({port});

      const response = await supertest(`http://localhost:${port}`)
        .get('/browsers')
        .expect(200)
        .expect('Content-Type', /json/);

      expect(response.body).toEqual({
        success: true,
        data: {
          sessions: [],
          count: 0,
        },
      });
    });

    test('should return active sessions after creation', async () => {
      await app.listen({port});

      // Create a browser session
      await supertest(`http://localhost:${port}`)
        .post('/browsers')
        .send({
          userId: 'session-list-user',
          password: 'testpassword',
        })
        .expect(200);

      // Check that it appears in the list
      const response = await supertest(`http://localhost:${port}`).get('/browsers').expect(200);

      expect(response.body.success).toBe(true);
      expect(response.body.data.count).toBe(1);
      expect(response.body.data.sessions).toHaveLength(1);
      expect(response.body.data.sessions[0]).toMatchObject({
        userId: 'session-list-user',
        state: expect.any(String),
        createdAt: expect.any(String),
      });
    });

    test('should return multiple active sessions', async () => {
      await app.listen({port});

      // Create multiple browser sessions
      const userIds = ['multi-user-1', 'multi-user-2', 'multi-user-3'];

      for (const userId of userIds) {
        await supertest(`http://localhost:${port}`)
          .post('/browsers')
          .send({
            userId,
            password: 'testpassword',
          })
          .expect(200);
      }

      // Check that all appear in the list
      const response = await supertest(`http://localhost:${port}`).get('/browsers').expect(200);

      expect(response.body.success).toBe(true);
      expect(response.body.data.count).toBe(3);
      expect(response.body.data.sessions).toHaveLength(3);

      const returnedUserIds = response.body.data.sessions.map((s: any) => s.userId);
      userIds.forEach((userId) => {
        expect(returnedUserIds).toContain(userId);
      });
    });

    test('should handle sessions with different states', async () => {
      await app.listen({port});

      // Create multiple browser sessions that will have different states
      const userIds = ['state-user-1', 'state-user-2', 'state-user-3'];

      for (const userId of userIds) {
        await supertest(`http://localhost:${port}`)
          .post('/browsers')
          .send({
            userId,
            password: 'testpassword',
          })
          .expect(200);
      }

      // Check that all appear in the list with various states
      const response = await supertest(`http://localhost:${port}`).get('/browsers').expect(200);

      expect(response.body.success).toBe(true);
      expect(response.body.data.count).toBe(3);
      expect(response.body.data.sessions).toHaveLength(3);

      // Each session should have a valid state
      response.body.data.sessions.forEach((session: any) => {
        expect(userIds).toContain(session.userId);
        expect(typeof session.state).toBe('string');
        expect(typeof session.createdAt).toBe('string');
      });
    });
  });

  describe('DELETE /browsers', () => {
    test('should successfully remove an existing session', async () => {
      await app.listen({port});

      // First create a session
      await supertest(`http://localhost:${port}`)
        .post('/browsers')
        .send({
          userId: 'delete-test-user',
          password: 'testpassword',
        })
        .expect(200);

      // Verify it exists
      const listResponse = await supertest(`http://localhost:${port}`).get('/browsers').expect(200);

      expect(listResponse.body.data.count).toBe(1);

      // Remove the session
      const deleteResponse = await supertest(`http://localhost:${port}`)
        .delete('/browsers')
        .send({
          userId: 'delete-test-user',
        })
        .expect(200);

      expect(deleteResponse.body).toEqual({
        success: true,
        message: expect.stringContaining('scheduled for removal'),
      });

      // Verify the session is marked for removal/cleanup
      const finalListResponse = await supertest(`http://localhost:${port}`).get('/browsers').expect(200);

      // Session might still be in list but with 'stopping' state, or might be cleaned up
      if (finalListResponse.body.data.count > 0) {
        const session = finalListResponse.body.data.sessions.find((s: any) => s.userId === 'delete-test-user');
        if (session) {
          expect(['stopping', 'cleanup_failed']).toContain(session.state);
        }
      }
    });

    test('should return 400 when userId is missing', async () => {
      await app.listen({port});

      const response = await supertest(`http://localhost:${port}`)
        .delete('/browsers')
        .send({})
        .expect(400)
        .expect('Content-Type', /json/);

      expect(response.body).toEqual({
        success: false,
        error: {
          code: 'USER_ID_MISSING',
          message: 'userId is missing',
        },
      });
    });

    test('should return 400 when userId is empty string', async () => {
      await app.listen({port});

      const response = await supertest(`http://localhost:${port}`)
        .delete('/browsers')
        .send({
          userId: '',
        })
        .expect(400)
        .expect('Content-Type', /json/);

      expect(response.body).toEqual({
        success: false,
        error: {
          code: 'USER_ID_MISSING',
          message: 'userId is missing',
        },
      });
    });

    test('should fail to remove non-existent session', async () => {
      await app.listen({port});

      const response = await supertest(`http://localhost:${port}`)
        .delete('/browsers')
        .send({
          userId: 'non-existent-user',
        })
        .expect(400);

      expect(response.body).toEqual({
        success: false,
        error: {
          code: 'REMOVE_SESSION_FAILED',
          message: expect.stringContaining('does not exist'),
        },
      });
    });
  });

  test('should handle complete session lifecycle', async () => {
    await app.listen({port});

    const userId = 'lifecycle-test-user';

    // 1. Initially no sessions
    let response = await supertest(`http://localhost:${port}`).get('/browsers').expect(200);
    expect(response.body.data.count).toBe(0);

    // 2. Create session
    response = await supertest(`http://localhost:${port}`)
      .post('/browsers')
      .send({
        userId,
        password: 'testpassword',
      })
      .expect(200);
    expect(response.body.success).toBe(true);

    // 3. Verify session exists
    response = await supertest(`http://localhost:${port}`).get('/browsers').expect(200);
    expect(response.body.data.count).toBe(1);
    expect(response.body.data.sessions[0].userId).toBe(userId);

    // 4. Try to create duplicate (should fail)
    response = await supertest(`http://localhost:${port}`)
      .post('/browsers')
      .send({
        userId,
        password: 'testpassword',
      })
      .expect(400);
    expect(response.body.success).toBe(false);

    // 5. Remove session
    response = await supertest(`http://localhost:${port}`)
      .delete('/browsers')
      .send({
        userId,
      })
      .expect(200);
    expect(response.body.success).toBe(true);

    // 6. Session should be marked for cleanup or removed
    response = await supertest(`http://localhost:${port}`).get('/browsers').expect(200);

    if (response.body.data.count > 0) {
      const session = response.body.data.sessions.find((s: any) => s.userId === userId);
      if (session) {
        expect(['stopping', 'cleanup_failed']).toContain(session.state);
      }
    }
  });
});
