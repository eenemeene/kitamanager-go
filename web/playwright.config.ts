import { defineConfig, devices } from 'playwright/test'

export default defineConfig({
  testDir: './e2e',
  outputDir: './test-results',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? 'github' : 'list',
  timeout: 30000,

  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:8080',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    // Video recording - set VIDEO=1 to enable, videos saved to test-results/
    video: process.env.VIDEO ? { mode: 'on', size: { width: 1280, height: 720 } } : 'off',
    // Slow down actions for human viewing - set SLOWMO=500 for 500ms delay
    launchOptions: {
      slowMo: process.env.SLOWMO ? parseInt(process.env.SLOWMO) : 0,
    },
  },

  projects: [
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Web server configuration for local development
  // In CI, we start services manually for more control
  ...(process.env.CI
    ? {}
    : {
        webServer: {
          command: 'cd .. && make dev',
          url: 'http://localhost:8080/api/v1/health',
          reuseExistingServer: true,
          timeout: 120000,
        },
      }),
})
