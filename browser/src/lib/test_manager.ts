// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {log} from 'src/app.js';
import * as tests from 'src/tests/scenario1.js';

import type {BrowserInstance, ActiveBrowserSessions} from 'src/lib/browser_manager.js';
import {SessionState} from 'src/lib/browser_manager.js';

interface TestError {
  error: Error;
  testId: string;
}

export async function startTest(
  browserInstance: BrowserInstance,
  activeBrowserSessions: ActiveBrowserSessions,
  serverURL: string,
  userId: string,
  scenarioId: string,
): Promise<BrowserInstance | undefined> {
  let updatedBrowserInstance: BrowserInstance | undefined = {...browserInstance};

  try {
    log.info(`[simulation][start][${scenarioId}][${userId}]`);

    await getScenarioFromId(scenarioId)(browserInstance, serverURL);

    updatedBrowserInstance.state = SessionState.COMPLETED;

    log.info(`[simulation][end][${scenarioId}][${userId}]`);
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

      if (isTestError(error)) {
        log.error(`[simulation][failed][${scenarioId}][${userId}][${error.testId}][${error.error.message}]`);
      } else {
        log.error(`[simulation][failed][${scenarioId}][${userId}][${error}]`);
      }
    }
  } finally {
    log.info(`[simulation][end][${scenarioId}][${userId}]`);
  }

  return updatedBrowserInstance;
}

export function getScenarioFromId(scenarioId: string) {
  switch (scenarioId) {
    case 'scenario1':
      return tests.scenario1;
    default:
      throw new Error(`Scenario ${scenarioId} not found`);
  }
}

function isTestError(error: unknown): error is TestError {
  return typeof error === 'object' && error !== null && 'error' in error && 'testId' in error;
}
