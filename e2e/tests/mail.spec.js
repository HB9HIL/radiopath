import { test, expect, c, login, mailLink, mailpit, pathname, session, toast } from '../helpers.js'

const submit = async (page, field) => page.locator(`[name=${field}]`).press('Enter')

test('registers, confirms the address and logs in', async ({ page }) => {
  await page.goto('/login')
  await c(page, 'a', 'Create an account').click()
  // client-side navigation: fill only once the register page replaced the login page
  await expect(c(page, 'h1', 'Register')).toBeVisible()
  await page.locator('[name=name]').fill('dave')
  await page.locator('[name=email]').fill('dave@example.test')
  await page.locator('[name=password]').fill('dave-password-1')
  await page.locator('[name=password2]').fill('dave-password-1')
  await submit(page, 'password2')
  await expect.poll(() => pathname(page)).toBe('/login')
  await expect(toast(page, 'Almost done')).toBeVisible()

  await page.locator('[name=name]').fill('dave')
  await page.locator('[name=password]').fill('dave-password-1')
  await submit(page, 'password')
  await expect(c(page, '[role=alert]', 'not confirmed yet')).toBeVisible()
  expect(await session(page)).toBeUndefined()
  await c(page, 'button', 'Resend confirmation mail').click()
  await expect(toast(page, 'a new link is on its way')).toBeVisible()

  // the token travels in the fragment, so it never reaches a server log; the page drops it from the address bar
  const confirm = await mailLink(page, 'dave@example.test', 'Confirm')
  expect(confirm).toMatch(/\/verify#[\w-]{43}$/)
  await page.goto(confirm)
  await expect.poll(() => pathname(page)).toBe('/login')
  await expect(toast(page, 'Address confirmed.')).toBeVisible()
  await page.locator('[name=name]').fill('dave')
  await page.locator('[name=password]').fill('dave-password-1')
  await submit(page, 'password')
  await expect.poll(() => pathname(page)).toBe('/sites')

  await page.goto(confirm)
  await expect(c(page, '[role=alert]', 'expired or was already used')).toBeVisible()
})

test('resets a forgotten password', async ({ page }) => {
  await page.goto('/login')
  await c(page, 'a', 'Forgot password?').click()
  await expect(c(page, 'h1', 'Forgot password')).toBeVisible()
  await page.locator('[name=email]').fill('dave@example.test')
  await submit(page, 'email')
  await expect(toast(page, 'a reset link is on its way')).toBeVisible()
  const reset = await mailLink(page, 'dave@example.test', 'Reset')
  expect(reset).toMatch(/\/reset#[\w-]{43}$/)
  await page.goto(reset)
  await expect(c(page, 'h1', 'Choose a new password')).toBeVisible()
  expect(page.url()).not.toContain('#')
  await page.locator('[name=password]').fill('dave-password-2')
  await page.locator('[name=password2]').fill('dave-password-2')
  await submit(page, 'password2')
  await expect.poll(() => pathname(page)).toBe('/login')
  await expect(toast(page, 'Password changed. Log in with the new one.')).toBeVisible()
  await login(page, 'dave', 'dave-password-2')

  await page.goto(reset)
  await expect(c(page, '[role=alert]', 'expired or was already used')).toBeVisible()
  await expect(page.locator('[name=password]')).toHaveCount(0)
})

test('logs in with the confirmed address, and fails with it like with a name', async ({ page }) => {
  const attempt = async (name, pw) => {
    await page.goto('/login')
    await page.locator('[name=name]').fill(name)
    await page.locator('[name=password]').fill(pw)
    await submit(page, 'password')
  }
  for (const [name, pw] of [['dave@example.test', 'not-the-password'], ['nobody@example.test', 'dave-password-2']]) {
    await attempt(name, pw)
    await expect(toast(page, 'Wrong user name or password.')).toBeVisible()
    expect(await session(page)).toBeUndefined()
  }
  await attempt('Dave@Example.test', 'dave-password-2')
  await expect.poll(() => pathname(page)).toBe('/sites')
})

test('answers a taken address or name like a new account and tells only the mailbox', async ({ page }) => {
  const register = async (name, email) => {
    await page.goto('/register')
    await page.locator('[name=name]').fill(name)
    await page.locator('[name=email]').fill(email)
    await page.locator('[name=password]').fill('some-password-1')
    await page.locator('[name=password2]').fill('some-password-1')
    await submit(page, 'password2')
    await expect.poll(() => pathname(page)).toBe('/login')
    await expect(toast(page, 'Almost done')).toBeVisible()
  }
  await register('dave2', 'dave@example.test')
  expect(await mailLink(page, 'dave@example.test', 'Your Radiopath account')).toMatch(/\/forgot$/)
  await register('dave', 'eve@example.test')
  expect(await mailLink(page, 'eve@example.test', 'Choose another')).toMatch(/\/register$/)
})

test('answers the same for an unknown address', async ({ page }) => {
  await page.goto('/forgot')
  await page.locator('[name=email]').fill('nobody@example.test')
  await submit(page, 'email')
  await expect(toast(page, 'a reset link is on its way')).toBeVisible()
  const res = await page.request.get(`${mailpit}/api/v1/search?query=${encodeURIComponent('to:nobody@example.test')}`)
  expect((await res.json()).messages).toHaveLength(0)
})
