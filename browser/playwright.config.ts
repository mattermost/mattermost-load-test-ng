import {defineConfig, devices} from '@playwright/test';

export default defineConfig({
  testDir: './src/tests',
  outputDir: './test-results',
  fullyParallel: true,
  retries: 0,
  workers: 1, // TODO: Adjust this configuration
  use: {
    trace: 'off',
  },

  projects: [
    {
      name: 'chromium',
      use: {...devices['Desktop Chrome']},
    },
  ],
});
