import { defineConfig } from '@playwright/test'
import { baseURL } from './helpers.js'

export default defineConfig({
  testDir: 'tests',
  globalSetup: './global-setup.js',
  // the tests of a file build on each other, a retry would start halfway
  retries: 0,
  workers: process.env.CI ? 4 : undefined,
  reporter: 'list',
  use: { baseURL, viewport: { width: 1280, height: 800 }, screenshot: 'only-on-failure', trace: 'retain-on-failure' },
  projects: [
    { name: 'e2e', testIgnore: 'admin.spec.js' },
    // maintenance mode blocks every login, so the admin panel runs after all other specs
    { name: 'admin', testMatch: 'admin.spec.js', dependencies: ['e2e'] },
  ],
})
