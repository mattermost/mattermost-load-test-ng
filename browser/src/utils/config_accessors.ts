// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {configJson, browserControllerConfigJson} from './config_helpers.js';

console.log('browserControllerConfigJson', browserControllerConfigJson);
console.log('configJson', configJson);
export function getMattermostServerURL(): string {
  return configJson.ConnectionConfiguration.ServerURL;
}

export function isConsoleLoggingEnabled(): boolean {
  return configJson.BrowserLogSettings.EnableConsole;
}

export function getConsoleLoggingLevel(): string {
  return configJson.BrowserLogSettings.ConsoleLevel;
}

export function isFileLoggingEnabled(): boolean {
  return configJson.BrowserLogSettings.EnableFile;
}

export function getFileLoggingLevel(): string {
  return configJson.BrowserLogSettings.FileLevel;
}

export function getFileLoggingLocation(): string {
  return configJson.BrowserLogSettings.FileLocation;
}

export function isBrowserHeadless(): boolean {
  return browserControllerConfigJson.RunInHeadless;
}

const DEFAULT_SIMULATION_TIMEOUT_MS = 60_000;
export function getSimulationTimeoutMs(): number {
  return browserControllerConfigJson.SimulationTimeoutMs ?? DEFAULT_SIMULATION_TIMEOUT_MS;
}
