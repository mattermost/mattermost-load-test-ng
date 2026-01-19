// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import type {Browser, BrowserContext, Page} from 'playwright';

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

/**
 * Browser instance containing a Playwright page and user credentials.
 * This is the base interface that simulation scenarios receive.
 */
export type BrowserInstance = {
  browser: Browser | null;
  context: BrowserContext | null;
  page: Page | null;
  userId: string;
  password: string;
  createdAt: Date;
  state: SessionState;
};

interface LoggerFn {
  (message?: string, ...args: unknown[]): void;
}

export interface Logger {
  error: LoggerFn;
  warn: LoggerFn;
  info: LoggerFn;
}

/**
 * Registry item type for a browser simulation scenario.
 */
export interface SimulationRegistryItem {
  /** Unique identifier for the simulation */
  id: string;
  /** Human-readable name for the simulation */
  name?: string;
  /** Description of what the simulation does */
  description?: string;
  /** The scenario function to execute */
  scenario: (browserInstance: BrowserInstance, serverURL: string, logger: Logger, runInLoop?: boolean) => Promise<void>;
}
