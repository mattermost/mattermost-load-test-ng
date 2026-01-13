// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {type BrowserInstance, type SimulationRegistryItem} from '@mattermost/load-test-ng-browser';

import {postAndScrollScenario} from './post_and_scroll_scenario.js';
import {log} from '../app.js';

// Import playbooks simulations
import {PlaybooksSimulationIds, PlaybooksSimulationsRegistry, type SimulationLogger} from 'playbooks-load-simulations';

// Combine core simulation IDs with playbooks simulation IDs
export enum SimulationIds {
  // Core simulations
  postAndScroll = 'postAndScroll',
  // Playbooks simulations
  createAndRunPlaybook = PlaybooksSimulationIds.createAndRunPlaybook,
  browsePlaybooks = PlaybooksSimulationIds.browsePlaybooks,
  viewPlaybookRun = PlaybooksSimulationIds.viewPlaybookRun,
}

/**
 * Creates a logger adapter that converts the app's pino logger to the SimulationLogger interface
 * expected by the playbooks simulations.
 */
function createLoggerAdapter(): SimulationLogger {
  return {
    info: (message: string) => log.info(message),
    error: (message: string) => log.error(message),
    warn: (message: string) => log.warn(message),
    // debug falls back to info since the app logger doesn't have a debug level
    debug: (message: string) => log.info(message),
  };
}

/**
 * Creates an adapter function that wraps a playbooks simulation scenario
 * to match the BrowserInstance interface used by the load testing framework.
 */
function createPlaybooksAdapter(
  playbooksScenario: (
    browserInstance: {page: any; userId: string; password: string},
    serverURL: string,
    logger?: SimulationLogger,
  ) => Promise<void>,
): (browserInstance: BrowserInstance, serverURL: string) => Promise<void> {
  return async (browserInstance: BrowserInstance, serverURL: string) => {
    if (!browserInstance.page) {
      throw new Error('Page is not initialized');
    }

    const playbooksInstance = {
      page: browserInstance.page,
      userId: browserInstance.userId,
      password: browserInstance.password,
    };

    const logger = createLoggerAdapter();
    await playbooksScenario(playbooksInstance, serverURL, logger);
  };
}

// Build the registry by combining core and playbooks simulations
export const SimulationsRegistry: SimulationRegistryItem[] = [
  // Core simulations
  {
    id: SimulationIds.postAndScroll,
    name: 'Post and Scroll scenario',
    description: 'A basic scenario that posts and scrolls in a channel',
    scenario: postAndScrollScenario,
  },
  // Playbooks simulations - adapted from playbooks-load-simulations package
  ...PlaybooksSimulationsRegistry.map((playbooksSimulation) => ({
    id: playbooksSimulation.id as unknown as SimulationIds,
    name: playbooksSimulation.name,
    description: playbooksSimulation.description,
    scenario: createPlaybooksAdapter(playbooksSimulation.scenario),
  })),
];
