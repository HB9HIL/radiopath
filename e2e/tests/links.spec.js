import { test, expect, c, login, addSite, fillLink, newPage, pathname, contentType } from '../helpers.js'

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
  await expect(c(page, '.alert-error li', 'Site A and B must differ')).toBeVisible()
})

test('lists, edits and deletes', async ({ page }) => {
  await page.goto('/links')
  await expect(c(page, 'tr', 'Alpha to Bravo')).toContainText('145.600 MHz')
  await c(c(page, 'tr', 'Alpha to Bravo'), 'a', 'Edit').click()
  await page.locator('[name=frequency_mhz]').fill('433.5')
  await c(page, 'button', 'Save').click()
  await expect.poll(() => pathname(page)).toMatch(/^\/links\/\d+$/)
  await page.goto('/links')
  await expect(c(page, 'tr', 'Alpha to Bravo')).toContainText('433.500 MHz')
  await c(c(page, 'tr', 'Alpha to Bravo'), 'button', 'Delete').click()
  await expect(page.getByText('No links yet').first()).toBeVisible()
})

test('keeps a used site', async ({ page }) => {
  await fillLink(page, 'Keep', 'Alpha', 'Bravo')
  await c(page, 'button', 'Save').click()
  await page.goto('/sites')
  await c(c(page, 'tr', 'Alpha'), 'button', 'Delete').click()
  await expect(c(page, '[role=alert]', 'Site is used by a link or coverage')).toBeVisible()
  await expect(c(page, '.table-note', '2 sites')).toBeVisible()
})
