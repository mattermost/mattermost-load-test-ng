// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {describe, expect, test, beforeEach, afterEach, vi} from 'vitest';

vi.mock('playwright', () => {
  const mockPageClose = vi.fn().mockResolvedValue(undefined);
  const mockPage = {close: mockPageClose};

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
    __mocks: {
      mockPageClose,
      mockContextNewPage,
      mockContextClose,
      mockBrowserNewContext,
      mockBrowserClose,
      mockChromiumLaunch,
      mockPage,
      mockContext,
      mockBrowser,
    },
  };
});

vi.mock('../simulations/post_and_scroll_scenario.js', () => ({
  postAndScrollScenario: vi.fn().mockImplementation(async () => {
    // Wait a bit before resolving to simulate test execution
    await new Promise((resolve) => setTimeout(resolve, 10));
    return undefined;
  }),
}));

import {BrowserTestSessionManager, browserTestSessionManager} from './browser_manager.js';
import * as playwright from 'playwright';
import * as postAndScrollScenario from '../simulations/post_and_scroll_scenario.js';
import {SimulationIds} from '../simulations/registry.js';

const mocks = (playwright as any).__mocks;
const postAndScrollScenarioMock = vi.mocked(postAndScrollScenario.postAndScrollScenario);

describe('BrowserManager', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.useFakeTimers();

    // Reset mocks
    mocks.mockChromiumLaunch.mockClear();
    mocks.mockBrowserNewContext.mockClear();
    mocks.mockContextNewPage.mockClear();
    mocks.mockPageClose.mockClear();
    mocks.mockContextClose.mockClear();
    mocks.mockBrowserClose.mockClear();
    postAndScrollScenarioMock.mockClear();

    // Reset default return values
    mocks.mockChromiumLaunch.mockResolvedValue(mocks.mockBrowser);
    mocks.mockBrowserNewContext.mockResolvedValue(mocks.mockContext);
    mocks.mockContextNewPage.mockResolvedValue(mocks.mockPage);
  });

  afterEach(async () => {
    await browserTestSessionManager.shutdown();
    vi.useRealTimers();
  });

  test('should create a instance of BrowserTestSessionManager', () => {
    const browserManager = BrowserTestSessionManager.getInstance();
    expect(browserManager).toBeDefined();
  });

  test('should maintain the same instance of BrowserTestSessionManager', () => {
    const instance1 = BrowserTestSessionManager.getInstance();
    const instance2 = BrowserTestSessionManager.getInstance();
    expect(instance1).toBe(instance2);
  });

  test('getActiveBrowserSessions should return empty array when no user"s sessions exist', () => {
    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    expect(sessions).toEqual([]);
  });

  test('createBrowserSession should successfully create a user"s browser session', async () => {
    const {isCreated, message} = await browserTestSessionManager.createBrowserSession(
      'user1',
      'password',
      'http://localhost:8065',
      SimulationIds.postAndScroll,
      true,
    );

    expect(isCreated).toBe(true);
    expect(message).toContain('Successfully created browser instance for user user1');

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    expect(sessions.length).toBe(1);
    expect(sessions[0].userId).toBe('user1');
    expect(sessions[0].state).toBe('started');
  });

  test('createBrowserSession should fail when user"s session already exists', async () => {
    await browserTestSessionManager.createBrowserSession(
      'user1',
      'password',
      'http://localhost:8065',
      SimulationIds.postAndScroll,
      true,
    );

    // Try to create another session with same userId
    const {isCreated, message} = await browserTestSessionManager.createBrowserSession(
      'user1',
      'password',
      'http://localhost:8065',
      SimulationIds.postAndScroll,
      true,
    );

    expect(isCreated).toBe(false);
    expect(message).toContain('Browser instance already exists for user');
  });

  test('removeBrowserSession should fail when user"s session does not exist', async () => {
    const {isRemoved, message} = await browserTestSessionManager.removeBrowserSession('nonexistent');

    expect(isRemoved).toBe(false);
    expect(message).toContain('does not exist');
  });

  test('removeBrowserSession should mark user"s session for removal', async () => {
    await browserTestSessionManager.createBrowserSession(
      'user1',
      'password',
      'http://localhost:8065',
      SimulationIds.postAndScroll,
      true,
    );

    const {isRemoved, message} = await browserTestSessionManager.removeBrowserSession('user1');

    expect(isRemoved).toBe(true);
    expect(message).toContain('Browser instance scheduled for removal for user');

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    expect(sessions.length).toBe(1);
    expect(sessions[0].userId).toBe('user1');
    expect(sessions[0].state).toBe('stopping');
  });

  test('should mark user"s session as creation_failed when browser creation fails', async () => {
    // Mock browser launch to fail
    mocks.mockChromiumLaunch.mockRejectedValueOnce(new Error('Launch failed'));

    const {isCreated, message} = await browserTestSessionManager.createBrowserSession(
      'failuser',
      'password',
      'http://localhost:8065',
      SimulationIds.postAndScroll,
      true,
    );

    expect(isCreated).toBe(false);
    expect(message).toContain('Failed to create browser instance');

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    const failedSession = sessions.find((s) => s.userId === 'failuser');
    expect(failedSession).toBeDefined();
    expect(failedSession?.state).toBe('creation_failed');
  });

  test('should mark user"s session as creation_failed when context creation fails', async () => {
    // Set up browser success but context failure
    mocks.mockBrowserNewContext.mockRejectedValueOnce(new Error('Context creation failed'));

    const {isCreated, message} = await browserTestSessionManager.createBrowserSession(
      'contextfailuser',
      'password',
      'http://localhost:8065',
      SimulationIds.postAndScroll,
      true,
    );

    expect(isCreated).toBe(false);
    expect(message).toContain('Failed to create browser context for user');

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    const failedSession = sessions.find((s) => s.userId === 'contextfailuser');
    expect(failedSession).toBeDefined();
    expect(failedSession?.state).toBe('creation_failed');
  });

  test('should mark user"s session as creation_failed when page creation fails', async () => {
    // Set up browser & context success but page failure
    mocks.mockContextNewPage.mockRejectedValueOnce(new Error('Page creation failed'));

    const {isCreated, message} = await browserTestSessionManager.createBrowserSession(
      'pagefailuser',
      'password',
      'http://localhost:8065',
      SimulationIds.postAndScroll,
      true,
    );

    expect(isCreated).toBe(false);
    expect(message).toContain('Failed to create page');

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    const failedSession = sessions.find((s) => s.userId === 'pagefailuser');
    expect(failedSession).toBeDefined();
    expect(failedSession?.state).toBe('creation_failed');
  });

  test('should mark user"s session as failed when tests fail', async () => {
    // Set up test scenario to fail
    postAndScrollScenarioMock.mockRejectedValueOnce(new Error('Test scenario failed'));

    await browserTestSessionManager.createBrowserSession(
      'testfailuser',
      'password',
      'http://localhost:8065',
      SimulationIds.postAndScroll,
      true,
    );

    await vi.runAllTimersAsync();

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    const failedSession = sessions.find((s) => s.userId === 'testfailuser');
    expect(failedSession).toBeDefined();
    expect(failedSession?.state).toBe('failed');
  });

  test('should mark user"s session as completed when tests complete successfully', async () => {
    await browserTestSessionManager.createBrowserSession(
      'user1',
      'password',
      'http://localhost:8065',
      SimulationIds.postAndScroll,
      true,
    );

    await vi.runAllTimersAsync();

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    const completedSession = sessions.find((s) => s.userId === 'user1');
    expect(completedSession).toBeDefined();
    expect(completedSession?.state).toBe('completed');
  });

  test('shutdown should clean up all sessions', async () => {
    await browserTestSessionManager.createBrowserSession(
      'user1',
      'password',
      'http://localhost:8065',
      SimulationIds.postAndScroll,
      true,
    );
    await browserTestSessionManager.createBrowserSession(
      'user2',
      'password',
      'http://localhost:8065',
      SimulationIds.postAndScroll,
      true,
    );

    await browserTestSessionManager.shutdown();

    expect(mocks.mockBrowserClose).toHaveBeenCalled();

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    expect(sessions).toEqual([]);
  });

  test('should mark user"s session as cleanup_failed when cleanup fails', async () => {
    await browserTestSessionManager.createBrowserSession(
      'user1',
      'password',
      'http://localhost:8065',
      SimulationIds.postAndScroll,
      true,
    );

    await vi.runAllTimersAsync();

    // Mock browser close to fail
    mocks.mockBrowserClose.mockRejectedValueOnce(new Error('Cleanup failed'));

    await browserTestSessionManager.shutdown();

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    const failedSession = sessions.find((s) => s.userId === 'user1');
    expect(failedSession).toBeDefined();
    expect(failedSession?.state).toBe('cleanup_failed');
  });
});
