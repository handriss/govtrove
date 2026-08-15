import { defineConfig, devices } from '@playwright/test';

// Defaults to production so these can be run as a periodic check with no setup.
// Point elsewhere with E2E_BASE_URL=http://localhost:5173 npm test
const BASE_URL = process.env.E2E_BASE_URL || 'https://app.govtrove.com';

// Searches run here land in search_events and PostHog like any other traffic.
// The marker below makes them filterable so they don't pollute the zero-result
// rate or the usage numbers:
//   WHERE user_agent NOT LIKE '%GovTroveE2E%'
const E2E_UA =
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 ' +
  '(KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36 GovTroveE2E/1.0';

export default defineConfig({
  testDir: './tests',
  // Search is the slowest thing here: an all-stopword phrase like "IT" scans a
  // lot of text, and the rescue endpoint runs several probes before answering.
  timeout: 120_000,
  expect: { timeout: 30_000 },
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: BASE_URL,
    userAgent: E2E_UA,
    actionTimeout: 30_000,
    navigationTimeout: 60_000,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    { name: 'desktop', use: { ...devices['Desktop Chrome'], userAgent: E2E_UA, viewport: { width: 1440, height: 900 } } },
    // The mobile layout is a different component tree (bottom nav, full-screen
    // filter sheet), so the same journeys are re-run at phone width.
    //
    // Chromium emulation rather than devices['iPhone 13'], which is WebKit and
    // would mean a second ~100MB browser download. The layout is driven by CSS
    // breakpoints, so Chromium at phone width exercises the same components.
    // Add a webkit project (and `playwright install webkit`) if Safari-specific
    // rendering ever becomes worth catching.
    {
      name: 'mobile',
      use: {
        ...devices['Desktop Chrome'],
        userAgent: E2E_UA,
        viewport: { width: 390, height: 844 },
        isMobile: true,
        hasTouch: true,
        deviceScaleFactor: 2,
      },
    },
  ],
});
