// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {app, log} from './app.js';

// This is a constant for the API URL of the LTBrowser server.
// Also defined in loadtest/control/browsercontroller/controller.go
const LT_BROWSER_API_URL = 'http://localhost:5000';

async function startServer() {
  try {
    const url = new URL(LT_BROWSER_API_URL);
    const host = url.hostname;
    const port = Number(url.port);

    await app.listen({host, port});
  } catch (err) {
    log.error(`[server] LTBrowser server failed to start: ${err}`);
    process.exit(1);
  }
}

async function stopServer(signal: string) {
  log.info(`[server] Received ${signal}, LTBrowser server stopping`);

  try {
    await app.close();
    log.info('[server] LTBrowser server stopped');
    process.exit(0);
  } catch (err) {
    log.error(`[server] LTBrowser server encountered error during shutdown: ${err}`);
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
