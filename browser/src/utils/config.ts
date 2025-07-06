// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import path from 'path';
import fs from 'fs';
import {fileURLToPath} from 'url';

/**
 * Read and parse the config.json file located in config/config.json
 */
export function loadConfigJson() {
  try {
    const dirname = path.dirname(fileURLToPath(import.meta.url));
    const configPath = path.resolve(dirname, '../../../config/config.json');

    const configData = fs.readFileSync(configPath, 'utf8');
    return JSON.parse(configData);
  } catch (error) {
    console.error('Failed to load config.json file:', error);
    process.exit(1);
  }
}

export const configJson = loadConfigJson();

export function getMattermostServerURL(): string {
  return configJson.ConnectionConfiguration.ServerURL;
}

export function isBrowserHeadless(): boolean {
  return configJson.BrowserConfiguration.Headless;
}

export function isConsoleLoggingEnabled(): boolean {
  return configJson.BrowserLogSettings?.EnableConsole ?? false;
}

export function getConsoleLoggingLevel(): string {
  return configJson.BrowserLogSettings.ConsoleLevel;
}

export function isFileLoggingEnabled(): boolean {
  return configJson.BrowserLogSettings?.EnableFile ?? true;
}

export function getFileLoggingLevel(): string {
  return configJson.BrowserLogSettings.FileLevel;
}

export function getFileLoggingLocation(): string {
  return configJson.BrowserLogSettings.FileLocation;
}

/**
 * Generates a random port in the specified range
 * Useful when you want to avoid sequential ports for parallel tests
 * If not provided, the default range is 10000-65000
 */
export function getRandomPortForTests(min = 10000, max = 65000): number {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}
