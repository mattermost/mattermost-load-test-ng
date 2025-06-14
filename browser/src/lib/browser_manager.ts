import {Browser, BrowserContext, chromium, Page} from 'playwright';

import * as tests from '../tests/scenario.js';

export type BrowserInstance = {
  browser: Browser;
  context: BrowserContext;
  page: Page;
  userId: string;
  password: string;
  createdAt: Date;
};

type BrowserInstanceAsResponse = Pick<BrowserInstance, 'userId' | 'createdAt'>;

class BrowserTestSessionManager {
  private static instance: BrowserTestSessionManager;

  private activeBrowserSessions: Map<string, BrowserInstance> = new Map();
  private activeUsersInTests: Set<string> = new Set();

  private constructor() {
    this.activeBrowserSessions = new Map();
    this.activeUsersInTests = new Set();
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

    const browser = await chromium.launch({
      headless: false,
    });

    const context = await browser.newContext();
    const page = await context.newPage();
    const instance = {
      browser,
      context,
      page,
      userId,
      password,
      createdAt: new Date(),
    };

    this.activeBrowserSessions.set(userId, instance);

    this.startTestsInBrowserSession(instance);

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

    try {
      await this.cleanupBrowserSession(browserInstance);
    } catch (error) {
      console.error(`Either page, context or browser failed to close for user ${userId}:`, error);

      return {
        isRemoved: false,
        message: `Failed to terminate browser session for user ${userId}`,
      };
    }

    console.log(`[browser_manager] Browser instance removed for user ${userId}`);
    return {
      isRemoved: true,
      message: `Browser instance removed for user ${userId}`,
    };
  }

  private async cleanupBrowserSession(browserInstance: BrowserInstance) {
    try {
      this.activeUsersInTests.delete(browserInstance.userId);

      await browserInstance.page.close();
      await browserInstance.context.close();
      await browserInstance.browser.close();

      this.activeBrowserSessions.delete(browserInstance.userId);
    } catch (error) {
      throw error;
    }
  }

  /**
   * Start tests asynchronously for a browser session
   * It infroms if the test failed or was stopped.
   * Also cleans up the browser session regardless of test success or failure
   */
  private async startTestsInBrowserSession(browserInstance: BrowserInstance) {
    const userId = browserInstance.userId;

    // Mark user as running the tests in the browser session
    this.activeUsersInTests.add(userId);

    try {
      await tests.scenario1(browserInstance);
    } catch (error) {
      // Check if this is an expected error due to test being stopped
      // We can know this because if test failed and we never marked the user running test as stopped,
      if (this.activeUsersInTests.has(userId)) {
        console.error(`Failed test for user "${userId}"`, error);
      } else {
        console.log(`Stopped test for user ${userId}`);
      }
    } finally {
      await this.cleanupBrowserSession(browserInstance);
    }
  }
}

export const browserTestSessionManager = BrowserTestSessionManager.getInstance();
