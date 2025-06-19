// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {app, log} from './app.js';

async function startServer() {
  try {
    const portNumber = Number(process.env.PORT) || 8080;
    const host = process.env.HOST || '127.0.0.1';
    await app.listen({port: portNumber, host});
  } catch (err) {
    log.error('[server] Server failed to start', err);
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
    log.error('[server] Error during shutdown:', err);
    process.exit(1);
  }
}

// Register shutdown handlers
process.on('SIGTERM', () => stopServer('SIGTERM'));
process.on('SIGINT', () => stopServer('SIGINT'));

// Handle uncaught errors
process.on('uncaughtException', (err) => {
  log.error('[server] Uncaught exception:', err);
  stopServer('uncaughtException');
});

// Handle unhandled rejections
process.on('unhandledRejection', (reason, promise) => {
  log.error('[server] Unhandled rejection at:', promise, 'reason:', reason);
  stopServer('unhandledRejection');
});

startServer();
