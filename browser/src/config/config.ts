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
