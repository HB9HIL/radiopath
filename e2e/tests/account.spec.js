import { test, expect, login, mailLink, password, pathname, status, toast } from '../helpers.js'

const user = 'account'
const newPassword = 'e2e-password-2'

const changePassword = async (page, current, next) => {
  await page.goto('/account')
  await page.locator('[name=current]').fill(current)
  await page.locator('[name=password]').fill(next)
  await page.locator('[name=password2]').fill(next)
  await page.locator('[name=password2]').press('Enter')
}

test('changes the password and logs the other device out', async ({ page, browser }) => {
  const other = await browser.newContext()
  await login(other, user)
  await login(page, user)
  await changePassword(page, password, newPassword)
  await expect(toast(page, 'Password changed. Other devices have been logged out.')).toBeVisible()
  await expect(page.locator('header').getByRole('button', { name: user })).toBeVisible()
  expect(await status(other, '/api/me')).toBe(401)
  await other.close()
  expect(await status(page, '/api/session', { method: 'POST', data: { name: user, password } })).toBe(401)
})

test('rejects a wrong current password', async ({ page }) => {
  await login(page, user, newPassword)
  await changePassword(page, password, 'e2e-password-3')
  await expect(toast(page, 'Current password is wrong')).toBeVisible()
})

test('adds an address by confirmation link', async ({ page }) => {
  await login(page, user, newPassword)
  await page.goto('/account')
  await expect(page.getByText('No address on file').first()).toBeVisible()
  await page.locator('[name=email]').fill('account@example.test')
  await page.locator('[name=email]').press('Enter')
  await expect(toast(page, 'Confirmation link sent to the new address')).toBeVisible()
  await page.goto(await mailLink(page, 'account@example.test'))
  await expect.poll(() => pathname(page)).toBe('/account')
  await expect(toast(page, 'Address confirmed.')).toBeVisible()
  await expect(page.locator('[name=email]')).toHaveValue('account@example.test')
})
