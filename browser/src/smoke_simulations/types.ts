// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

export interface SmokeSimulationConfig {
  users: Array<{username: string; password: string}>;
  simulations: string[];
  serverURL: string;
  RunInHeadless: boolean;
  testDurationMs: number;
  sessionMonitorIntervalMs: number;
}
