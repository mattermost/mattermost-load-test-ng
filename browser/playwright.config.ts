import {defineConfig, devices} from '@playwright/test';

export default defineConfig({
  testDir: './src/e2e',
  outputDir: './e2e_results',
  fullyParallel: true,
  use: {
    trace: 'off',
  },
  workers: "100%",
  projects: [
    {
      name: 'chromium',
      use: {...devices['Desktop Chrome']},
    },
  ],
});
