import {SimulationIds} from '../simulations/registry.js';

export const addSessionsConfig = {
  users: [
    {
      username: 'user1@example.com',
      password: 'Password-1!',
    },
    {
      username: 'user2@example.com',
      password: 'password-1',
    },
  ],
  sessionMonitorIntervalMs: 5_000,
  testDurationMs: 120_000,
  simulations: [SimulationIds.postAndScroll, SimulationIds.postAndScroll],
};
