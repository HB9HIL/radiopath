# Contributing to Radiopath

Thanks for helping. This guide explains how to set up a development environment, how the code is organised and
what a change needs before it can be merged. README.md covers running and operating Radiopath.

Bug reports and feature ideas are welcome as GitHub issues. For larger changes, open an issue first so we can agree
on the approach before you spend time on it.

## Principles

- **Keep it simple.** The smallest change that solves the problem. No speculative abstractions, options or
  configuration "for later".
- **Stay focused.** A pull request does one thing. Don't reformat or refactor unrelated code in the same change;
  mention it in the PR instead.
- **Match the surroundings.** Follow the style of the code around your change. Go is formatted by gofmt,
  TypeScript and Svelte by Prettier.
- **English** for code, comments, commit messages, documentation and the user interface.

## Development environment

You need **Go** (the version in `go.mod`) and **Docker**. Everything else (PostgreSQL/PostGIS, Valkey, Mailpit,
Node, Playwright) runs in containers. Please don't install or run npm on the host: `web/node_modules` is built for
Linux inside `node:26-alpine` and shared with the containers.

```sh
make dev
```

starts the whole stack: PostGIS, Valkey and Mailpit in Docker, the Go server in the background and the Vite dev
server in the foreground. Open http://localhost:5173 and log in as `dev` / `devpassword`; mails end up in Mailpit at
http://localhost:8025, the admin token is in `.env` (created from `.env.example` on first start). Terrain and canopy
tiles for one area (N47E009, eastern Switzerland) are cloned from
[radiopath/testfiles](https://github.com/radiopath/testfiles) into `tmp/testfiles`. Ctrl-C stops everything.

| Command | What it does |
|---|---|
| `make dev` | local stack, app on http://localhost:5173 |
| `make test` | Go unit tests |
| `make vet` | `go vet ./...` |
| `make check` | svelte-check and the Prettier check of the web app |
| `make format` | formats TypeScript and Svelte with Prettier |
| `make api` | regenerates `web/openapi.json` and `web/src/lib/api.gen.ts` after an API change |
| `make web` | builds the web app into `internal/webui/dist` |
| `make build` | web app and a local binary in `bin/` |
| `make e2e` | builds everything and runs the Playwright end-to-end tests in Docker |
| `make docker` | builds the production image |
| `go tool staticcheck ./...` | static analysis |
| `python3 scripts/funcdoc.py` | checks that every function has a doc comment |

`go vet`, `go test` and `go build` need `internal/webui/dist` to exist; `make test` and `make vet` create an empty
placeholder when the web app has not been built.

## How the code is organised

```
cmd/radiopath/        main: configuration, server, workers, metrics; subcommands openapi, useradd, userdel, users
internal/
  web/                HTTP layer: JSON API (huma), auth, admin panel, reports, share links, job events
  webui/              embeds the built web app
  store/              PostgreSQL/PostGIS access, embedded migrations in store/migrations/
  worker/             background coverage jobs
  coverage/ link/     area coverage and point-to-point analysis
  itm/                Irregular Terrain Model (Longley-Rice) port
  dem/ s3/            terrain and canopy tiles from a directory or S3
  antenna/ plot/      antenna patterns, SVG plots, raster PNGs and legends
  report/ tiles/      PDF reports, map tile proxy
  mail/ metrics/ geo/ SMTP, Prometheus, geodesy
web/                  SvelteKit app (Svelte 5, TypeScript, Tailwind CSS 4, shadcn-svelte, TanStack Query, Leaflet)
e2e/                  Playwright end-to-end tests
docker/               compose stacks for development and e2e, Dockerfiles
scripts/              doc comment check, DEM and canopy sync scripts
```

One Go binary serves everything: the JSON API under `/api`, the web app (a static single-page app embedded in the
binary), a small server-rendered admin panel, map tiles, raster images and PDF reports. The web app talks to the
server only through the API, which is described by OpenAPI at `/api/openapi.json` with interactive docs at
`/api/docs`.

## Conventions

### Go

- Every function has a doc comment directly above it (`scripts/funcdoc.py`, enforced in CI). Comments explain why,
  not what.
- Data that belongs to a user is always queried with its owner, so someone else's id behaves like a missing one
  (404). Keep that property in every new store function.
- Database changes are new migrations (`internal/store/migrations/NNNN_name.sql`); never edit one that was released.
- staticcheck fails on unused code: remove what your change leaves unused.

### API

- Operations live next to their resource (`internal/web/sites.go`, `links.go`, `coverages.go`, ...) and are
  registered with `op(id, method, path, scope, summary)`. Operations without a scope are public; the test
  `TestAPISecurity` lists them, so adding a public operation is a deliberate change.
- Requests and responses use their own types with snake_case JSON keys. Never return store structs, they carry
  internal fields such as the owner. Empty lists are `[]`, not `null`.
- Validate input with the `check` helper, so errors come back as one RFC 7807 problem (422) with
  `errors[].location` pointing at the field.
- The API is not versioned: changes must be backwards compatible. Add fields, don't rename or remove them.
- After changing an operation or a type, run `make api` and commit `web/openapi.json` and
  `web/src/lib/api.gen.ts` with your change. CI fails when they are out of date.

### Web app

- Data is loaded with TanStack Query through `call()` from `src/lib/api.ts`, which throws an `ApiError` the forms
  show with `FormErrors`.
- Number fields are text inputs; the form reports values that are not numbers itself, range checks stay on the
  server.
- Every function in TypeScript and Svelte files has a doc comment as well (except the generated
  `src/lib/components/ui`).
- Run `make format` before you commit; `make check` and CI check the formatting.
- shadcn-svelte components are added with its CLI in the node container, see `web/components.json`. Check the diff
  of `web/package.json` afterwards: the CLI tends to add dependencies that are not needed. Dependency versions are
  pinned exactly.

### End-to-end tests

- Specs live in `e2e/tests/`. Each spec file gets its own user (created in `global-setup.js` through the admin
  panel), so files run in parallel without seeing each other's data; tests inside a file build on each other.
- Find alerts by `[role=alert]`, dialogs by `getByRole('dialog')`, forms by their `aria-label` or field names.
- After a click that navigates inside the app, wait for the new page (for example its heading) before filling in
  fields.
- A change to the user interface should come with a test that covers it.

## Before you open a pull request

1. `make test`, `make vet`, `go tool staticcheck ./...` and `python3 scripts/funcdoc.py` pass.
2. `make check` passes (run `make format` first).
3. `make api` leaves no diff if you touched the API.
4. `make e2e` passes if you changed behaviour that the end-to-end tests cover or should cover.
5. `CHANGELOG.md` has an entry for your change under `## [Unreleased]` (see below).
6. README.md is updated when configuration, behaviour or operation changes.

CI runs all of these on every pull request: formatting, vet, staticcheck, unit tests with the race detector,
govulncheck, the doc comment check, the API description check, svelte-check, Prettier and the end-to-end tests.

## Changelog

`CHANGELOG.md` is not an afterthought: it follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and its
section of a version becomes the notes of the GitHub release. Keep it current with every change.

- Every pull request that changes something a user, an operator or a contributor notices adds its lines under
  `## [Unreleased]` in the same pull request, not later. Pure refactorings and tooling count too, under
  Refactored and Chore.
- Use these sections, in this order, and add the heading when it is missing: **Added**, **Changed**, **Fixed**,
  **Removed**, **Refactored**, **Chore**.
- One line per change, written for the reader of the release notes: what changed and what it means for them,
  not which files were touched. Name configuration variables, routes or commands when they change, and say so
  when something behaves differently than before.
- End every line with your GitHub handle: `(by @your-handle)`. Dependency updates from Renovate are summed up as
  `Dependency Updates (by @renovate)`.
- Released sections are not edited (apart from typos). When releasing, the maintainer turns `[Unreleased]` into
  `## [X.Y.Z] - YYYY-MM-DD` and starts a new empty `[Unreleased]`.

```markdown
## [Unreleased]

### Added
- Coverage overlays can be exported as GeoTIFF from the coverage page (by @your-handle)

### Fixed
- The link report showed the RX gain in the TX row (by @your-handle)
```

## Commits and pull requests

- Commit messages are a single line in the
  [Conventional Commits](https://www.conventionalcommits.org/) form, for example
  `feat(web): show the measured level on the link page` or `fix(store): keep the legend when a coverage is copied`.
  Details belong in the pull request and the changelog.
- Keep pull requests small and focused; several small ones are easier to review than one large one.
- Releases are cut by the maintainer.

## License

Radiopath is licensed under AGPL-3.0-or-later (see `LICENSE`); a commercial license is offered as well (see
`COMMERCIAL-LICENSE.md`).
