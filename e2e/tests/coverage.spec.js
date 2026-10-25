import { test, expect, c, login, addSite, newPage, computeCoverage, contentType, status, toast } from '../helpers.js'

const user = 'coverage'

test.describe.configure({ timeout: 180000 })

test.beforeAll(async ({ browser }) => {
  const page = await newPage(browser)
  await login(page, user)
  await addSite(page, 'Tower', 47.35, 9.4, 20)
  await page.close()
})
test.beforeEach(({ page }) => login(page, user))

test('computes a coverage in the background', async ({ page }) => {
  await page.goto('/coverages/new')
  await page.locator('[name=name]').fill('Tower 2 km')
  await page.locator('[name=frequency_mhz]').fill('433.5')
  await page.locator('[name=tx_power_dbm]').fill('30')
  await page.locator('[name=rx_height_m]').fill('2')
  await page.locator('[name=rx_sensitivity_dbm]').fill('-110')
  await page.locator('[name=range_km]').fill('2')
  await page.locator('[name=resolution_m]').fill('100')
  await c(page, 'button', 'Save').click()
  await expect(page).toHaveURL(/\/coverages\/\d+$/)
  await expect(page.locator('#job-badge')).toHaveText('idle')
  await expect(page.getByText('Not computed yet').first()).toBeVisible()

  await c(page, 'button', 'Compute').click()
  await expect(page.locator('#job-badge')).toHaveText('done', { timeout: 120000 })
  await expect(c(page, 'h2', 'Legend')).toBeVisible()
  await expect(c(page, 'button', 'Recompute')).toBeVisible()
  const p = new URL(page.url()).pathname
  expect(await contentType(page, `${p}/image.png`)).toBe('image/png')
  const res = await page.request.get(`${p}/composite.png?with=`)
  expect(res.headers()['content-type']).toBe('image/png')
  expect(res.headers()['x-raster-bounds']).toMatch(/^-?[\d.]+,-?[\d.]+,-?[\d.]+,-?[\d.]+$/)
  expect(await status(page, `${p}/composite.png?with=999999`)).toBe(404)
  expect(await contentType(page, `${p}/report.pdf`)).toBe('application/pdf')
  await page.goto('/coverages')
  await expect(c(page, 'tr', 'Tower 2 km')).toContainText('2.0 km')
})

test('remembers the map view', async ({ page }) => {
  await page.goto('/coverages')
  await c(c(page, 'tr', 'Tower 2 km'), 'a', 'Tower 2 km').click()
  await expect(page.locator('#opacity')).toHaveValue('60')
  await expect(page.locator('#circles')).toBeChecked()
  const p = new URL(page.url()).pathname
  const view = p.replace('/coverages/', '/api/coverages/') + '/view'
  expect(await status(page, view, { method: 'PUT', data: { opacity: 35, overlays: [], circles: false } })).toBe(204)
  await page.reload()
  await expect(page.locator('#opacity')).toHaveValue('35')
  await expect(page.getByText('35 %').first()).toBeVisible()
  await expect(page.locator('#circles')).not.toBeChecked()
})

test('merges a ticked overlay into one raster', async ({ page }) => {
  await computeCoverage(page, 'Tower 1 km', 1)
  const layers = page.locator('img.leaflet-image-layer')
  await expect(page.locator('img.leaflet-image-layer[src*="/image.png?t="]')).toHaveCount(1)
  await expect(page.locator('input[data-overlay]')).toHaveCount(1)
  await page.locator('input[data-overlay]').check()
  await expect(page.locator('img.leaflet-image-layer[src^="blob:"]')).toHaveCount(1, { timeout: 10000 })
  await expect(layers).toHaveCount(1)
  await page.locator('input[data-overlay]').uncheck()
  await expect(page.locator('img.leaflet-image-layer[src*="/image.png?t="]')).toHaveCount(1)
  await expect(layers).toHaveCount(1)
})

test('edits the legend without recomputing', async ({ page }) => {
  await page.goto('/coverages')
  await c(page, 'a', 'Tower 2 km').click()
  await expect(c(page, 'td', '>= 30 dB')).toBeVisible()
  await expect(c(page, 'td', 'below 0 dB / no data')).toBeVisible()
  const legend = page.locator('#legend')
  await c(page, 'button', 'Edit legend').click()
  await expect(legend).toBeVisible()
  await legend.locator('[name=min_1]').fill('40')
  await legend.locator('[name=color_1]').fill('#ff0000')
  await legend.locator('[name=min_4]').fill('')
  await c(legend, 'button', 'Save').click()
  await expect(c(page, 'td', '>= 40 dB')).toBeVisible()
  await expect(c(page, 'td', '20 to 40 dB')).toBeVisible()
  await expect(c(page, 'td', 'below 10 dB / no data')).toBeVisible()
  // the browser normalises the inline colour, so compare the rendered one
  await expect(c(page, 'tr', '>= 40 dB').locator('.swatch')).toHaveCSS('background-color', 'rgb(255, 0, 0)')
  await expect(page.locator('#job-badge')).toHaveText('done')
  const p = new URL(page.url()).pathname
  expect(await contentType(page, `${p}/image.png`)).toBe('image/png')
  await c(page, 'a', 'Duplicate').click()
  await expect(c(page, 'legend', 'Legend, copied along')).toBeVisible()
  await expect(c(page, 'td', '>= 40 dB')).toBeVisible()
  await expect(c(page, 'td', '20 to 40 dB')).toBeVisible()
  await page.goBack()
  await c(page, 'button', 'Edit legend').click()
  await legend.locator('[name=min_2]').fill('40')
  await c(legend, 'button', 'Save').click()
  await expect(toast(page, 'listed twice')).toBeVisible()
  await expect(c(page, 'td', '>= 40 dB')).toBeVisible()
  await c(legend, 'button', 'Reset to defaults').click()
  await expect(c(page, 'td', '>= 30 dB')).toBeVisible()
  await expect(c(page, 'td', 'below 0 dB / no data')).toBeVisible()
})

test('refuses too many pixels', async ({ page }) => {
  await page.goto('/coverages/new')
  await page.locator('[name=name]').fill('Too fine')
  await page.locator('[name=range_km]').fill('300')
  await page.locator('[name=resolution_m]').fill('10')
  await c(page, 'button', 'Save').click()
  await expect(toast(page, 'Range / resolution exceeds 1000 pixels per radius')).toBeVisible()
})
