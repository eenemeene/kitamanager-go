import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  outputDir: './test-results',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 2, // Add 2 retries locally to handle flaky login
  workers: process.env.CI ? 1 : 4, // Reduce parallelism to avoid race conditions
  reporter: process.env.CI ? 'github' : 'list',
  timeout: 30000,

  use: {
    // Next.js runs on port 3000
    baseURL: process.env.BASE_URL || 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: process.env.VIDEO ? { mode: 'on', size: { width: 1280, height: 720 } } : 'off',
    launchOptions: {
      // Add slight slowdown for stability (50ms between actions)
      slowMo: process.env.SLOWMO ? parseInt(process.env.SLOWMO) : 50,
    },
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'mobile-chrome',
      use: { ...devices['Pixel 7'] },
    },
    {
      name: 'tablet-chrome',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 768, height: 1024 },
      },
    },
  ],

  // Web server configuration for local development
  ...(process.env.CI
    ? {}
    : {
        webServer: {
          // make dev starts DB, API, and Next.js dev server
          command: 'cd .. && make dev',
          url: 'http://localhost:3000',
          reuseExistingServer: true,
          timeout: 120000,
        },
      }),
});
