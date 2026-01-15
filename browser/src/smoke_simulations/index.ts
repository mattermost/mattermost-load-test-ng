// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import ms from 'ms';

import {browserTestSessionManager} from '../services/browser_manager.js';
import {getMattermostServerURL} from '../utils/config_accessors.js';
import {SimulationsRegistry} from '../simulations/registry.js';

// @ts-ignore smoke_simulation.json may not be present in the project depending upon usage
import smokeSimulationConfig from './smoke_simulation.json' with {type: 'json'};

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
    } else {
      console.error(`❌ Failed: ${r.message}`);
      return false;
    }
  } catch (error) {
    console.error(`❌ Exception:`, error);
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
  } else if (!smokeSimulationConfig.simulations.every((sim: any) => simulationIds.includes(sim))) {
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

  const cs: Promise<boolean>[] = [];
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

(async function () {
  monitor();
  await run();
  stop();
})();
