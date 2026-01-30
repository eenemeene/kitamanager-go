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
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
  ],

  // Web server configuration for local development
  ...(process.env.CI
    ? {}
    : {
        webServer: [
          {
            // Start the API server
            command: 'cd .. && LOGIN_RATE_LIMIT_PER_MINUTE=0 make dev',
            url: 'http://localhost:8080/api/v1/health',
            reuseExistingServer: true,
            timeout: 120000,
          },
          {
            // Start Next.js dev server
            command: 'npm run dev',
            url: 'http://localhost:3000',
            reuseExistingServer: true,
            timeout: 60000,
          },
        ],
      }),
});
