import {Browser, BrowserContext, chromium, Page} from 'playwright';

import * as tests from '../tests/scenario.js';

const CLEANUP_TIMEOUT = 4 * 1000; // 2 seconds

enum SessionState {
  CREATING = 'creating', // The browser and other instances are being created
  CREATION_FAILED = 'creation_failed', // The browser or any other instances failed to be created
  CREATED = 'created', // The browser and other instances were created successfully
  STARTED = 'started', // The test was started
  STOPPING = 'stopping', // The test was stopped by the user
  COMPLETED = 'completed', // The test was completed successfully
  FAILED = 'failed', // The test failed at any point
  CLEANUP_FAILED = 'cleanup_failed', // The browser or any other instances failed to be cleaned up
}

export type BrowserInstance = {
  browser: Browser | null;
  context: BrowserContext | null;
  page: Page | null;
  userId: string;
  password: string;
  createdAt: Date;
  state: SessionState;
};

type BrowserInstanceAsResponse = Pick<BrowserInstance, 'userId' | 'createdAt'>;

class BrowserTestSessionManager {
  private static instance: BrowserTestSessionManager;

  private activeBrowserSessions: Map<string, BrowserInstance> = new Map();
  private cleanupBrowserSessionTimeout: NodeJS.Timeout | null = null;

  private constructor() {
    this.activeBrowserSessions = new Map();

    this.startPeriodicBrowserSessionCleanup(CLEANUP_TIMEOUT);
  }

  public static getInstance() {
    if (!BrowserTestSessionManager.instance) {
      BrowserTestSessionManager.instance = new BrowserTestSessionManager();
    }
    return BrowserTestSessionManager.instance;
  }

  /**
   * We don't want to return the whole browser instance, because it's too big and we don't need it in most cases
   * Instead, we return a subset of the data of the browser instances
   */
  public getActiveBrowserSessions(): BrowserInstanceAsResponse[] {
    if (this.activeBrowserSessions.size === 0) {
      return [];
    }

    const instancesAsResponse: BrowserInstanceAsResponse[] = [];

    for (const [userId, value] of this.activeBrowserSessions.entries()) {
      instancesAsResponse.push({
        userId,
        createdAt: value.createdAt,
      });
    }

    return instancesAsResponse;
  }

  public async createBrowserSession(userId: string, password: string): Promise<{isCreated: boolean; message: string}> {
    if (this.activeBrowserSessions.has(userId)) {
      return {
        isCreated: false,
        message: `Browser instance already exists for user ${userId}`,
      };
    }

    let instance: BrowserInstance = {
      browser: null,
      context: null,
      page: null,
      userId,
      password,
      createdAt: new Date(),
      state: SessionState.CREATING,
    };
    this.activeBrowserSessions.set(userId, instance);

    // Try to create the browser instance first
    try {
      const browser = await chromium.launch({
        headless: false,
      });

      instance = {...instance, browser};
      this.activeBrowserSessions.set(userId, instance);
    } catch (error) {
      const creationFailedInstance: BrowserInstance = {
        ...instance,
        state: SessionState.CREATION_FAILED,
      };
      this.activeBrowserSessions.set(userId, creationFailedInstance);

      console.error(`[browser_manager] Failed to create browser instance for "${userId}"`, error);
      return {
        isCreated: false,
        message: `Failed to create browser instance for user "${userId}"`,
      };
    }

    // Try to create the context second
    try {
      const context = await instance.browser!.newContext();

      instance = {...instance, context};
      this.activeBrowserSessions.set(userId, instance);
    } catch (error) {
      const creationFailedInstance: BrowserInstance = {
        ...instance,
        state: SessionState.CREATION_FAILED,
      };
      this.activeBrowserSessions.set(userId, creationFailedInstance);

      console.error(`[browser_manager] Failed to create context for "${userId}"`, error);
      return {
        isCreated: false,
        message: `Failed to create context for user "${userId}"`,
      };
    }

    // Try to create the page third
    try {
      const page = await instance.context!.newPage();

      instance = {...instance, page};
      this.activeBrowserSessions.set(userId, instance);
    } catch (error) {
      const creationFailedInstance: BrowserInstance = {
        ...instance,
        state: SessionState.CREATION_FAILED,
      };
      this.activeBrowserSessions.set(userId, creationFailedInstance);

      console.error(`[browser_manager] Failed to create page for "${userId}"`, error);
      return {
        isCreated: false,
        message: `Failed to create page for user "${userId}"`,
      };
    }

    instance = {
      ...instance,
      state: SessionState.CREATED,
    };
    this.activeBrowserSessions.set(userId, instance);

    this.startTestsInBrowserSession(userId, instance);

    return {
      isCreated: true,
      message: `Browser instance created for user ${userId}`,
    };
  }

  public async removeBrowserSession(userId: string): Promise<{isRemoved: boolean; message: string}> {
    const browserInstance = this.activeBrowserSessions.get(userId);
    if (!browserInstance) {
      return {
        isRemoved: false,
        message: `Browser instance does not exist for user ${userId}`,
      };
    }

    const toBeRemovedInstance = {...browserInstance, state: SessionState.STOPPING};
    this.activeBrowserSessions.set(userId, toBeRemovedInstance);

    return {
      isRemoved: true,
      message: `Browser instance scheduled for removal for user ${userId}`,
    };
  }

  /**
   * Start tests asynchronously for a browser session
   * It infroms if the test failed or was stopped.
   * Also cleans up the browser session regardless of test success or failure
   */
  private async startTestsInBrowserSession(userId: string, browserInstance: BrowserInstance) {
    const instance = {...browserInstance, state: SessionState.STARTED};
    this.activeBrowserSessions.set(userId, instance);

    try {
      await tests.scenario1(browserInstance);

      const stoppedInstance: BrowserInstance = {
        ...browserInstance,
        state: SessionState.COMPLETED,
      };
      this.activeBrowserSessions.set(userId, stoppedInstance);

      console.log(`[browser_manager] Test completed for user "${userId}"`);
    } catch (error) {
      // This is a race condition, where as soon as we force stop the test, page instance is already closed
      // and tests still runs for a bit but fails due to page being closed and its going to be catched by this block
      // so we check if instance was purposely stopped by the user here
      if (this.activeBrowserSessions.get(userId)?.state === SessionState.STOPPING) {
        console.log(`[browser_manager] Test stopped for user "${userId}"`);
      } else {
        const failedInstance: BrowserInstance = {
          ...browserInstance,
          state: SessionState.FAILED,
        };
        this.activeBrowserSessions.set(userId, failedInstance);

        console.error(`[browser_manager] Failed test for user "${userId}"`, error);
      }
    }
  }

  /**
   * Start a periodic cleanup of browser sessions
   * But only starts the next cleanup after the current one is finished
   */
  private startPeriodicBrowserSessionCleanup(timeout: number) {
    this.cleanupBrowserSessionTimeout = setTimeout(async () => {
      await this.cleanupBrowserSessions();

      this.startPeriodicBrowserSessionCleanup(timeout);
    }, timeout);
  }

  private async cleanupBrowserSessions() {
    if (this.activeBrowserSessions.size === 0) {
      return;
    }

    for (const instance of this.activeBrowserSessions.values()) {
      if (
        instance.state === SessionState.CREATION_FAILED ||
        instance.state === SessionState.COMPLETED ||
        instance.state === SessionState.STOPPING ||
        instance.state === SessionState.FAILED ||
        instance.state === SessionState.CLEANUP_FAILED
      ) {
        await this.cleanupBrowserSession(instance);
      }
    }
  }

  private async cleanupBrowserSession(browserInstance: BrowserInstance): Promise<boolean> {
    try {
      if (browserInstance.page) {
        await browserInstance.page.close();
      }

      if (browserInstance.context) {
        await browserInstance.context.close();
      }

      if (browserInstance.browser) {
        await browserInstance.browser.close();
      }

      this.activeBrowserSessions.delete(browserInstance.userId);

      return true;
    } catch (error) {
      // the browser session was not cleaned up successfully
      // then we need to mark the browser instance as cleanup failed so we can retry cleanup later
      const cleanupFailedInstance: BrowserInstance = {
        ...browserInstance,
        state: SessionState.CLEANUP_FAILED,
      };
      this.activeBrowserSessions.set(browserInstance.userId, cleanupFailedInstance);

      return false;
    }
  }

  public async shutdown() {
    // Clear the cleanup timeout
    if (this.cleanupBrowserSessionTimeout) {
      clearTimeout(this.cleanupBrowserSessionTimeout);
      this.cleanupBrowserSessionTimeout = null;
    }

    // Clean up all active sessions
    const cleanupPromises: Promise<boolean>[] = [];
    for (const instance of this.activeBrowserSessions.values()) {
      cleanupPromises.push(this.cleanupBrowserSession(instance));
    }

    const cleanupPromisesResults = await Promise.allSettled(cleanupPromises);

    if (cleanupPromisesResults.length === 0) {
      console.log('[browser_manager] No active browser sessions to clean up');
    } else if (cleanupPromisesResults.every((result) => result.status === 'fulfilled')) {
      console.log('[browser_manager] Successfully cleaned up all browser sessions');
    } else {
      console.error('[browser_manager] Failed to clean up some browser sessions');
    }
  }
}

export const browserTestSessionManager = BrowserTestSessionManager.getInstance();
