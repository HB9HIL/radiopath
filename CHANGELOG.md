# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.2.7] - 2026-10-25

### Changed
- `make dev` starts the local stack (PostgreSQL, Valkey, Mailpit, stylesheet watcher) in Docker and the server in the foreground, with a dev user and terrain from radiopath/testfiles; `make run`, `make db-up` and `make db-down` are gone. Node only runs in Docker, the host needs Go and Docker (by @HB9HIL)
- One CI workflow instead of two: the binaries are built once and reused for e2e, the image and the release; new checks staticcheck, govulncheck and a doc comment on every function. Images are also tagged with the commit (`:<sha>`) (by @HB9HIL)

### Fixed
- Data race on the map tile cache key prefix under concurrent tile requests; the per-IP tile limit now always uses the same key (by @HB9HIL)

### Refactored
- Using Playwright for e2e testing instead of Cypress, every service in its own container, on real terrain and canopy (by @HB9HIL)

### Chore
- Update to Go 1.27 (by @HB9HIL)
- Helm chart documentation moved to radiopath/helmchart (by @HB9HIL)
- Doc comments for all functions (by @HB9HIL)

## [1.2.6] - 2026-10-14

### Chore
- Updated renovate rules to pick up all necessary versions (by @HB9HIL)
- Dependency Updates (by @renovate)

## [1.2.5] - 2026-10-14

### Chore
- Dependency Updates (by @HB9HIL)

## [1.2.4] - 2026-10-10

### Chore
- Dependency Updates (by @HB9HIL)

## [1.2.3] - 2026-10-02

### Fixed
- Fixed some fuzzy log messages (by @HB9HIL)

## [1.2.2] - 2026-09-28

### Chore
- Dependency Updates and Pipeline fixes (by @HB9HIL)

## [1.2.1] - 2026-09-28

### Chore
- Dependency Updates (by @HB9HIL)

## [1.2.0] - 2026-09-20

### Added
- Added a rate limiter to the tile proxy (by @HB9HIL)

### Changed
- Dev stack now runs on 9.1-alpine aswell, just as the helmchart in production. (by @HB9HIL)

## [1.1.0] - 2026-09-19

### Added
- Every commit to main branch now builds a :staging image for the staging environment (by @HB9HIL)

### Changed
- Sessions now expire seven days after they were last used instead of thirty days after login, with thirty days from login as a hard cap. Existing sessions are pulled into the shorter window the next time they are used. (by @HB9HIL)

### Fixed
- The ci pipline was still on push to master, fixed to main (by @HB9HIL)

### Chore
- Moved stuff in the repo around to make it a bit more organized (by @HB9HIL)

## [1.0.0] - 2026-09-19

### Added
- Initial public release: point-to-point link analysis and area coverage with the NTIA Irregular Terrain Model, antenna patterns, tree canopy loss, share links, PDF reports, admin panel, Prometheus metrics and a Helm chart. (by @HB9HIL)
