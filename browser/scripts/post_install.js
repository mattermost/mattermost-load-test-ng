import {execSync} from 'child_process';
import path from 'path';
import {fileURLToPath} from 'url';
import dotenv from 'dotenv';

const dirname = path.dirname(fileURLToPath(import.meta.url));
const envPath = path.resolve(dirname, '../.env');
const dotenvConfig = dotenv.config({path: envPath});

if (dotenvConfig.error) {
  console.error('Post install failed: Error loading .env file:', dotenvConfig.error);
  process.exit(1);
}

try {
  const playwrightBin = path.resolve(__dirname, '../node_modules/.bin/playwright');

  console.log(`Installing Playwright Chromium browser with BROWSERS_PATH=${process.env.PLAYWRIGHT_BROWSERS_PATH}`);
  execSync(`${playwrightBin} install --with-deps chromium`, {stdio: 'inherit', env: process.env});

  console.log('Post install completed.');
} catch (error) {
  console.error('Post install failed:', error);
  process.exit(1);
}
