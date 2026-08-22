import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  outputDir: './test-results',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 2, // Add 2 retries locally to handle flaky login
  workers: process.env.CI ? 1 : 4, // Reduce parallelism to avoid race conditions
  // In CI each shard writes a blob instead of its own report, and a follow-up
  // job merges all six into one HTML report. Six shards previously produced six
  // separate artifacts, each a fragment: finding the screenshots for one failing
  // test meant guessing which shard had run it and downloading that zip. The
  // merged report is a single browsable page carrying every screenshot, diff and
  // trace. `github` stays alongside it for the inline PR annotations.
  reporter: process.env.CI ? [['github'], ['blob']] : 'list',
  timeout: 30000,

  use: {
    // Next.js runs on port 3000
    baseURL: process.env.BASE_URL || 'http://localhost:3000',
    timezoneId: 'Europe/Berlin',
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
    // The two device projects run the product specs at a real phone and tablet
    // size. Both exclude the specs that pin their own viewport with
    // `test.use({ viewport })` — responsive.spec.ts and navigation.spec.ts already
    // assert 375 / 820 / 1280 layouts themselves, and running them inside a device
    // project overrides only the width while keeping that device's touch emulation
    // and pixel ratio: a hybrid that matches no real device and asserts nothing
    // useful. They keep running in full under chromium.
    //
    // The longer timeout is for the device emulation, not for slack in the
    // assertions: a 3x pixel ratio at 412px renders slower, and several specs do
    // login + navigate + seed + reload before their first assertion.
    {
      name: 'mobile-chrome',
      testIgnore: [/responsive\.spec\.ts/, /navigation\.spec\.ts/],
      timeout: 60000,
      use: { ...devices['Pixel 7'], contextOptions: { reducedMotion: 'reduce' } },
    },
    {
      name: 'tablet-chrome',
      testIgnore: [/responsive\.spec\.ts/, /navigation\.spec\.ts/],
      timeout: 60000,
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 768, height: 1024 },
        contextOptions: { reducedMotion: 'reduce' },
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
