import { test, expect, c, login, addSite, fillLink, newPage, computeCoverage, contentType, pathname, status } from '../helpers.js'

const user = 'share'

test.describe.configure({ timeout: 180000 })

test.beforeAll(async ({ browser }) => {
  const page = await newPage(browser)
  await login(page, user)
  await addSite(page, 'Alpha', 47.3, 9.3)
  await addSite(page, 'Bravo', 47.4, 9.45)
  await fillLink(page, 'Shared link', 'Alpha', 'Bravo')
  await c(page, 'button', 'Save').click()
  await expect.poll(() => pathname(page)).toMatch(/^\/links\/\d+$/)
  await page.close()
})
test.beforeEach(({ page }) => login(page, user))

const sharePath = async (page) => (await page.locator('input[readonly]').inputValue()).replace(/^https?:\/\/[^/]+/, '')
const dropSession = (page) => page.context().clearCookies({ name: 'radiopath_session' })
// exact: the user of this spec is called share, and the user menu in the header carries that name
const shareButton = (page) => page.getByRole('button', { name: 'Share', exact: true })

test('shows a link to a visitor without a session', async ({ page }) => {
  await page.goto('/links')
  await c(c(page, 'tr', 'Shared link'), 'a', 'Shared link').click()
  await expect(c(page, 'button', 'Create public link')).not.toBeVisible()
  await shareButton(page).click()
  await c(page, 'button', 'Create public link').click()
  await expect(shareButton(page)).toHaveClass(/btn-shared/)
  await expect(page.locator('input[readonly]')).toBeVisible()
  const path = await sharePath(page)
  expect(path).toMatch(/^\/s\/l\/\S+$/)

  await dropSession(page)
  await page.goto(path)
  await expect(c(page, 'h1', 'Shared link')).toBeVisible()
  await expect(c(page, '[role=alert]', 'Shared read-only view')).toBeVisible()
  await expect(c(page, 'div', 'Link margin, median')).toContainText(/[+-]\d+\.\d dB/)
  await expect(page.locator('.panel svg').first()).toBeVisible()
  await expect(page.locator('body')).not.toContainText(user)
  await expect(c(page, 'a', 'Edit')).toHaveCount(0)
  await expect(c(page, 'a', 'Back')).toHaveCount(0)
  expect(await contentType(page, `${path}/report.pdf`)).toBe('application/pdf')
})

test('revokes a link share', async ({ page }) => {
  await page.goto('/links')
  await c(c(page, 'tr', 'Shared link'), 'a', 'Shared link').click()
  await shareButton(page).click()
  const path = await sharePath(page)
  await c(page, 'button', 'Revoke').click()
  await expect(c(page, 'button', 'Create public link')).toBeVisible()
  await expect(shareButton(page)).not.toHaveClass(/btn-shared/)

  await dropSession(page)
  expect(await status(page, path.replace('/s/l/', '/api/share/links/'))).toBe(404)
  expect(await status(page, `${path}/report.pdf`)).toBe(404)
  await page.goto(path)
  await expect(c(page, 'h1', 'Not found')).toBeVisible()
})

test('shows a computed coverage to a visitor', async ({ page }) => {
  await computeCoverage(page, 'Shared coverage', 2)
  await shareButton(page).click()
  await c(page, 'button', 'Create public link').click()
  await expect(page.locator('input[readonly]')).toBeVisible()
  const path = await sharePath(page)
  expect(path).toMatch(/^\/s\/c\/\S+$/)

  await dropSession(page)
  await page.goto(path)
  await expect(c(page, 'h1', 'Shared coverage')).toBeVisible()
  await expect(c(page, 'h2', 'Legend')).toBeVisible()
  await expect(page.locator('#job-badge')).toHaveCount(0)
  await expect(c(page, 'button', 'Recompute')).toHaveCount(0)
  expect(await contentType(page, `${path}/image.png`)).toBe('image/png')
  expect(await contentType(page, `${path}/composite.png?with=`)).toBe('image/png')
  expect(await status(page, `${path}/composite.png?with=999999`)).toBe(404)
  expect(await contentType(page, `${path}/report.pdf`)).toBe('application/pdf')
})
