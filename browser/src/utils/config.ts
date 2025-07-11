// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import path from 'path';
import fs from 'fs';
import {fileURLToPath} from 'url';
import {z as zod} from 'zod';
import pino from 'pino';

const logLabelLevels = Object.values(pino.levels.labels);

const SliceOfConfigJsonSchema = zod.object({
  ConnectionConfiguration: zod.object({
    ServerURL: zod.string().min(1, 'ConnectionConfiguration.ServerURL cannot be empty'),
  }),
  BrowserConfiguration: zod.object({
    Headless: zod.boolean(),
  }),
  BrowserLogSettings: zod.object({
    EnableConsole: zod.boolean(),
    ConsoleLevel: zod.enum(logLabelLevels, {
      message: `BrowserLogSettings.ConsoleLevel must be one of: ${logLabelLevels.join(', ')}`,
    }),
    EnableFile: zod.boolean(),
    FileLevel: zod.enum(logLabelLevels, {
      message: `BrowserLogSettings.FileLevel must be one of: ${logLabelLevels.join(', ')}`,
    }),
    FileLocation: zod.string().min(1, 'BrowserLogSettings.FileLocation cannot be empty'),
  }),
});

export const configJson = loadConfigJson();

export function getMattermostServerURL(): string {
  return configJson.ConnectionConfiguration.ServerURL;
}

export function isBrowserHeadless(): boolean {
  return configJson.BrowserConfiguration.Headless;
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

/**
 * Read and parse the config.json file located in config/config.json
 */
function loadConfigJson() {
  try {
    const dirname = path.dirname(fileURLToPath(import.meta.url));
    const configPath = path.resolve(dirname, '../../../config/config.json');

    const configData = fs.readFileSync(configPath, 'utf8');
    const rawConfig = JSON.parse(configData);

    const parsedConfig = SliceOfConfigJsonSchema.safeParse(rawConfig);

    if (!parsedConfig.success) {
      const issues = parsedConfig.error.issues.map((issue) => {
        const fieldPath = issue.path.join('.') || 'unknownField';
        return `${issue.message} for '${fieldPath}'`;
      });

      throw new Error(`${issues.join(', ')}`);
    }

    return parsedConfig.data;
  } catch (error) {
    console.error('Failed loading config.json.', error);
    process.exit(1);
  }
}
