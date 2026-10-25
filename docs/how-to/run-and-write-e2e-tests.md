# Run and write end-to-end tests

## Run them

```sh
make e2e                   # builds the Linux binary and its image, runs the Playwright specs against it, cleans up
```

Every service runs in its own container from the compose profile `e2e`: the
image (`e2e-app`, with the tiles of `tmp/testfiles`, registration on and a
fixed admin token), PostGIS on a tmpfs, so each run starts with an empty
database, and Mailpit. Playwright runs in the official image (its version follows
`@playwright/test` in `e2e/package.json`) on the compose network and reaches the
others by service name, the same on Linux, macOS and in CI. `e2e/global-setup.js`
creates one user per spec file through the admin panel, so the files run in
parallel; `admin.spec.js` runs last, since its maintenance mode blocks all logins.
Map tiles are stubbed in the browser, nothing reaches the tile server.

The specs in `e2e/tests` cover login, sites, antennas, links, coverage jobs,
per-user ownership, sharing, the account page, the admin panel and the mail flows
(registration, confirmation, password reset). A failed run leaves screenshots,
traces and the app log in `e2e/test-results/`; open a `trace.zip` at
https://trace.playwright.dev. `make e2e` builds the Linux binary for the Docker
host and packs it with `docker/ci.Dockerfile`, like CI does; `make e2e-run` skips
the build and tests the binary already in `bin/`.

## Write one

1. Pick the spec file of the area you change in `e2e/tests/`, or add a new one. A new file needs its own user:
   add its name to `users` in `e2e/helpers.js`; `global-setup.js` creates it through the admin panel.
2. Log in without the form with `login(page, user)`; prepare data through the API where the UI is not what you
   test (`addSite(page, name, lat, lon)` posts to `/api/sites`).
3. Find elements the way a person would. Messages after an action are toasts: `toast(page, text)`. What describes
   the page itself (a shared view, an expired link) is an inline alert: `[role=alert]`. Row actions sit in a menu:
   `rowAction(page, rowText, 'Delete')`, and deleting asks first: `confirmDialog(page)`. The header's user menu
   opens with `userMenu(page)`. Dialogs are `page.getByRole('dialog')`, forms are found by their `aria-label`
   (`page.getByRole('form', { name: 'Describe an antenna' })`) or field names (`[name=frequency_mhz]`).
   `c(scope, selector, text)` finds the innermost element containing a text.
4. After a click that navigates inside the app, wait for the new page (for example
   `await expect(c(page, 'h1', 'Register')).toBeVisible()`) before filling in fields; otherwise the inputs of the
   old page can receive the text.
5. Tests inside a file run in order and may build on each other; there are no retries. Mails are read from
   Mailpit with `mailLink(page, address)`, map tiles are stubbed, the browser's own `confirm()` (only used by the
   admin panel) is accepted.
6. Phones: `e2e/tests/mobile.spec.js` runs as a Pixel 7 (`test.use({ ...devices['Pixel 7'] })`) and checks the tab
   bar, the row menu and that no page scrolls sideways; add a path there when you add a page.
7. Run `make e2e`. A failure leaves a screenshot, a trace and `app.log` in `e2e/test-results/`.
