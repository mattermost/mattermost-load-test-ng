// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {execSync} from 'child_process';

const DEFAULT_STANDALONE_PORT = 5000;
const DEFAULT_STANDALONE_HOST = '127.0.0.1';
const DEFAULT_STANDALONE_API_URL = `http://${DEFAULT_STANDALONE_HOST}:${DEFAULT_STANDALONE_PORT}`;

const args = process.argv.slice(2);
const watch = args.includes('--watch') ? 'watch' : '';
const dev = args.includes('--dev');
const isDev = watch || dev;

try {
  console.log('Installing dependencies');
  execSync('npm install', {stdio: 'inherit'});
  console.log('Successfully installed dependencies');

  if (isDev) {
    execSync(`BROWSER_AGENT_API_URL=${DEFAULT_STANDALONE_API_URL} tsx ${watch} src/server.ts`, {stdio: 'inherit'});
  } else {
    console.log('Building server');
    execSync('npm run server:build', {stdio: 'inherit'});
    console.log('Successfully built server');

    const browserAgentApiURL = process.env.BROWSER_AGENT_API_URL;

    if (!browserAgentApiURL) {
      console.log(`Starting server in standalone mode on ${DEFAULT_STANDALONE_API_URL}`);
      execSync(`BROWSER_AGENT_API_URL=${DEFAULT_STANDALONE_API_URL} node build/server.js`, {stdio: 'inherit'});
    } else {
      console.log(`Starting server in load-test cluster on ${browserAgentApiURL}`);
      execSync(`BROWSER_AGENT_API_URL=${browserAgentApiURL} node build/server.js`, {stdio: 'inherit'});
    }
  }
} catch (error) {
  console.error('Failed to setup and start server:', error);
  process.exit(1);
}
