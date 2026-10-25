# Contributing to Radiopath

Thanks for helping. This guide lists the rules for a change: how to work, what to check and how to document it.
The details live in the [documentation](docs/README.md), starting with the
[development tutorial](docs/tutorials/getting-started.md).

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
make dev    # http://localhost:5173, login dev / devpassword; Ctrl-C stops everything
```

The [development tutorial](docs/tutorials/getting-started.md) walks through the first start and a first change;
[Development commands and CI](docs/reference/development.md) lists every make target and check, and
[Architecture](docs/explanation/architecture.md) explains how the code is organised.

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
- Step by step: [Add an API operation](docs/how-to/add-an-api-operation.md).
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

- A change to the user interface comes with an end-to-end test that covers it, and works on a phone:
  `e2e/tests/mobile.spec.js` checks that no page scrolls sideways.
- Each spec file has its own user; tests inside a file build on each other. How to write and run them:
  [Run and write end-to-end tests](docs/how-to/run-and-write-e2e-tests.md).

## Before you open a pull request

1. `make test`, `make vet`, `go tool staticcheck ./...` and `python3 scripts/funcdoc.py` pass.
2. `make check` passes (run `make format` first).
3. `make api` leaves no diff if you touched the API.
4. `make e2e` passes if you changed behaviour that the end-to-end tests cover or should cover.
5. `CHANGELOG.md` has an entry for your change under `## [Unreleased]` (see below).
6. The documentation in `docs/` is updated in the same pull request when configuration, behaviour, operation or
   the way of working changes (see below).

CI runs all of these on every pull request: formatting, vet, staticcheck, unit tests with the race detector,
govulncheck, the doc comment check, the API description check, svelte-check, Prettier and the end-to-end tests.
CodeQL analyses every pull request as well; look at its findings before asking for a review.

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

## Documentation

Documentation lives in `docs/` and follows the [Diátaxis](https://diataxis.fr) framework. Every page is one of
four kinds, chosen by what the reader needs, not by its topic:

| Kind | Directory | The reader wants to | Example |
|---|---|---|---|
| Tutorial | `docs/tutorials/` | learn by doing, guided from start to finish | [Getting started with development](docs/tutorials/getting-started.md) |
| How-to guide | `docs/how-to/` | reach a specific goal, knowing the basics | [Add an API operation](docs/how-to/add-an-api-operation.md) |
| Reference | `docs/reference/` | look up exact facts | [Configuration](docs/reference/configuration.md) |
| Explanation | `docs/explanation/` | understand why it works this way | [Architecture](docs/explanation/architecture.md) |

- Keep the kinds apart: a how-to does not stop to explain design decisions, a reference does not teach. Link to
  the other kinds instead.
- Update the documentation in the same pull request as the change it describes, and add new pages to the index in
  [docs/README.md](docs/README.md).
- `README.md` stays a short entry page: what Radiopath is, how to run it, where the documentation is.
- Write in English, in short sentences; name variables, routes and commands exactly as they are in the code.

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
