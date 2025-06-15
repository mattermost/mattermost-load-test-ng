// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {execSync} from 'child_process';

try {
  console.log('Installing dependencies');
  execSync('npm install', {stdio: 'inherit'});
  console.log('Successfully installed dependencies');

  console.log('Building server');
  execSync('npm run server:build', {stdio: 'inherit'});
  console.log('Successfully built server');

  console.log('Starting server');
  execSync('node build/server.js', {stdio: 'inherit'});
} catch (error) {
  console.error('Failed to setup and start server:', error.message);
  process.exit(1);
}
