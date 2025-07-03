import {defineConfig, devices} from '@playwright/test';

export default defineConfig({
  testDir: './src/e2e_specs',
  outputDir: './e2etest_results',
  fullyParallel: true,
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
