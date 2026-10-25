import { test, expect, c, login, addSite, newPage, pathname, status } from '../helpers.js'

let siteID

test.beforeAll(async ({ browser }) => {
  const page = await newPage(browser)
  await login(page, 'owner-a')
  siteID = await addSite(page, 'Owner A site', 47.3, 9.3)
  await page.close()
})

test("hides one user's data from another", async ({ page }) => {
  await login(page, 'owner-b')
  await page.goto('/sites')
  await expect(page.getByText('No sites yet').first()).toBeVisible()
  await expect(page.getByText('Owner A site')).toHaveCount(0)
  expect(await status(page, `/api/sites/${siteID}`)).toBe(404)
  expect(await status(page, `/api/sites/${siteID}`, { method: 'DELETE' })).toBe(404)
  await page.goto('/')
  await expect.poll(() => pathname(page)).toBe('/sites')
  await page.goto('/links')
  await expect(c(page, 'button', 'New link')).toBeDisabled()
  await expect(page.getByText('Create your sites').first()).toBeVisible()
  await page.goto('/coverages')
  await expect(c(page, 'button', 'New coverage')).toBeDisabled()
  await expect(page.getByText('Create one').first()).toBeVisible()
  await page.goto('/links/new')
  await expect(page.getByText('Create at least two').first()).toBeVisible()

  await login(page, 'owner-a')
  await page.goto('/sites')
  await expect(c(page, 'tr', 'Owner A site')).toBeVisible()
})
