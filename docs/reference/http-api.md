# HTTP API

The app uses the same JSON API that other programs can use, under `/api`. It is
described by OpenAPI 3.1 at `/api/openapi.json` (for logged-in users only; there is
no docs page, it would load its scripts from a CDN), generated from the Go types with
[huma](https://huma.rocks) (`internal/web/api.go`, operations next to their
resource in `sites.go`, `links.go`, `coverages.go`, ...). Errors are RFC 7807
problem details; invalid input answers 422 with one entry per field in
`errors[].location` (`body.lat`). Authentication is the session cookie of
`POST /api/session`; personal API tokens with scopes for third-party programs
are planned. Rasters, reports and the job event stream of a coverage
(`/coverages/{id}/image.png`, `composite.png`, `report.pdf`, `events`) are plain
HTTP routes next to the API, as are their shared variants under `/s/`.

## Routes outside the JSON API

| Route | Auth | Answer |
|---|---|---|
| `GET /links/{id}/report.pdf` | session | PDF report of a link |
| `GET /coverages/{id}/image.png` | session | coverage raster, coloured with its legend |
| `GET /coverages/{id}/composite.png?with=1,2` | session | this raster merged with other computed coverages; bounds in `X-Raster-Bounds` (S,W,N,E) |
| `GET /coverages/{id}/report.pdf` | session | PDF report of a coverage |
| `GET /coverages/{id}/events` | session | server-sent events with the job state (`state`, `since` in RFC 3339, `error`) until it is final; at most 5 open streams per user, 429 beyond |
| `GET /s/l/{token}/report.pdf`, `GET /s/c/{token}/{image.png,composite.png,report.pdf}` | share token | the same for shared links and coverages; overlays only as released by the owner |
| `GET /tiles/{z}/{x}/{y}.png` | none | map tiles through the proxy |
| `GET /healthz`, `GET /readyz` | none | liveness and readiness; `not ready` without details |

Without a session these routes answer 401; the web app sends the browser to the login.

## Errors

Every error of the JSON API is an RFC 7807 problem (`application/problem+json`) with `title`, `status` and
`detail`. Invalid input answers 422 with one entry per field in `errors[]`, for example
`{"location": "body.lat", "message": "Latitude: must be between -90 and 90"}`; checks across fields use the
location `body`. 401 means no session, 404 also covers objects of other users, 409 a conflict (a site still in
use, a taken name), 429 a rate limit, 503 maintenance mode. The public account operations never tell whether a
name or an address exists: registration answers 204 either way and says what happened only in the mail to that
address. A failed calculation reports a fixed reason (`No terrain data for this area.`, `Calculation failed.`);
the underlying error is only in the log.

## Conventions

- JSON keys are snake_case; optional values are omitted rather than `null` where possible; empty lists are `[]`.
- The API is not versioned. It only changes in backwards-compatible ways: new operations and fields are added,
  existing ones are not renamed or removed.
