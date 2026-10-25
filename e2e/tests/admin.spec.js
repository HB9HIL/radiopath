import { test, expect, c, adminToken, login, status } from '../helpers.js'

const userCount = async (page) => Number((await page.locator('.table-note').innerText()).match(/\d+/)[0])

test('needs the token', async ({ page }) => {
  await page.goto('/admin')
  await expect(c(page, 'h1', 'Admin')).toBeVisible()
  await page.locator('[name=token]').fill('wrong-token-wrong-token-wrong-token')
  await page.locator('[name=token]').press('Enter')
  await expect(c(page, '.alert-error', 'Wrong token.')).toBeVisible()
  expect(await status(page, '/admin/users/admin/password')).not.toBe(200)
})

test('creates a user, blocks logins, deletes the user', async ({ page }) => {
  await page.goto('/admin')
  await page.locator('[name=token]').fill(adminToken)
  await page.locator('[name=token]').press('Enter')
  await expect(c(page, 'h1', 'Users')).toBeVisible()
  await expect(c(page, 'tr', 'admin')).toBeVisible()
  const before = await userCount(page)

  const create = page.locator('form[action="/admin/users"]')
  await create.locator('[name=name]').fill('carol')
  await create.locator('[name=password]').fill('carol-password-1')
  await c(create, 'button', 'Create').click()
  await expect(c(page, 'tr', 'carol')).toBeVisible()
  await expect(page.locator('.table-note')).toHaveText(`${before + 1} users`)

  await c(page, 'button', 'Maintenance mode').click()
  await expect(c(page, '.badge', 'maintenance')).toBeVisible()
  expect(await status(page, '/api/session', { method: 'POST', data: { name: 'carol', password: 'carol-password-1' } })).toBe(503)
  await c(page, 'button', 'Allow logins again').click()
  await expect(c(page, '.badge', 'open')).toBeVisible()
  await login(page, 'carol', 'carol-password-1')

  await page.goto('/admin')
  await c(c(page, 'tr', 'carol'), 'button', 'Delete').click()
  await expect(page.locator('.table-note')).toHaveText(`${before} users`)
  await expect(c(page, 'tr', 'carol')).toHaveCount(0)
})
