import { test, expect, c, login, password, pathname, session, toast, userMenu } from '../helpers.js'

const user = 'auth'

test('sends a guest to the login page and back', async ({ page }) => {
  await page.goto('/sites')
  await expect.poll(() => pathname(page)).toBe('/login')
  expect(new URL(page.url()).search).toBe('?next=%2Fsites')
  await page.locator('[name=name]').fill(user)
  await page.locator('[name=password]').fill(password)
  await page.locator('[name=password]').press('Enter')
  await expect.poll(() => pathname(page)).toBe('/sites')
  await expect(page.locator('header').getByRole('button', { name: user })).toBeVisible()
})

test('rejects a wrong password', async ({ page }) => {
  await page.goto('/login')
  await page.locator('[name=name]').fill(user)
  await page.locator('[name=password]').fill('not-the-password')
  await page.locator('[name=password]').press('Enter')
  await expect(toast(page, 'Wrong user name or password.')).toBeVisible()
  expect(await session(page)).toBeUndefined()
})

test('logs out', async ({ page }) => {
  await login(page, user)
  await page.goto('/links')
  await userMenu(page)
  await page.getByRole('menuitem', { name: 'Log out' }).click()
  await expect.poll(() => pathname(page)).toBe('/login')
  await page.goto('/links')
  await expect.poll(() => pathname(page)).toBe('/login')
})
