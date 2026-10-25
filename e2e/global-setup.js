import { request, expect } from '@playwright/test'
import { baseURL, adminToken, password, users } from './helpers.js'

export default async function globalSetup() {
  const api = await request.newContext({ baseURL })
  await expect.poll(async () => (await api.get('/healthz').catch(() => null))?.ok() ?? false, { timeout: 60000 }).toBe(true)
  // a single admin login for all users: the admin panel allows 10 logins per IP and 15 minutes
  expect((await api.post('/admin/login', { form: { token: adminToken }, maxRedirects: 0 })).status()).toBe(303)
  for (const name of users) {
    expect((await api.post('/admin/users', { form: { name, password }, maxRedirects: 0 })).status(), `create ${name}`).toBe(303)
  }
  await api.dispose()
}
