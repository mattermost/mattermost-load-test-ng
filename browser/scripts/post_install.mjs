// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {execSync} from 'child_process';
import path from 'path';
import {fileURLToPath} from 'url';
import dotenv from 'dotenv';
import chalk from 'chalk';

const dirname = path.dirname(fileURLToPath(import.meta.url));

try {
  const envPath = path.resolve(dirname, '../.env');
  const dotenvConfig = dotenv.config({path: envPath});

  if (dotenvConfig.error) {
    throw new Error(dotenvConfig.error);
  }

  console.log(chalk.green('Successfully loaded .env file'));
} catch (error) {
  console.error(chalk.red('Failed to load .env file:'), error);
  process.exit(1);
}

try {
  const playwrightCli = path.resolve(dirname, '../node_modules/.bin/playwright');

  console.log(
    chalk.blue(`Installing Playwright Chromium browser with BROWSERS_PATH=${process.env.PLAYWRIGHT_BROWSERS_PATH}`),
  );
  execSync(`${playwrightCli} install --with-deps chromium`, {stdio: 'inherit', env: process.env});

  console.log(chalk.green('Successfully installed Playwright Chromium browser'));
} catch (error) {
  console.error(chalk.red('Failed to install Playwright Chromium browser:'), error);
  process.exit(1);
}
