// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

/**
 * Generates a random port in the specified range
 * Useful when you want to avoid sequential ports for parallel tests
 */
export function getRandomPort(): number {
  const min = 10000;
  const max = 65000;
  return Math.floor(Math.random() * (max - min + 1)) + min;
}
