import { test, expect, baseURL, c, login, password, pathname, session, toast, userMenu } from '../helpers.js'

const user = 'auth'

test('sends a guest from / to the login, the only auth page for search engines', async ({ page }) => {
  await page.goto('/')
  await expect.poll(() => pathname(page)).toBe('/login')
  await expect(c(page, 'h1', 'Log in')).toBeVisible()
  await expect(page.locator('meta[name=robots]')).toHaveCount(0)
  await c(page, 'a', 'Create an account').click()
  await expect(c(page, 'h1', 'Register')).toBeVisible()
  await expect(page.locator('meta[name=robots]')).toHaveAttribute('content', 'noindex')
})

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

test('stays on the site whatever next says', async ({ page }) => {
  // browsers drop the tab and read a backslash as /, so each of these would mean //evil.example
  for (const next of ['//evil.example', '/\\evil.example', '/\t/evil.example', 'https://evil.example/']) {
    await page.context().clearCookies()
    await page.goto('/login?next=' + encodeURIComponent(next))
    await page.locator('[name=name]').fill(user)
    await page.locator('[name=password]').fill(password)
    await page.locator('[name=password]').press('Enter')
    // the start page sends a logged-in user on to a list; wait for it, so the next goto is not cut off
    await expect.poll(() => pathname(page), next).toMatch(/^\/(sites|links)$/)
    expect(new URL(page.url()).host, next).toBe(new URL(baseURL).host)
  }
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
