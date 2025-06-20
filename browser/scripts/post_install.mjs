// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {execSync} from 'child_process';
import path from 'path';
import {fileURLToPath} from 'url';

const dirname = path.dirname(fileURLToPath(import.meta.url));

const withHermitInstallation = 'PLAYWRIGHT_BROWSERS_PATH=0';

try {
  const playwrightCli = path.resolve(dirname, '../node_modules/.bin/playwright');

  console.log('Installing Playwright Chromium browser');
  execSync(`${withHermitInstallation} ${playwrightCli} install --with-deps chromium`, {stdio: 'inherit'});

  console.log(
    'Successfully installed Playwright Chromium browser binaries in node_modules/playwright-core/.local-browsers',
  );
} catch (error) {
  console.error('Failed to install Playwright Chromium browser:', error);
  process.exit(1);
}
