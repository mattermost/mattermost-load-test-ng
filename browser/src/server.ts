// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {app, log} from './app.js';

async function startServer() {
  try {
    const browserAgentApiURL = process.env.BROWSER_AGENT_API_URL;

    if (!browserAgentApiURL) {
      throw new Error('BROWSER_AGENT_API_URL must be set');
    }

    const apiURL = new URL(browserAgentApiURL);
    const port = Number(apiURL.port);
    const host = apiURL.hostname;

    await app.listen({port, host});
  } catch (err) {
    log.error(`[server] Server failed to start: ${err}`);
    process.exit(1);
  }
}

async function stopServer(signal: string) {
  log.info(`[server] Received ${signal}, Server stopping`);

  try {
    await app.close();
    log.info('[server] Server stopped');
    process.exit(0);
  } catch (err) {
    log.error(`[server] Error during shutdown: ${err}`);
    process.exit(1);
  }
}

// Register shutdown handlers
process.on('SIGTERM', () => stopServer('SIGTERM'));
process.on('SIGINT', () => stopServer('SIGINT'));

// Handle uncaught errors
process.on('uncaughtException', (err) => {
  log.error(`[server] Uncaught exception: ${err}`);
  stopServer('uncaughtException');
});

// Handle unhandled rejections
process.on('unhandledRejection', (reason, promise) => {
  log.error(`[server] Unhandled rejection at: ${promise}, reason: ${reason}`);
  stopServer('unhandledRejection');
});

startServer();
