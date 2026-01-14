// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {chromium, Page, devices} from 'playwright';
import {join} from 'path';

import {SessionState} from '@mattermost/loadtest-browser';
import {type BrowserInstance} from '@mattermost/loadtest-browser';

import {log} from '../app.js';
import {testManager} from '../lib/test_manager.js';
import {screenshotsDirectory} from '../utils/config_helpers.js';
import {getSimulationTimeoutMs} from '../utils/config_accessors.js';

const CLEANUP_TIMEOUT_MS = 4_000;

const browserArguments = [
  '--enable-automation', // Disables UI prompts that interfere with automation eg extension warning etc.

  // Common unwanted browser features
  '--disable-client-side-phishing-detection', // Disables client-side phishing detection
  '--disable-component-extensions-with-background-pages', // Disables loading of Chrome extensions with background pages
  '--disable-default-apps', // Disables installing/loading default Chrome apps such as Youtube etc.
  '--disable-extensions', // Disables loading of Chrome extensions
  '--disable-features=InterestFeedContentSuggestions', // Disables content suggestions
  '--disable-features=Translate', // Disables translation of web pages
  '--disable-search-engine-choice-screen', // Disables search engine choice screen
  '--no-first-run', // Skip Chromium's setup dialogs, wizard and welcome screen

  // Background network services
  '--disable-background-networking', // Disables background network services such as extension updates etc
  '--disable-sync', // Disables syncing of Chrome settings across devices
];

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
    simulationId: string,
    isHeadless: boolean,
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
        headless: isHeadless,
        args: browserArguments,
      });

      instance = {...instance, browser};
      this.activeBrowserSessions.set(userId, instance);
    } catch (error) {
      const creationFailedInstance: BrowserInstance = {
        ...instance,
        state: SessionState.CREATION_FAILED,
      };
      this.activeBrowserSessions.set(userId, creationFailedInstance);

      return {
        isCreated: false,
        message: `Failed to create browser instance for user ${userId}, error: ${error}`,
      };
    }

    // Try to create the context after the browser instance is created
    try {
      const context = await instance.browser!.newContext({
        viewport: {width: 1366, height: 768},
        isMobile: false,
        hasTouch: false,
        deviceScaleFactor: 1,
      });

      instance = {...instance, context};
      this.activeBrowserSessions.set(userId, instance);
    } catch (error) {
      const creationFailedInstance: BrowserInstance = {
        ...instance,
        state: SessionState.CREATION_FAILED,
      };
      this.activeBrowserSessions.set(userId, creationFailedInstance);

      return {
        isCreated: false,
        message: `Failed to create browser context for user ${userId}, error: ${error}`,
      };
    }

    // Try to create the page after the context is created
    try {
      const page = await instance.context!.newPage();
      page.setDefaultTimeout(getSimulationTimeoutMs());

      instance = {...instance, page};
      this.activeBrowserSessions.set(userId, instance);
    } catch (error) {
      const creationFailedInstance: BrowserInstance = {
        ...instance,
        state: SessionState.CREATION_FAILED,
      };
      this.activeBrowserSessions.set(userId, creationFailedInstance);

      return {
        isCreated: false,
        message: `Failed to create browser page for user ${userId}, error: ${error}`,
      };
    }

    instance = {
      ...instance,
      state: SessionState.CREATED,
    };
    this.activeBrowserSessions.set(userId, instance);

    this.startTestsInBrowserSession(userId, instance, serverURL, simulationId);

    return {
      isCreated: true,
      message: `Successfully created browser instance for user ${userId}`,
    };
  }

  public async removeBrowserSession(userId: string): Promise<{isRemoved: boolean; message: string}> {
    const browserInstance = this.activeBrowserSessions.get(userId);
    if (!browserInstance) {
      const message = `Browser instance does not exist for user ${userId}`;
      log.info(message);
      return {
        isRemoved: false,
        message,
      };
    }

    const toBeRemovedInstance = {...browserInstance, state: SessionState.STOPPING};
    this.activeBrowserSessions.set(userId, toBeRemovedInstance);

    const message = `Browser instance scheduled for removal for user ${userId}`;
    log.info(message);
    return {
      isRemoved: true,
      message,
    };
  }

  /**
   * Start tests asynchronously for a browser session
   * It informs if the test failed or was stopped.
   * Also cleans up the browser session regardless of test success or failure
   */
  private async startTestsInBrowserSession(
    userId: string,
    browserInstance: BrowserInstance,
    serverURL: string,
    simulationId: string,
  ) {
    const message = `Starting ${simulationId} simulation tests for user ${userId}`;
    log.info(message);

    const instance = {...browserInstance, state: SessionState.STARTED};
    this.activeBrowserSessions.set(userId, instance);

    const updatedBrowserInstance = await testManager.startTest(
      browserInstance,
      this.activeBrowserSessions,
      serverURL,
      simulationId,
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

  private async takeScreenshotOnClose(page: Page, userId: string): Promise<void> {
    try {
      const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
      const filename = `on_page_close_screenshot_${userId}_${timestamp}.png`;
      const filepath = join(screenshotsDirectory, filename);

      await page.screenshot({
        path: filepath,
        fullPage: false,
      });

      log.info(`Successfully took screenshot on page close for user ${userId}`);
    } catch (error) {
      log.error(`Failed to take screenshot on page close for user ${userId}: ${error}`);
    }
  }

  private async cleanupBrowserSessions() {
    if (this.activeBrowserSessions.size === 0) {
      return;
    }

    const promises: Promise<void>[] = [];
    for (const instance of this.activeBrowserSessions.values()) {
      if (
        instance.state === SessionState.CREATION_FAILED ||
        instance.state === SessionState.COMPLETED ||
        instance.state === SessionState.STOPPING ||
        instance.state === SessionState.FAILED
      ) {
        // Take a screenshot on page close and then cleanup the browser session
        // We use finally because we want to cleanup the browser session regardless of the screenshot operation success or failure
        promises.push(
          this.takeScreenshotOnClose(instance.page!, instance.userId).finally(() =>
            this.cleanupBrowserSession(instance),
          ),
        );
      }
    }

    // Wait for all cleanup operations to complete in parallel
    // we used allSettled because each cleanup operation is independent of the others
    await Promise.allSettled(promises);
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

    if (
      cleanupPromisesResults.length !== 0 &&
      cleanupPromisesResults.every((result) => result.status === 'fulfilled')
    ) {
      log.info('successfully cleaned up all browser sessions in browser_manager');
    }
  }
}

export const browserTestSessionManager = BrowserTestSessionManager.getInstance();
