import { test, expect, c, login, password, pathname, session } from '../helpers.js'

const user = 'auth'

test('sends a guest to the login page and back', async ({ page }) => {
  await page.goto('/sites')
  await expect.poll(() => pathname(page)).toBe('/login')
  expect(new URL(page.url()).search).toBe('?next=%2Fsites')
  await page.locator('[name=name]').fill(user)
  await page.locator('[name=password]').fill(password)
  await page.locator('[name=password]').press('Enter')
  await expect.poll(() => pathname(page)).toBe('/sites')
  await expect(c(page.locator('header'), 'a', user)).toBeVisible()
})

test('rejects a wrong password', async ({ page }) => {
  await page.goto('/login')
  await page.locator('[name=name]').fill(user)
  await page.locator('[name=password]').fill('not-the-password')
  await page.locator('[name=password]').press('Enter')
  await expect(c(page, '[role=alert]', 'Wrong user name or password.')).toBeVisible()
  expect(await session(page)).toBeUndefined()
})

test('logs out', async ({ page }) => {
  await login(page, user)
  await page.goto('/links')
  await c(page, 'button', 'Logout').click()
  await expect.poll(() => pathname(page)).toBe('/login')
  await page.goto('/links')
  await expect.poll(() => pathname(page)).toBe('/login')
})
