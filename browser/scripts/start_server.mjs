// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {execSync} from 'child_process';
import chalk from 'chalk';

try {
  console.log(chalk.blue('Installing dependencies'));
  execSync('npm install', {stdio: 'inherit'});
  console.log(chalk.green('Successfully installed dependencies'));

  console.log(chalk.blue('Building server'));
  execSync('npm run server:build', {stdio: 'inherit'});
  console.log(chalk.green('Successfully built server'));

  console.log(chalk.blue('Starting server'));
  execSync('node build/index.js', {stdio: 'inherit'});
} catch (error) {
  console.error(chalk.red('Failed to setup and start server:'), error.message);
  process.exit(1);
}
