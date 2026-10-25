import { test, expect, c, login, toast, rowAction, confirmDialog } from '../helpers.js'

test.beforeEach(({ page }) => login(page, 'antennas'))

test('imports an MSI file', async ({ page }) => {
  await page.goto('/antennas')
  await expect(page.getByText('No antennas yet').first()).toBeVisible()
  await page.locator('input[type=file]').setInputFiles('fixtures/yagi.msi')
  await c(page, 'button', 'Import').click()
  const row = c(page, 'tr', 'E2E Yagi')
  await expect(row).toContainText('12.0 dBi')
  await expect(row).toContainText('145 MHz')
  await expect(row.locator('.pattern-svg')).toBeVisible()
})

test('rejects a file without a pattern', async ({ page }) => {
  await page.goto('/antennas')
  await page.locator('input[type=file]').setInputFiles({ name: 'bad.msi', mimeType: 'application/octet-stream', buffer: Buffer.from('NAME x\nGAIN 3 dBd\n') })
  await c(page, 'button', 'Import').click()
  await expect(toast(page, 'no HORIZONTAL pattern')).toBeVisible()
  await expect(c(page, '.table-note', '1 antenna')).toBeVisible()
})

test('builds a sector, edits and deletes it', async ({ page }) => {
  await page.goto('/antennas')
  const manual = page.getByRole('form', { name: 'Describe an antenna' })
  await manual.locator('[name=name]').fill('Sector 65')
  await manual.locator('[name=gain_dbi]').fill('15')
  await c(manual, 'button', 'Create').click()
  await expect(c(page, 'tr', 'Sector 65')).toContainText('15.0 dBi')
  await expect(c(page, 'tr', 'Sector 65')).toContainText('25.0 dB')
  // imported patterns cannot be edited
  await c(page, 'tr', 'E2E Yagi').getByRole('button', { name: 'Actions' }).click()
  await expect(page.getByRole('menuitem', { name: 'Edit' })).toHaveCount(0)
  await page.keyboard.press('Escape')

  await rowAction(page, 'Sector 65', 'Edit')
  const dialog = page.getByRole('dialog')
  await expect(dialog).toHaveCount(1)
  await expect(dialog.locator('[name=beamwidth_deg]')).toHaveValue('65')
  await expect(dialog.locator('[name=front_back_db]')).toHaveValue('25')
  await dialog.locator('[name=name]').fill('Sector 90')
  await dialog.locator('[name=gain_dbi]').fill('abc')
  await c(dialog, 'button', 'Save').click()

  await expect(toast(page, 'Gain: not a number')).toBeVisible()
  await expect(dialog.locator('[name=name]')).toHaveValue('Sector 90')
  await dialog.locator('[name=gain_dbi]').fill('17')
  await dialog.locator('[name=beamwidth_deg]').fill('90')
  await dialog.locator('[name=front_back_db]').fill('30')
  await c(dialog, 'button', 'Save').click()
  await expect(dialog).toHaveCount(0)
  await expect(page.getByText('Sector 65')).toHaveCount(0)
  await expect(c(page, 'tr', 'Sector 90')).toContainText('17.0 dBi')
  await expect(c(page, 'tr', 'Sector 90')).toContainText('30.0 dB')

  await rowAction(page, 'Sector 90', 'Edit')
  await expect(dialog.locator('[name=beamwidth_deg]')).toHaveValue('90')
  await c(dialog, 'button', 'Close').click()
  await expect(dialog).toHaveCount(0)

  await rowAction(page, 'Sector 90', 'Delete')
  await confirmDialog(page)
  await expect(page.getByText('Sector 90')).toHaveCount(0)
  await expect(c(page, '.table-note', '1 antenna')).toBeVisible()
})
