// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {describe, expect, test, beforeEach, vi} from 'vitest';

import {SessionState} from '@mattermost/loadtest-browser-lib';

import {TestManager, testManager} from './test_manager.js';

vi.mock('../app.js', () => {
  const mockLog = {
    info: vi.fn(),
    error: vi.fn(),
  };

  return {
    log: mockLog,
    __mockLog: mockLog,
  };
});

vi.mock('../simulations/post_and_scroll_scenario.js', () => ({
  postAndScrollScenario: vi.fn().mockImplementation(async () => {
    // Wait a bit before resolving to simulate test execution
    await new Promise((resolve) => setTimeout(resolve, 10));
    return undefined;
  }),
}));

import * as appModule from '../app.js';

const mockLog = (appModule as any).__mockLog;

describe('TestManager', () => {
  const mockBrowserInstance = {
    browser: null,
    context: null,
    page: null,
    userId: 'testUser',
    password: 'password',
    createdAt: new Date(),
    state: SessionState.CREATED,
  };

  const mockActiveBrowserSessions = new Map();

  beforeEach(() => {
    vi.clearAllMocks();
    mockActiveBrowserSessions.clear();
    mockLog.info.mockClear();
    mockLog.error.mockClear();
  });

  describe('TestManager Class', () => {
    test('should create a singleton instance', () => {
      const instance1 = TestManager.getInstance();
      const instance2 = TestManager.getInstance();
      expect(instance1).toBe(instance2);
      expect(instance1).toBe(testManager);
    });

    test('should get scenario function by id', () => {
      const scenario = testManager.getScenario('mattermostPostAndScroll' as any);
      expect(scenario).toBeDefined();
      expect(typeof scenario).toBe('function');
    });

    test('should throw error for invalid scenario id', () => {
      expect(() => testManager.getScenario('invalid' as any)).toThrow('Scenario invalid not found');
    });

    test('should complete test successfully', async () => {
      const updatedInstance = await testManager.startTest(
        mockBrowserInstance,
        mockActiveBrowserSessions,
        'http://localhost:8065',
        'mattermostPostAndScroll',
      );

      expect(updatedInstance).toBeDefined();
      expect(updatedInstance?.state).toBe(SessionState.COMPLETED);

      expect(mockLog.info).toHaveBeenCalledWith('simulation-starting--mattermostPostAndScroll--testUser');
      expect(mockLog.info).toHaveBeenCalledWith('simulation-completed--mattermostPostAndScroll--testUser');
      expect(mockLog.info).toHaveBeenCalledTimes(2);
      expect(mockLog.error).not.toHaveBeenCalled();
    });

    test('should handle test failure', async () => {
      // Mock scenario to throw error
      vi.mocked(await import('../simulations/post_and_scroll_scenario.js')).postAndScrollScenario.mockRejectedValueOnce(
        new Error('Test failed'),
      );

      const updatedInstance = await testManager.startTest(
        mockBrowserInstance,
        mockActiveBrowserSessions,
        'http://localhost:8065',
        'mattermostPostAndScroll',
      );

      expect(updatedInstance).toBeDefined();
      expect(updatedInstance?.state).toBe(SessionState.FAILED);

      expect(mockLog.info).toHaveBeenCalledWith('simulation-starting--mattermostPostAndScroll--testUser');
      expect(mockLog.error).toHaveBeenCalledWith(
        'simulation-failed--mattermostPostAndScroll--testUser--Error: Test failed',
      );
      expect(mockLog.info).toHaveBeenCalledTimes(1);
      expect(mockLog.error).toHaveBeenCalledTimes(1);
    });

    test('should handle test being stopped by user', async () => {
      const stoppingMockSessions = new Map();
      stoppingMockSessions.set('testUser', {
        ...mockBrowserInstance,
        state: SessionState.STOPPING,
      });

      // Mock scenario to throw error to simulate interruption
      vi.mocked(await import('../simulations/post_and_scroll_scenario.js')).postAndScrollScenario.mockRejectedValueOnce(
        new Error('Test interrupted'),
      );

      const updatedInstance = await testManager.startTest(
        mockBrowserInstance,
        stoppingMockSessions,
        'http://localhost:8065',
        'mattermostPostAndScroll',
      );

      expect(updatedInstance).toBeUndefined();

      expect(mockLog.info).toHaveBeenCalledWith('simulation-starting--mattermostPostAndScroll--testUser');
      expect(mockLog.info).toHaveBeenCalledWith('simulation-stopped--mattermostPostAndScroll--testUser');
      expect(mockLog.info).toHaveBeenCalledTimes(2);
      expect(mockLog.error).not.toHaveBeenCalled();
    });

    test('should handle test error with testId', async () => {
      // Mock scenario to throw TestError
      const testError = {
        error: new Error('Test step failed'),
        testId: 'login',
      };
      vi.mocked(await import('../simulations/post_and_scroll_scenario.js')).postAndScrollScenario.mockRejectedValueOnce(
        testError,
      );

      const updatedInstance = await testManager.startTest(
        mockBrowserInstance,
        mockActiveBrowserSessions,
        'http://localhost:8065',
        'mattermostPostAndScroll',
      );

      expect(updatedInstance).toBeDefined();
      expect(updatedInstance?.state).toBe(SessionState.FAILED);

      // Verify correct logging for test error with testId
      expect(mockLog.info).toHaveBeenCalledWith('simulation-starting--mattermostPostAndScroll--testUser');
      expect(mockLog.error).toHaveBeenCalledWith(
        'simulation-failed--mattermostPostAndScroll--testUser--login--Error: Test step failed',
      );
      expect(mockLog.info).toHaveBeenCalledTimes(1);
      expect(mockLog.error).toHaveBeenCalledTimes(1);
    });

    test('should handle test error without testId', async () => {
      // Mock scenario to throw error
      vi.mocked(await import('../simulations/post_and_scroll_scenario.js')).postAndScrollScenario.mockRejectedValueOnce(
        new Error('Test failed'),
      );

      const updatedInstance = await testManager.startTest(
        mockBrowserInstance,
        mockActiveBrowserSessions,
        'http://localhost:8065',
        'mattermostPostAndScroll',
      );

      expect(updatedInstance).toBeDefined();
      expect(updatedInstance?.state).toBe(SessionState.FAILED);

      expect(mockLog.info).toHaveBeenCalledWith('simulation-starting--mattermostPostAndScroll--testUser');
      expect(mockLog.error).toHaveBeenCalledWith(
        'simulation-failed--mattermostPostAndScroll--testUser--Error: Test failed',
      );
      expect(mockLog.info).toHaveBeenCalledTimes(1);
      expect(mockLog.error).toHaveBeenCalledTimes(1);
    });
  });
});
