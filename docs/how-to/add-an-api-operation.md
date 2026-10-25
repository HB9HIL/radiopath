# Add an API operation

This guide adds an operation to the JSON API and uses it in the web app. The example is a made-up
`GET /api/sites/{id}/elevation` that returns the ground elevation at a site; replace it with your own.

## 1. Define the types

In the file of the resource (`internal/web/sites.go` for sites), add the response type. Use snake_case JSON keys,
a `doc` tag for anything that is not obvious, and never return a store struct:

```go
type SiteElevation struct {
	ElevationM float64 `json:"elevation_m" doc:"Ground elevation above sea level"`
}
```

Inputs get their own type (`SiteInput`, not `Site`). Optional fields are pointers with `omitempty`; huma cannot
mark a nested struct as nullable, so make such a field optional with `omitempty` instead. Lists must come back as
`[]`, so build them with `make([]T, 0, n)`.

## 2. Register the operation

Register it in the resource's `register…` function with `op()`, which sets the scope and the 401 response:

```go
huma.Register(api, op("get-site-elevation", http.MethodGet, "/api/sites/{id}/elevation", scopeSitesRead, "Ground elevation at a site"),
	func(ctx context.Context, in *IDPath) (*struct{ Body SiteElevation }, error) {
		st, err := s.Store.GetSite(ctx, userOf(ctx).ID, in.ID)
		if err != nil {
			return nil, s.storeErr(err)
		}
		// ...
	})
```

- Always pass the owner, `userOf(ctx).ID`, to the store; that is what keeps other users' data out.
- Read operations take the `…:read` scope, changes `…:write`. Operations a future API token must not call (password,
  email) take `scopeSession`. An empty scope makes the operation public; add it to `publicOps` in
  `internal/web/api_test.go`, or the test fails.
- Errors: `s.storeErr(err)` turns `store.ErrNotFound` into 404 and logs anything unexpected as 500. Answer known
  cases with a specific message, for example `huma.Error409Conflict("Site is used by a link or coverage")`.
- Validate input with the `check` helper (`c.num`, `c.text`, `c.deg`, `c.add`, `c.err()`), so all invalid fields come
  back in one 422 with their location.
- Set `DefaultStatus` for answers other than 200 (`http.StatusCreated`, `http.StatusNoContent`, `http.StatusAccepted`)
  and list further error codes in `Errors`.

Run `go vet ./...` and `go tool staticcheck ./...`; a schema huma cannot handle panics at registration, which
`make api` shows right away.

## 3. Regenerate the description and the types

```sh
make api
```

This writes `web/openapi.json` and `web/src/lib/api.gen.ts`. Commit both with the change; CI fails when they are
out of date. Logged in, http://localhost:5173/api/openapi.json now lists the new operation.

## 4. Use it in the web app

Export a type alias in `web/src/lib/api.ts` if the app needs the type by name, then load the data with TanStack
Query:

```ts
const elevation = createQuery(() => ({
	queryKey: ['sites', id, 'elevation'],
	queryFn: () => call(api.GET('/api/sites/{id}/elevation', { params: { path: { id } } }))
}));
```

`call()` returns the data or throws an `ApiError`, which `FormErrors` shows. After a change, invalidate the queries
that show the changed data (`queryClient.invalidateQueries({ queryKey: ['sites'] })`). Every new function gets a doc
comment; run `make format` and `make check`.

## 5. Test it

- Logic in Go that can run without a database gets a unit test next to it.
- Behaviour a user sees gets an end-to-end test, see [Run and write end-to-end tests](run-and-write-e2e-tests.md).
- A new binary route outside `/api` also needs an entry in the proxy expression in `web/vite.config.ts`, or it only
  works in production.

Finally add a line to `CHANGELOG.md` under `## [Unreleased]`, see [CONTRIBUTING.md](../../CONTRIBUTING.md#changelog).
