import { test, expect, c, login } from '../helpers.js'

test.beforeEach(({ page }) => login(page, 'sites'))

test('creates, edits, duplicates and deletes a site', async ({ page }) => {
  await page.goto('/sites')
  await expect(page.getByText('No sites yet').first()).toBeVisible()
  await c(page, 'a', 'New site').click()
  await page.locator('[name=name]').fill('Säntis')
  await page.locator('[name=lat]').fill('47.2495')
  await page.locator('[name=lon]').fill('9.3433')
  await page.locator('[name=antenna_height_m]').fill('12')
  await c(page, 'button', 'Save').click()
  await expect(page).toHaveURL(/\/sites$/)
  await expect(c(page, 'tr', 'Säntis')).toContainText('47.24950')
  await expect(c(page, 'tr', 'Säntis')).toContainText('12.0 m')

  await c(c(page, 'tr', 'Säntis'), 'a', 'Edit').click()
  await page.locator('[name=antenna_height_m]').fill('20')
  await c(page, 'button', 'Save').click()
  await expect(c(page, 'tr', 'Säntis')).toContainText('20.0 m')

  await c(c(page, 'tr', 'Säntis'), 'a', 'Duplicate').click()
  await expect(page.locator('[name=name]')).toHaveValue('Säntis (copy)')
  await expect(page.locator('[name=lat]')).toHaveValue('47.2495')
  await c(page, 'button', 'Save').click()
  await expect(c(page, '.table-note', '2 sites')).toBeVisible()

  await c(c(page, 'tr', 'Säntis (copy)'), 'button', 'Delete').click()
  await expect(c(page, '.table-note', '1 site')).toBeVisible()
  await expect(page.getByText('Säntis (copy)')).toHaveCount(0)
})

test('re-renders the form with the errors', async ({ page }) => {
  await page.goto('/sites/new')
  await page.locator('[name=name]').fill('Bad')
  await page.locator('[name=lat]').fill('95')
  await page.locator('[name=lon]').fill('abc')
  await c(page, 'button', 'Save').click()
  await expect(c(page, '.alert-error li', 'Latitude: must be between -90 and 90')).toBeVisible()
  await expect(c(page, '.alert-error li', 'Longitude: not a number')).toBeVisible()
  await expect(page.locator('[name=name]')).toHaveValue('Bad')
})
