import {defineConfig, devices} from '@playwright/test';

export default defineConfig({
  testDir: '../tests',
  outputDir: '../test-results',
  fullyParallel: true,
  retries: 0,
  workers: 1, // TODO: Adjust this configuration
  // reporter: './playwright_reporter.ts', // TODO: Add this back in when we have a reporter
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
