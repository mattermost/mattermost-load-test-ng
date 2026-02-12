// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {readFileSync} from 'node:fs';
import {dirname, join} from 'node:path';
import {fileURLToPath} from 'node:url';

import ms from 'ms';

import type {SmokeSimulationConfig} from './types.js';
import {browserTestSessionManager} from '../services/browser_manager.js';
import {SimulationsRegistry} from '../registry.js';
import {getMattermostServerURL} from '../utils/config_accessors.js';

function readConfig(): SmokeSimulationConfig {
  try {
    const __dirname = dirname(fileURLToPath(import.meta.url));
    const configPath = join(__dirname, 'smoke_simulation.json');
    const config = readFileSync(configPath, 'utf-8');
    return JSON.parse(config);
  } catch (error) {
    console.error(`❌ Failed to read config: ${error}`);
    process.exit(1);
  }
}

async function createBrowserSession(user: {username: string; password: string}, simulationId: string) {
  console.info(`🔍 Creating session for ${user.username}`);

  try {
    const r = await browserTestSessionManager.createBrowserSession(
      user.username,
      user.password,
      smokeSimulationConfig.serverURL,
      simulationId,
      smokeSimulationConfig.RunInHeadless,
    );

    if (r.isCreated) {
      console.info(`✅ Session created: ${r.message}`);
      console.info(`⌛️ Starting simulation ${simulationId} for ${user.username}`);
      return true;
    }
    console.error(`❌ Failed: ${r.message}`);
    return false;
  } catch (error) {
    console.error('❌ Exception:', error);
    return false;
  }
}

let mInterval: NodeJS.Timeout | null = null;
function monitor() {
  if (smokeSimulationConfig.sessionMonitorIntervalMs === 0) {
    return;
  }

  function mfunc() {
    const as = browserTestSessionManager.getActiveBrowserSessions();

    let m = '📋 Active browser sessions:';
    as.forEach((session) => {
      m += `${session.userId}->${session.state}, `;
    });
    m = m.trim().slice(0, -1);

    if (as.length === 0) {
      console.info('🔍 No current active browser sessions');
    } else {
      console.info(m);
    }
  }

  console.info('🔍 Starting monitor');
  mfunc();
  mInterval = setInterval(() => {
    mfunc();
  }, smokeSimulationConfig.sessionMonitorIntervalMs);
}

function verifyConfig(): string | null {
  if (!smokeSimulationConfig.serverURL) {
    return 'Mattermost Server URL is not set in smoke_simulation.json, check "serverURL" field';
  }

  if (!smokeSimulationConfig.users || smokeSimulationConfig.users.length === 0) {
    return 'Users are not set in smoke_simulation.json, check "users" field';
  }

  const simulationIds = SimulationsRegistry.map((sim) => sim.id);
  if (!smokeSimulationConfig.simulations || smokeSimulationConfig.simulations.length === 0) {
    return 'Simulations are not set in smoke_simulation.json, check "simulations" field';
  } else if (!smokeSimulationConfig.simulations.every((sim) => simulationIds.includes(sim))) {
    return 'All or some simulations ids are not valid in smoke_simulation.json, check "simulations" field';
  }

  if (
    !Number.isInteger(smokeSimulationConfig.sessionMonitorIntervalMs) ||
    smokeSimulationConfig.sessionMonitorIntervalMs < 0
  ) {
    return 'Session monitor interval is not set in smoke_simulation.json, check "sessionMonitorIntervalMs" field';
  }

  if (!Number.isInteger(smokeSimulationConfig.testDurationMs) || smokeSimulationConfig.testDurationMs < 0) {
    return 'Test duration is not set in smoke_simulation.json, check "testDurationMs" field';
  }

  return null;
}

async function run() {
  const e = verifyConfig();
  if (e) {
    console.error(`❌ Config error: ${e}`);
    process.exit(1);
  }

  console.info('ℹ️ Starting "Add sessions" smoke simulations');
  console.info(`ℹ️ MM App URL: ${getMattermostServerURL()}`);
  console.info(`ℹ️ Users: ${smokeSimulationConfig.users.length}`);
  console.info(`ℹ️ Test duration: ${ms(smokeSimulationConfig.testDurationMs, {long: true})}`);

  const cs: Array<Promise<boolean>> = [];
  for (let i = 0; i < smokeSimulationConfig.users.length; i++) {
    cs.push(createBrowserSession(smokeSimulationConfig.users[i], smokeSimulationConfig.simulations[i]));
  }

  const rs = await Promise.allSettled(cs);
  if (rs.every((r) => r.status === 'fulfilled' && r.value === true)) {
    console.info('✅ All sessions created');
  } else {
    console.error('❌ Some or all sessions failed to be created');
  }
}

function stop() {
  setTimeout(() => {
    if (mInterval) {
      clearInterval(mInterval);
    }

    console.info(
      `🧹 Stopped "Add sessions" smoke simulations after ${ms(smokeSimulationConfig.testDurationMs, {long: true})}`,
    );
    process.exit(0);
  }, smokeSimulationConfig.testDurationMs);
}

let smokeSimulationConfig: SmokeSimulationConfig;
(async function () {
  smokeSimulationConfig = readConfig();
  monitor();
  await run();
  stop();
})();
