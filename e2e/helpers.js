import { test as base, expect } from '@playwright/test'

export { expect }
export const baseURL = process.env.BASE_URL || 'http://e2e-app:8080'
export const mailpit = process.env.MAILPIT || 'http://e2e-mail:8025'
export const adminToken = 'e2e-only-admin-token-not-a-secret-1234'
export const password = 'e2e-password-1'

// one user per spec file, so the files run in parallel without seeing each other's data; created by global-setup.js
export const users = ['account', 'admin', 'antennas', 'auth', 'coverage', 'links', 'owner-a', 'owner-b', 'share', 'sites']

// map tiles are stubbed in the browser, nothing reaches the tile server; confirm() dialogs are accepted
export const test = base.extend({
  page: async ({ page }, use) => {
    await page.route('**/tiles/**', (route) => route.fulfill({ status: 204 }))
    page.on('dialog', (dialog) => dialog.accept())
    await use(page)
  },
})

/** The first innermost element matching selector that contains text, like cy.contains(selector, text). */
export function c(scope, selector, text) {
  const root = typeof scope.page === 'function' ? scope.page() : scope
  return scope.locator(selector, { hasText: text }).filter({ hasNot: root.locator(selector, { hasText: text }) }).first()
}

export const pathname = (page) => new URL(page.url()).pathname

/** A page in its own context for setup in beforeAll. */
export const newPage = (browser) => browser.newPage({ baseURL })

export const session = async (page) => (await page.context().cookies()).find((k) => k.name === 'radiopath_session')

// login without the form: a request carries no Origin, so the CSRF check lets it pass
export async function login(page, name, pw = password) {
  const res = await page.request.post('/login', { form: { name, password: pw, next: '/' }, maxRedirects: 0 })
  expect(res.status(), `login ${name}`).toBe(303)
}

export async function addSite(page, name, lat, lon, height = 10) {
  await page.request.post('/sites', { form: { name, lat, lon, antenna_height_m: height } })
  const m = (await (await page.request.get('/sites')).text()).match(new RegExp(`/sites/(\\d+)">${name}<`))
  expect(m, `site ${name} listed`).not.toBeNull()
  return Number(m[1])
}

/** Fills the link form; the caller saves it. */
export async function fillLink(page, name, a, b) {
  await page.goto('/links/new')
  await page.locator('[name=name]').fill(name)
  await page.locator('[name=site_a_id]').selectOption({ label: a })
  await page.locator('[name=site_b_id]').selectOption({ label: b })
  await page.locator('[name=frequency_mhz]').fill('145.6')
  await page.locator('[name=tx_power_dbm]').fill('40')
  await page.locator('[name=rx_sensitivity_dbm]').fill('-120')
}

/** Saves a new coverage around the first site and computes it. */
export async function computeCoverage(page, name, rangeKm) {
  await page.goto('/coverages/new')
  await page.locator('[name=name]').fill(name)
  await page.locator('[name=frequency_mhz]').fill('433.5')
  await page.locator('[name=tx_power_dbm]').fill('30')
  await page.locator('[name=rx_height_m]').fill('2')
  await page.locator('[name=rx_sensitivity_dbm]').fill('-110')
  await page.locator('[name=range_km]').fill(String(rangeKm))
  await page.locator('[name=resolution_m]').fill('100')
  await c(page, 'button', 'Save').click()
  await expect.poll(() => pathname(page)).toMatch(/^\/coverages\/\d+$/)
  await c(page, 'button', 'Compute').click()
  await expect(page.locator('#job-badge')).toHaveText('done', { timeout: 120000 })
}

/** The first URL in the newest mail to an address. */
export async function mailLink(page, to) {
  const query = encodeURIComponent('to:' + to)
  let id
  await expect.poll(async () => {
    id = (await (await page.request.get(`${mailpit}/api/v1/search?query=${query}`)).json()).messages[0]?.ID
    return id
  }).toBeTruthy()
  const msg = await (await page.request.get(`${mailpit}/api/v1/message/${id}`)).json()
  return msg.Text.match(/https?:\/\/\S+/)[0]
}

export async function contentType(page, url) {
  const res = await page.request.get(url)
  expect(res.status(), url).toBe(200)
  return res.headers()['content-type']
}

export const status = async (page, url, opts = {}) => (await page.request.fetch(url, { maxRedirects: 0, ...opts })).status()
