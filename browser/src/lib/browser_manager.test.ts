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

  // Store mocks on a global object so tests can access them
  vi.stubGlobal('playwrightMocks', {
    mockPageClose,
    mockContextNewPage,
    mockContextClose,
    mockBrowserNewContext,
    mockBrowserClose,
    mockChromiumLaunch,
    mockPage,
    mockContext,
    mockBrowser,
  });

  return {
    chromium: {
      launch: mockChromiumLaunch,
    },
  };
});

// We mock the test scenario to simulate a test running in the browser
vi.mock('../tests/scenario1.js', () => {
  const scenario1 = vi.fn().mockImplementation(async () => {
    // Wait a bit before resolving to simulate test execution
    await new Promise((resolve) => setTimeout(resolve, 10));
    return undefined;
  });

  vi.stubGlobal('testScenarioMocks', {
    scenario1,
  });

  return {
    scenario1,
  };
});

// Import after mocks are set up
import {BrowserTestSessionManager, browserTestSessionManager} from './browser_manager.js';

// Type for our mocks to be used in the tests
declare global {
  var playwrightMocks: {
    mockPageClose: ReturnType<typeof vi.fn>;
    mockContextNewPage: ReturnType<typeof vi.fn>;
    mockContextClose: ReturnType<typeof vi.fn>;
    mockBrowserNewContext: ReturnType<typeof vi.fn>;
    mockBrowserClose: ReturnType<typeof vi.fn>;
    mockChromiumLaunch: ReturnType<typeof vi.fn>;
    mockPage: any;
    mockContext: any;
    mockBrowser: any;
  };

  var testScenarioMocks: {
    scenario1: ReturnType<typeof vi.fn>;
  };
}

describe('BrowserManager', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.useFakeTimers();

    // Reset mocks
    globalThis.playwrightMocks.mockChromiumLaunch.mockClear();
    globalThis.playwrightMocks.mockBrowserNewContext.mockClear();
    globalThis.playwrightMocks.mockContextNewPage.mockClear();
    globalThis.playwrightMocks.mockPageClose.mockClear();
    globalThis.playwrightMocks.mockContextClose.mockClear();
    globalThis.playwrightMocks.mockBrowserClose.mockClear();
    globalThis.testScenarioMocks.scenario1.mockClear();

    // Reset default return values
    globalThis.playwrightMocks.mockChromiumLaunch.mockResolvedValue(globalThis.playwrightMocks.mockBrowser);
    globalThis.playwrightMocks.mockBrowserNewContext.mockResolvedValue(globalThis.playwrightMocks.mockContext);
    globalThis.playwrightMocks.mockContextNewPage.mockResolvedValue(globalThis.playwrightMocks.mockPage);
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
    const {isCreated, message} = await browserTestSessionManager.createBrowserSession('user1', 'password');

    expect(isCreated).toBe(true);
    expect(message).toContain('Browser instance created for user user1');

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    expect(sessions.length).toBe(1);
    expect(sessions[0].userId).toBe('user1');
    expect(sessions[0].state).toBe('started');
  });

  test('createBrowserSession should fail when user"s session already exists', async () => {
    await browserTestSessionManager.createBrowserSession('user1', 'password');

    // Try to create another session with same userId
    const {isCreated, message} = await browserTestSessionManager.createBrowserSession('user1', 'password');

    expect(isCreated).toBe(false);
    expect(message).toContain('Browser instance already exists for user');
  });

  test('removeBrowserSession should fail when user"s session does not exist', async () => {
    const {isRemoved, message} = await browserTestSessionManager.removeBrowserSession('nonexistent');

    expect(isRemoved).toBe(false);
    expect(message).toContain('does not exist');
  });

  test('removeBrowserSession should mark user"s session for removal', async () => {
    await browserTestSessionManager.createBrowserSession('user1', 'password');

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
    globalThis.playwrightMocks.mockChromiumLaunch.mockRejectedValueOnce(new Error('Launch failed'));

    const {isCreated, message} = await browserTestSessionManager.createBrowserSession('failuser', 'password');

    expect(isCreated).toBe(false);
    expect(message).toContain('Failed to create browser instance');

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    const failedSession = sessions.find((s) => s.userId === 'failuser');
    expect(failedSession).toBeDefined();
    expect(failedSession?.state).toBe('creation_failed');
  });

  test('should mark user"s session as creation_failed when context creation fails', async () => {
    // Set up browser success but context failure
    globalThis.playwrightMocks.mockBrowserNewContext.mockRejectedValueOnce(new Error('Context creation failed'));

    const {isCreated, message} = await browserTestSessionManager.createBrowserSession('contextfailuser', 'password');

    expect(isCreated).toBe(false);
    expect(message).toContain('Failed to create context');

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    const failedSession = sessions.find((s) => s.userId === 'contextfailuser');
    expect(failedSession).toBeDefined();
    expect(failedSession?.state).toBe('creation_failed');
  });

  test('should mark user"s session as creation_failed when page creation fails', async () => {
    // Set up browser & context success but page failure
    globalThis.playwrightMocks.mockContextNewPage.mockRejectedValueOnce(new Error('Page creation failed'));

    const {isCreated, message} = await browserTestSessionManager.createBrowserSession('pagefailuser', 'password');

    expect(isCreated).toBe(false);
    expect(message).toContain('Failed to create page');

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    const failedSession = sessions.find((s) => s.userId === 'pagefailuser');
    expect(failedSession).toBeDefined();
    expect(failedSession?.state).toBe('creation_failed');
  });

  test('should mark user"s session as failed when tests fail', async () => {
    // Set up test scenario to fail
    globalThis.testScenarioMocks.scenario1.mockRejectedValueOnce(new Error('Test scenario failed'));

    await browserTestSessionManager.createBrowserSession('testfailuser', 'password');

    await vi.runAllTimersAsync();

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    const failedSession = sessions.find((s) => s.userId === 'testfailuser');
    expect(failedSession).toBeDefined();
    expect(failedSession?.state).toBe('failed');
  });

  test('should mark user"s session as completed when tests complete successfully', async () => {
    await browserTestSessionManager.createBrowserSession('user1', 'password');

    await vi.runAllTimersAsync();

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    const completedSession = sessions.find((s) => s.userId === 'user1');
    expect(completedSession).toBeDefined();
    expect(completedSession?.state).toBe('completed');
  });

  test('shutdown should clean up all sessions', async () => {
    await browserTestSessionManager.createBrowserSession('user1', 'password');
    await browserTestSessionManager.createBrowserSession('user2', 'password');

    await browserTestSessionManager.shutdown();

    expect(globalThis.playwrightMocks.mockBrowserClose).toHaveBeenCalled();

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    expect(sessions).toEqual([]);
  });

  test('should mark user"s session as cleanup_failed when cleanup fails', async () => {
    await browserTestSessionManager.createBrowserSession('user1', 'password');

    await vi.runAllTimersAsync();

    // Mock browser close to fail
    globalThis.playwrightMocks.mockBrowserClose.mockRejectedValueOnce(new Error('Cleanup failed'));

    await browserTestSessionManager.shutdown();

    const sessions = browserTestSessionManager.getActiveBrowserSessions();
    const failedSession = sessions.find((s) => s.userId === 'user1');
    expect(failedSession).toBeDefined();
    expect(failedSession?.state).toBe('cleanup_failed');
  });
});
