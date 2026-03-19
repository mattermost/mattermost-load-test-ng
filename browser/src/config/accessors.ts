// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {browserControllerConfigJson} from './loader.js';

/**
 * Server URl is always passed as a parameter to the browser controller while
 * its created. So we don't need to read it from the config.json. But we need
 * hardcoded value for tests and smoke simulations.
 */
export function getMattermostServerURL(): string {
  return 'http://localhost:8065';
}

export function isConsoleLoggingEnabled(): boolean {
  return browserControllerConfigJson.LogSettings.EnableConsole;
}

export function getConsoleLoggingLevel(): string {
  return browserControllerConfigJson.LogSettings.ConsoleLevel;
}

export function isFileLoggingEnabled(): boolean {
  return browserControllerConfigJson.LogSettings.EnableFile;
}

export function getFileLoggingLevel(): string {
  return browserControllerConfigJson.LogSettings.FileLevel;
}

export function getFileLoggingLocation(): string {
  return browserControllerConfigJson.LogSettings.FileLocation;
}

export function isBrowserHeadless(): boolean {
  return browserControllerConfigJson.RunInHeadless;
}

const DEFAULT_SIMULATION_TIMEOUT_MS = 60_000;
export function getSimulationTimeoutMs(): number {
  return browserControllerConfigJson.SimulationTimeoutMs ?? DEFAULT_SIMULATION_TIMEOUT_MS;
}

export function getSimulationId(): string {
  return browserControllerConfigJson.SimulationId;
}

export function arePluginsEnabled(): boolean {
  return browserControllerConfigJson.EnabledPlugins ?? false;
}
