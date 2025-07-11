// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Browser, BrowserContext, chromium, Page} from 'playwright';

import {log} from '../app.js';
import {testManager} from '../lib/test_manager.js';
import {isBrowserHeadless} from '../utils/config.js';

const CLEANUP_TIMEOUT_MS = 4_000;

export enum SessionState {
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

export type ActiveBrowserSessions = Map<string, BrowserInstance>;

type BrowserInstanceAsResponse = Pick<BrowserInstance, 'userId' | 'createdAt' | 'state'>;

export class BrowserTestSessionManager {
  private static instance: BrowserTestSessionManager;

  private activeBrowserSessions: ActiveBrowserSessions = new Map();
  private cleanupBrowserSessionTimeout: NodeJS.Timeout | null = null;

  private constructor() {
    this.activeBrowserSessions = new Map();

    this.startPeriodicBrowserSessionCleanup(CLEANUP_TIMEOUT_MS);
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
        state: value.state,
      });
    }

    return instancesAsResponse;
  }

  public async createBrowserSession(
    userId: string,
    password: string,
    serverURL: string,
  ): Promise<{isCreated: boolean; message: string}> {
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
        headless: isBrowserHeadless(),
      });

      instance = {...instance, browser};
      this.activeBrowserSessions.set(userId, instance);
    } catch (error) {
      const creationFailedInstance: BrowserInstance = {
        ...instance,
        state: SessionState.CREATION_FAILED,
      };
      this.activeBrowserSessions.set(userId, creationFailedInstance);

      log.error(`failed to create browser instance for "${userId}" in browser_manager ${error}`);
      return {
        isCreated: false,
        message: `Failed to create browser instance for user "${userId}"`,
      };
    }

    // Try to create the context after the browser instance is created
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

      log.error(`failed to create context for "${userId}" in browser_manager ${error}`);
      return {
        isCreated: false,
        message: `Failed to create context for user "${userId}"`,
      };
    }

    // Try to create the page after the context is created
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

      log.error(`failed to create page for "${userId}" in browser_manager ${error}`);
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

    this.startTestsInBrowserSession(userId, instance, serverURL);

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
   * It informs if the test failed or was stopped.
   * Also cleans up the browser session regardless of test success or failure
   */
  private async startTestsInBrowserSession(userId: string, browserInstance: BrowserInstance, serverURL: string) {
    const instance = {...browserInstance, state: SessionState.STARTED};
    this.activeBrowserSessions.set(userId, instance);

    const scenarioId = 'noop';
    const updatedBrowserInstance = await testManager.startTest(
      browserInstance,
      this.activeBrowserSessions,
      serverURL,
      scenarioId,
    );

    if (updatedBrowserInstance) {
      this.activeBrowserSessions.set(userId, updatedBrowserInstance);
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
        instance.state === SessionState.FAILED
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
      log.info('no active browser sessions to clean up in browser_manager');
    } else if (cleanupPromisesResults.every((result) => result.status === 'fulfilled')) {
      log.info('successfully cleaned up all browser sessions in browser_manager');
    } else {
      log.error('failed to clean up some browser sessions in browser_manager');
    }
  }
}

export const browserTestSessionManager = BrowserTestSessionManager.getInstance();
