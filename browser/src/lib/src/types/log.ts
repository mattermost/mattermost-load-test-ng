// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

interface LoggerFn {
  (message?: string, ...args: unknown[]): void;
}

export interface Logger {
  error: LoggerFn;
  warn: LoggerFn;
  info: LoggerFn;
}
