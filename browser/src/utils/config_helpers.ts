// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import fs from 'fs';
import {dirname, join} from 'path';
import {fileURLToPath} from 'url';

import pino from 'pino';
import {z as zod} from 'zod';

const logLabelLevels = Object.values(pino.levels.labels);

const SliceOfConfigJsonSchema = zod.object({
  ConnectionConfiguration: zod.object({
    ServerURL: zod.string().min(1, 'ConnectionConfiguration.ServerURL cannot be empty'),
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

const BrowserControllerConfigJsonSchema = zod.object({
  RunInHeadless: zod.boolean(),
  SimulationTimeoutMs: zod
    .number()
    .gte(0, 'SimulationTimeoutMs must be greater than or equal to 0. Set to 0 to disable timeout.'),
  EnabledPlugins: zod.boolean(),
  SimulationId: zod.string().min(1, 'SimulationId cannot be empty'),
});

const GoModFileName = 'go.mod';
const GitFolderName = '.git';
const BinFolderName = 'bin';
const NvmrcFileName = '.nvmrc';

/**
 * Find the repository root by looking for specific marker files
 * and returns the path to the root directory of the repository
 */
export function getRootDirectory(): string {
  const startDir = dirname(fileURLToPath(import.meta.url));

  const rootFiles = [GoModFileName, GitFolderName, BinFolderName, NvmrcFileName];

  let currentDir = startDir;
  while (currentDir !== dirname(currentDir)) {
    // Check if any of our defined root files exist in the current directory
    for (const rootFile of rootFiles) {
      if (fs.existsSync(join(currentDir, rootFile))) {
        return currentDir;
      }
    }

    // Move up one directory
    currentDir = dirname(currentDir);
  }

  // Fallback to the starting directory if no repo root found
  return startDir;
}

function loadJsonFile<T>(filePath: string, schema: zod.ZodSchema<T>): T {
  try {
    const fileData = fs.readFileSync(filePath, 'utf8');
    const parsedData = JSON.parse(fileData);
    const parsedSchema = schema.safeParse(parsedData);

    if (!parsedSchema.success) {
      const issues = parsedSchema.error.issues.map((issue) => {
        const fieldPath = issue.path.join('.') || 'unknownField';
        return `${issue.message} for '${fieldPath}'`;
      });
      throw new Error(`${issues.join(', ')}`);
    }

    return parsedSchema.data;
  } catch (error) {
    console.error(`Failed loading ${filePath}.`, error);
    process.exit(1);
  }
}

const ConfigFolderName = 'config';

const ConfigFileName = 'config.json';
const configJsonPath = join(getRootDirectory(), ConfigFolderName, ConfigFileName);
export const configJson = loadJsonFile(configJsonPath, SliceOfConfigJsonSchema);

const BrowserControllerConfigFileName = 'browsercontroller.json';
const browserControllerConfigJsonPath = join(getRootDirectory(), ConfigFolderName, BrowserControllerConfigFileName);
export const browserControllerConfigJson = loadJsonFile(
  browserControllerConfigJsonPath,
  BrowserControllerConfigJsonSchema,
);

const BrowserFolderName = 'browser';
const ScreenshotsFolderName = 'screenshots';
export const screenshotsDirectory = join(getRootDirectory(), BrowserFolderName, ScreenshotsFolderName);
