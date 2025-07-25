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
 * Find the repository root by looking for specific marker files
 */
function findRootDirectory(startDir: string): string {
  const rootFiles = ['go.mod', '.git', 'bin', '.nvmrc'];

  let currentDir = startDir;
  while (currentDir !== path.dirname(currentDir)) {
    // Check if any of our defined root files exist in the current directory
    for (const rootFile of rootFiles) {
      if (fs.existsSync(path.join(currentDir, rootFile))) {
        return currentDir;
      }
    }

    // Move up one directory
    currentDir = path.dirname(currentDir);
  }

  // Fallback to the starting directory if no repo root found
  return startDir;
}

function findConfigJsonPath(): string {
  const currentDir = path.dirname(fileURLToPath(import.meta.url));
  const rootDir = findRootDirectory(currentDir);
  const configJsonPath = path.join(rootDir, 'config', 'config.json');
  return configJsonPath;
}

function loadConfigJson() {
  try {
    const configJsonPath = findConfigJsonPath();

    const configData = fs.readFileSync(configJsonPath, 'utf8');
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
