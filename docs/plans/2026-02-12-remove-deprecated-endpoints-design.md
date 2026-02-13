# Remove Deprecated API Endpoints

## Goal

Remove all legacy/deprecated API endpoints and consolidate on the v1
API. Move `/ogpreview` under the v1 namespace as `/api/v1/preview`.

## Scope

### Routes to Remove (`cmd/tumble/main.go`)

| Route | Handler | Reason |
|-------|---------|--------|
| `/irclink/` | `IRCLinkHandler` | Replaced by `/api/v1/links` |
| `/index.cgi` | `Index` | CGI alias for `/` |
| `/search.cgi` | `Search` | CGI alias for `/search` |
| `/ogpreview` | `OGPreviewHandler` | Moving to `/api/v1/preview` |
| `/ogpreview.cgi` | `OGPreviewHandler` | CGI alias |
| `/buttons/` | `ButtonHandler` | Feature removed |
| `/buttons/button.cgi` | `ButtonHandler` | Feature removed |
| `/v0/` | `Index` | v0 alias |
| `/v0/index.cgi` | `Index` | v0 alias |
| `/v0/search.cgi` | `Search` | v0 alias |
| `/v0/link/` | `IRCLinkHandler` | v0 alias |
| `/v0/irclink/` | `IRCLinkHandler` | v0 alias |
| `/v0/ogpreview.cgi` | `OGPreviewHandler` | v0 alias |
| `/v0/quote/` | `QuoteHandler` | v0 alias |
| `/api/caching/invalidate` | `InvalidateCacheHandler` | Replaced by `DELETE /api/v1/cache` |
| `/api/kitten/fetch` | `FetchKittenHandler` | Replaced by `PUT /api/v1/kittens/daily` |

### Route to Add

| Route | Handler | Notes |
|-------|---------|-------|
| `/api/v1/preview` | `OGPreviewHandler` | Replaces `/ogpreview` |

### Routes Kept (not deprecated)

`/`, `/link/`, `/quote/`, `/search`, `/stats`, `/stats.json`,
`/index.xml`, `/go/`, `/api/v1/*`, `/api/docs`

## Dead Code Removal

### Handler code

- `ButtonHandler()` in `internal/handler/handlers.go`

### Template functions

- `irclinkURL` in `internal/templates/renderer.go` (both HTML and
  XML variants)

### Templates

- `internal/templates/views/tumble_buttons.html`

### Static assets

- `internal/assets/buttons/` directory (`button.cgi`, `index.html`)

### Template updates

- `internal/templates/views/header.html` -- remove `/buttons/` from
  desktop nav and mobile drawer nav
- `internal/templates/views/index.html` -- change `/ogpreview` JS
  call to `/api/v1/preview`, remove `/buttons/` links

## Test Updates

### `tests/add_link.sh`

Change from:
```
curl -s "$BASE_URL/irclink/?user=$USER&url=$URL&source=irc"
```
To POST to `/api/v1/links` with JSON body.

### `tests/delete_link.sh`

Change from:
```
curl -v -X DELETE "$BASE_URL/irclink/?id=$ID"
```
To `DELETE /api/v1/links/$ID` with `X-API-Key` header.

### `tests/api_test.sh`

- Remove tests for `/irclink/`, `/search.cgi`, `/v0/*`,
  `/ogpreview.cgi`
- Add test for `/api/v1/preview`
- Keep tests for routes that remain (`/`, `/search`, etc.)

## Documentation Updates

### `internal/assets/openapi.json`

- Add `/api/v1/preview` endpoint with query parameter `url` and
  JSON response schema
- Remove any references to deprecated endpoints

### `README.md`

- Remove `/irclink/` documentation
- Update endpoint references to v1 API

## Implementation Order

1. Add `/api/v1/preview` route
2. Update frontend JS to use `/api/v1/preview`
3. Remove deprecated routes from `main.go`
4. Remove dead handler code, templates, and assets
5. Update nav templates
6. Update test scripts
7. Update OpenAPI spec and README
8. Run `make test` and `make test-api` to verify
