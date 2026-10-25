import { test, expect, c, login, addSite, fillLink, newPage, pathname, contentType, toast, rowAction, confirmDialog } from '../helpers.js'

const user = 'links'

test.beforeAll(async ({ browser }) => {
  const page = await newPage(browser)
  await login(page, user)
  await addSite(page, 'Alpha', 47.3, 9.3)
  await addSite(page, 'Bravo', 47.4, 9.45)
  await page.close()
})
test.beforeEach(({ page }) => login(page, user))

test('analyses a link between two sites', async ({ page }) => {
  await fillLink(page, 'Alpha to Bravo', 'Alpha', 'Bravo')
  await c(page, 'button', 'Save').click()
  await expect.poll(() => pathname(page)).toMatch(/^\/links\/\d+$/)
  await expect(c(page, 'h1', 'Alpha to Bravo')).toBeVisible()
  await expect(c(page, 'div', 'Link margin, median')).toContainText(/[+-]\d+\.\d dB/)
  await expect(page.locator('.panel svg').first()).toBeVisible()
  const href = await c(page, 'a', 'Print report').getAttribute('href')
  expect(await contentType(page, href)).toBe('application/pdf')
})

test('refuses the same site on both ends', async ({ page }) => {
  await fillLink(page, 'Loop', 'Alpha', 'Alpha')
  await c(page, 'button', 'Save').click()
  await expect(toast(page, 'Site A and B must differ')).toBeVisible()
})

test('lists, edits and deletes', async ({ page }) => {
  await page.goto('/links')
  await expect(c(page, 'tr', 'Alpha to Bravo')).toContainText('145.600 MHz')
  await rowAction(page, 'Alpha to Bravo', 'Edit')
  await page.locator('[name=frequency_mhz]').fill('433.5')
  await c(page, 'button', 'Save').click()
  await expect.poll(() => pathname(page)).toMatch(/^\/links\/\d+$/)
  await page.goto('/links')
  await expect(c(page, 'tr', 'Alpha to Bravo')).toContainText('433.500 MHz')
  await rowAction(page, 'Alpha to Bravo', 'Delete')
  await confirmDialog(page)
  await expect(page.getByText('No links yet').first()).toBeVisible()
})

test('keeps a used site', async ({ page }) => {
  await fillLink(page, 'Keep', 'Alpha', 'Bravo')
  await c(page, 'button', 'Save').click()
  await page.goto('/sites')
  await rowAction(page, 'Alpha', 'Delete')
  await confirmDialog(page)
  await expect(toast(page, 'Site is used by a link or coverage')).toBeVisible()
  await expect(c(page, '.table-note', '2 sites')).toBeVisible()
})

// last in the file: afterwards the user has used up the minute
test('limits the analyses per user and minute', async ({ page }) => {
  const sites = await (await page.request.get('/api/sites')).json()
  const id = (name) => sites.find((s) => s.name === name).id
  const res = await page.request.post('/api/links', {
    data: { name: 'Busy', site_a_id: id('Alpha'), site_b_id: id('Bravo'), frequency_mhz: 145.6, polarization: 'vertical', tx_power_dbm: 40, rx_sensitivity_dbm: -120 },
  })
  expect(res.status()).toBe(201)
  const link = (await res.json()).id
  const codes = []
  for (let i = 0; i < 61; i++) codes.push((await page.request.get(`/api/links/${link}`)).status())
  expect(codes[0]).toBe(200)
  expect(codes).toContain(429)
  expect(new Set(codes)).toEqual(new Set([200, 429]))
})
