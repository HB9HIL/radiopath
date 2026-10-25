import { test, expect, c, login, mailLink, mailpit, pathname, session } from '../helpers.js'

const submit = async (page, field) => page.locator(`[name=${field}]`).press('Enter')

test('registers, confirms the address and logs in', async ({ page }) => {
  await page.goto('/login')
  await c(page, 'a', 'Create an account').click()
  await page.locator('[name=name]').fill('dave')
  await page.locator('[name=email]').fill('dave@example.test')
  await page.locator('[name=password]').fill('dave-password-1')
  await page.locator('[name=password2]').fill('dave-password-1')
  await submit(page, 'password2')
  await expect.poll(() => pathname(page)).toBe('/login')
  await expect(c(page, '.alert-info', 'Almost done')).toBeVisible()

  await page.locator('[name=name]').fill('dave')
  await page.locator('[name=password]').fill('dave-password-1')
  await submit(page, 'password')
  await expect(c(page, '.alert-info', 'not confirmed yet')).toBeVisible()
  expect(await session(page)).toBeUndefined()
  await c(page, 'button', 'Resend confirmation mail').click()
  await expect(c(page, '.alert-info', 'a new link is on its way')).toBeVisible()

  await page.goto(await mailLink(page, 'dave@example.test'))
  await expect.poll(() => pathname(page)).toBe('/login')
  await expect(c(page, '.alert-info', 'Address confirmed.')).toBeVisible()
  await page.locator('[name=name]').fill('dave')
  await page.locator('[name=password]').fill('dave-password-1')
  await submit(page, 'password')
  await expect.poll(() => pathname(page)).toBe('/sites')

  await page.goto(await mailLink(page, 'dave@example.test'))
  await expect(c(page, '.alert-error', 'expired or was already used')).toBeVisible()
})

test('resets a forgotten password', async ({ page }) => {
  await page.goto('/login')
  await c(page, 'a', 'Forgot password?').click()
  await page.locator('[name=email]').fill('dave@example.test')
  await submit(page, 'email')
  await expect(c(page, '.alert-info', 'a reset link is on its way')).toBeVisible()
  await page.goto(await mailLink(page, 'dave@example.test'))
  await expect(c(page, 'h1', 'Choose a new password')).toBeVisible()
  await page.locator('[name=password]').fill('dave-password-2')
  await page.locator('[name=password2]').fill('dave-password-2')
  await submit(page, 'password2')
  await expect.poll(() => pathname(page)).toBe('/login')
  await expect(c(page, '.alert-info', 'Password changed. Log in with the new one.')).toBeVisible()
  await login(page, 'dave', 'dave-password-2')

  await page.goto(await mailLink(page, 'dave@example.test'))
  await expect(c(page, '.alert-error', 'expired or was already used')).toBeVisible()
  await expect(page.locator('[name=password]')).toHaveCount(0)
})

test('answers the same for an unknown address', async ({ page }) => {
  await page.goto('/forgot')
  await page.locator('[name=email]').fill('nobody@example.test')
  await submit(page, 'email')
  await expect(c(page, '.alert-info', 'a reset link is on its way')).toBeVisible()
  const res = await page.request.get(`${mailpit}/api/v1/search?query=${encodeURIComponent('to:nobody@example.test')}`)
  expect((await res.json()).messages).toHaveLength(0)
})
