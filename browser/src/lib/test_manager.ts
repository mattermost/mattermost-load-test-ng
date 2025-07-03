// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {log} from '../app.js';
import * as tests from '../simulations/scenario1.js';
import type {BrowserInstance, ActiveBrowserSessions} from '../lib/browser_manager.js';
import {SessionState} from '../lib/browser_manager.js';

interface TestError {
  error: Error;
  testId: string;
}

type ScenarioMap = (browserInstance: BrowserInstance, serverURL: string) => Promise<void>;

export class TestManager {
  private static instance: TestManager;
  private scenarios: Map<string, ScenarioMap> = new Map();

  private constructor() {
    this.initScenarios();
  }

  public static getInstance(): TestManager {
    if (!TestManager.instance) {
      TestManager.instance = new TestManager();
    }
    return TestManager.instance;
  }

  private initScenarios(): void {
    this.scenarios.set('scenario1', tests.scenario1);
  }

  public async startTest(
    browserInstance: BrowserInstance,
    activeBrowserSessions: ActiveBrowserSessions,
    serverURL: string,
    userId: string,
    scenarioId: string,
  ): Promise<BrowserInstance | undefined> {
    let updatedBrowserInstance: BrowserInstance | undefined = {...browserInstance};

    try {
      log.info(`[simulation][start][${scenarioId}][${userId}]`);

      const scenario = this.getScenario(scenarioId);
      await scenario(browserInstance, serverURL);

      updatedBrowserInstance.state = SessionState.COMPLETED;
      log.info(`[simulation][completed][${scenarioId}][${userId}]`);
    } catch (error: unknown) {
      // This is a race condition, where as soon as we force stop the test, page instance is already closed
      // and tests still runs for a bit but fails due to page being closed and its going to be catched by this block
      // so we check if instance was forced to stop by the user here
      if (activeBrowserSessions.get(userId)?.state === SessionState.STOPPING) {
        log.info(`[simulation][stopped][${scenarioId}][${userId}]`);

        // We don't want to keep the browser instance in the active sessions map if it was stopped by the user
        // as it will be cleaned up by the browser manager eventually, so we dont need to update the state again
        updatedBrowserInstance = undefined;
      } else {
        updatedBrowserInstance.state = SessionState.FAILED;

        if (this.isTestFailureError(error)) {
          log.error(`[simulation][failed][${scenarioId}][${userId}][${error.testId}][${error.error.message}]`);
        } else {
          log.error(`[simulation][failed][${scenarioId}][${userId}][${error}]`);
        }
      }
    }

    return updatedBrowserInstance;
  }

  public getScenario(scenarioId: string): ScenarioMap {
    const scenario = this.scenarios.get(scenarioId);
    if (!scenario) {
      throw new Error(`Scenario ${scenarioId} not found`);
    }

    return scenario;
  }

  private isTestFailureError(error: unknown): error is TestError {
    return typeof error === 'object' && error !== null && 'error' in error && 'testId' in error;
  }
}

export const testManager = TestManager.getInstance();
