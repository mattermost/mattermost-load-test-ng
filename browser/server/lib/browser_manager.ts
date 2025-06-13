import {Browser, BrowserContext, chromium, Page} from 'playwright';

type BrowserInstance = {
  browser: Browser;
  context: BrowserContext;
  page: Page;
  userId: string;
  password: string;
  createdAt: Date;
};

type BrowserInstanceAsResponse = Pick<BrowserInstance, 'userId' | 'createdAt'>;

class BrowserManager {
  private browserInstances: Map<string, BrowserInstance> = new Map();
  private static instance: BrowserManager;

  private constructor() {
    this.browserInstances = new Map();
  }

  public static getInstance() {
    if (!BrowserManager.instance) {
      BrowserManager.instance = new BrowserManager();
    }
    return BrowserManager.instance;
  }

  public async createInstance(userId: string, password: string): Promise<{isCreated: boolean; message: string}> {
    if (this.browserInstances.has(userId)) {
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

    this.browserInstances.set(userId, instance);

    // TODO: start playwright test

    return {
      isCreated: true,
      message: `Browser instance created for user ${userId}`,
    };
  }

  public async removeInstance(userId: string): Promise<{isRemoved: boolean; message: string}> {
    if (!this.browserInstances.has(userId)) {
      return {
        isRemoved: false,
        message: `Browser instance does not exist for user ${userId}`,
      };
    }

    return {
      isRemoved: true,
      message: `Browser instance removed for user ${userId}`,
    };
  }

  public getInstancesCount(): number {
    return this.browserInstances.size;
  }

  public getUserKeys(): string[] {
    return Array.from(this.browserInstances.keys());
  }

  /**
   * We don't want to return the whole browser instance, because it's too big and we don't need it in most cases
   * Instead, we return a subset of the data of the browser instances
   */
  public getAllInstances(): BrowserInstanceAsResponse[] {
    if (this.getInstancesCount() === 0) {
      return [];
    }

    const instancesAsResponse: BrowserInstanceAsResponse[] = [];

    for (const [userId, value] of this.browserInstances.entries()) {
      instancesAsResponse.push({
        userId,
        createdAt: value.createdAt,
      });
    }

    return instancesAsResponse;
  }
}

export const browserManager = BrowserManager.getInstance();
