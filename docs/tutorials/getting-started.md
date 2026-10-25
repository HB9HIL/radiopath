# Getting started with development

In this tutorial you set up Radiopath on your machine, plan a radio link in the running app, make a small change
to the code and check it the way CI does. It takes about half an hour. You need Go (the version in `go.mod`),
Docker and git; nothing else is installed on your machine.

## 1. Clone and start

```sh
git clone https://github.com/radiopath/radiopath.git
cd radiopath
make dev
```

The first start takes a few minutes: it creates `.env` from `.env.example`, clones the terrain and canopy tiles of
one area (N47E009, eastern Switzerland) from [radiopath/testfiles](https://github.com/radiopath/testfiles) into
`tmp/testfiles`, starts PostGIS, Valkey and Mailpit in Docker, installs the web app's packages in a Node container,
builds and starts the server and finally the Vite dev server. When it prints `App: http://localhost:5173`, the
stack is up. Leave this terminal running; Ctrl-C later stops everything.

## 2. Plan a link

1. Open http://localhost:5173 and log in as `dev` with the password `devpassword`.
2. On **Sites**, choose **New site**, click on the map near Säntis (47.25 N, 9.34 E), call it "Säntis" and save.
   Add a second site near Hoher Kasten (47.28 N, 9.48 E).
3. On **Links**, choose **New link**, pick the two sites, keep the defaults and save. The page shows the terrain
   profile, the path loss and the link margin.
4. On **Coverage**, create a coverage around Säntis with a range of 10 km and a resolution of 100 m, then press
   **Compute**. The badge goes from *queued* to *running* to *done*, and the raster appears on the map.

Everything you just did went through the JSON API. While you are logged in, http://localhost:5173/api/openapi.json
describes the same operations; load it into any OpenAPI viewer.

## 3. Make a change

Change the text of the empty site list. Open `web/src/routes/sites/+page.svelte`, find
`No sites yet.` and change it, for example to `Nothing here yet.` Save the file:
the page in the browser updates by itself, without a reload. Delete your sites to see the text.

Go changes need a restart: stop `make dev` with Ctrl-C and start it again.

## 4. Check it like CI

In a second terminal:

```sh
make check                       # svelte-check and the Prettier check of the web app
make format                      # fixes formatting if make check complained
go vet ./...
go tool staticcheck ./...
python3 scripts/funcdoc.py       # every function needs a doc comment
make test                        # Go unit tests
```

Your change breaks the end-to-end tests, which expect the old text: `make e2e` builds everything and runs the
Playwright specs in Docker, and `e2e/tests/sites.spec.js` fails. That is the safety net doing its job. Revert the
change with `git checkout web/src/routes/sites/+page.svelte`.

## Where to go next

- [Architecture](../explanation/architecture.md) explains how the parts you just used fit together.
- [Add an API operation](../how-to/add-an-api-operation.md) walks through a change across server and web app.
- [CONTRIBUTING.md](../../CONTRIBUTING.md) lists what a pull request needs.
