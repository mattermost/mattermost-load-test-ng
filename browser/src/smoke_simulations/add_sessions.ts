import ms from 'ms';

import {browserTestSessionManager} from '../lib/browser_manager.js';
import {getMattermostServerURL} from '../utils/config.js';
import {addSessionsConfig} from './configurations.js';
import {SimulationIds} from 'src/simulations/registry.js';

async function createBrowserSession(user: {username: string; password: string}, simulationId: SimulationIds) {
  console.info(`🔍 Creating session for ${user.username}`);

  try {
    const r = await browserTestSessionManager.createBrowserSession(
      user.username,
      user.password,
      getMattermostServerURL(),
      simulationId,
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
async function monitor() {
  if (addSessionsConfig.sessionMonitorIntervalMs === 0) {
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

  mfunc();
  mInterval = setInterval(() => {
    mfunc();
  }, addSessionsConfig.sessionMonitorIntervalMs);
}

function verifyConfig(): string | null {
  if (!getMattermostServerURL()) {
    return 'Mattermost Server URL is not set';
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
  console.info(`ℹ️ Users: ${addSessionsConfig.users.length}`);
  console.info(`ℹ️ Test duration: ${ms(addSessionsConfig.testDurationMs, {long: true})}`);

  const cs: Promise<boolean>[] = [];
  for (let i = 0; i < addSessionsConfig.users.length; i++) {
    cs.push(createBrowserSession(addSessionsConfig.users[i], addSessionsConfig.simulations[i]));
  }

  const rs = await Promise.allSettled(cs);
  if (rs.every((r) => r.status === 'fulfilled' && r.value === true)) {
    console.info('✅ All sessions created');
  } else {
    console.error('❌ Some or all sessions failed to be created');
  }

  stop();
}

function stop() {
  setTimeout(() => {
    if (mInterval) {
      clearInterval(mInterval);
    }

    console.info(
      `🧹 Stopped "Add sessions" smoke simulations after ${ms(addSessionsConfig.testDurationMs, {long: true})}`,
    );
    process.exit(0);
  }, addSessionsConfig.testDurationMs);
}

(function () {
  run();
  monitor();
  stop();
})();
