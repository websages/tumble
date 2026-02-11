# Tumble API Redesign

## Overview

This document describes the redesign of the Tumble API to be more RESTful,
consistent, and idiomatic. The redesign removes legacy IRC-specific patterns
and establishes a clean, versioned API surface.

## Goals

- RESTful resource-oriented design
- Consistent URL structure with versioning
- Standardized request/response formats
- Clear content negotiation strategy
- Proper HTTP status code usage
- Consistent error handling

## Authentication Model

This API uses a **Trusted Intermediary** pattern. A single API key is issued
to a service (such as an IRC bot or backend application) that posts on behalf
of users. The `user` field in request bodies identifies which user the
intermediary is acting for.

This design assumes the intermediary is trusted to correctly identify users.
If the API is later opened to direct end-user access (mobile apps, browser
extensions), a per-user authentication scheme should be implemented where
the user identity is derived from the auth token rather than passed in the
request body.

## URL Structure

All API endpoints use the `/api/v1/` prefix with pluralized resource names.

### Resources

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/links` | List links (paginated) |
| POST | `/api/v1/links` | Create a new link |
| GET | `/api/v1/links/{id}` | Get link by ID |
| DELETE | `/api/v1/links/{id}` | Delete link (auth required) |
| GET | `/api/v1/quotes` | List quotes (paginated) |
| POST | `/api/v1/quotes` | Create a new quote |
| GET | `/api/v1/quotes/{id}` | Get quote by ID |
| DELETE | `/api/v1/quotes/{id}` | Delete quote (auth required) |
| GET | `/api/v1/stats` | Site-wide stats and leaderboard |
| GET | `/api/v1/users/{user}/stats` | Per-user statistics |
| GET | `/api/v1/search` | Search links and quotes |
| DELETE | `/api/v1/cache` | Clear preview cache |
| GET | `/api/v1/kittens/daily` | Get current daily kitten |
| PUT | `/api/v1/kittens/daily` | Ensure daily kitten exists |
| DELETE | `/api/v1/kittens/daily` | Remove daily kitten |

### Public Shortlinks

The redirect shortlink remains outside the API namespace:

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/go/{id}` | Redirect to link URL |

## Content Negotiation

### Strategy

1. Format suffix takes precedence: `/api/v1/links/42.json` forces JSON
2. Accept header otherwise: `Accept: application/json`
3. Default to JSON when no preference specified

### Supported Formats

| Suffix | Content-Type | Use Case |
|--------|--------------|----------|
| `.json` | `application/json` | Default, programmatic access |
| `.txt` | `text/plain` | Simple scripts, IRC bots |

### Examples

```bash
# Using Accept header (preferred for clients)
curl -H "Accept: application/json" https://tumble.example.com/api/v1/links/42

# Using suffix (convenient for testing)
curl https://tumble.example.com/api/v1/links/42.json

# Plain text for scripts
curl https://tumble.example.com/api/v1/links/42.txt
```

## Request Format

All creation and mutation endpoints accept JSON request bodies.

### Create Link

```http
POST /api/v1/links
Content-Type: application/json

{
  "url": "https://example.com/article",
  "user": "alice"
}
```

### Create Quote

```http
POST /api/v1/quotes
Content-Type: application/json

{
  "quote": "The only way to do great work is to love what you do.",
  "author": "Steve Jobs",
  "poster": "alice"
}
```

## Response Format

### Field Naming

All fields use snake_case. Legacy field names are mapped as follows:

| Old | New |
|-----|-----|
| `ircLinkID` | `id` |
| `quoteID` | `id` |
| `timestamp` | `created_at` |
| `link_id` | `id` |

### Single Resource

```json
{
  "id": 42,
  "url": "https://example.com/article",
  "title": "Example Article",
  "user": "alice",
  "clicks": 15,
  "created_at": "2026-01-15T10:30:00Z"
}
```

### Collections

Collections are wrapped in an object with metadata:

```json
{
  "data": [
    {"id": 42, "url": "...", "title": "..."},
    {"id": 43, "url": "...", "title": "..."}
  ],
  "meta": {
    "total": 1523,
    "limit": 50,
    "offset": 0
  }
}
```

### Create Response (with duplicate detection)

```json
{
  "id": 456,
  "url": "https://example.com/article",
  "title": "Example Article",
  "user": "bob",
  "created_at": "2026-02-08T14:30:00Z",
  "is_duplicate": true,
  "previous_submissions": [
    {
      "id": 123,
      "user": "alice",
      "created_at": "2026-01-15T10:30:00Z",
      "title": "Example Article"
    }
  ]
}
```

## Status Codes

| Code | Usage |
|------|-------|
| 200 | Successful GET, successful cache clear |
| 201 | Resource created (POST) |
| 204 | Resource deleted (DELETE) |
| 400 | Bad request (malformed input) |
| 401 | Unauthorized (missing API key) |
| 403 | Forbidden (invalid API key) |
| 404 | Resource not found |
| 422 | Validation error (valid JSON but invalid data) |
| 500 | Internal server error |

### Duplicate Link Handling

When a URL is submitted that already exists, a **new record is always created**.
This allows tracking who submitted what and when, even for duplicates. The
response uses 201 Created (not 409 Conflict) because a new resource was created.

The response includes:
- `is_duplicate: true` - flags that this URL was previously submitted
- `previous_submissions` - array of prior submissions with user, date, and ID

This enables clients to display contextual messages like "Bob already posted
this 3 months ago" while still recording the new submission.

## Error Responses

All errors return a consistent JSON structure:

```json
{
  "error": {
    "code": "not_found",
    "message": "Link 42 does not exist"
  }
}
```

### Error Codes

| HTTP Status | Code | Description |
|-------------|------|-------------|
| 400 | `bad_request` | Malformed request |
| 401 | `unauthorized` | API key required |
| 403 | `forbidden` | Invalid API key |
| 404 | `not_found` | Resource does not exist |
| 422 | `validation_error` | Invalid field values |
| 500 | `internal_error` | Server error |

### Validation Errors

Validation errors include field-level details:

```json
{
  "error": {
    "code": "validation_error",
    "message": "Invalid input",
    "details": {
      "url": "must be a valid URL",
      "user": "is required"
    }
  }
}
```

## Authentication

Protected endpoints require the `X-API-Key` header:

```http
DELETE /api/v1/links/42
X-API-Key: your-secret-key
```

### Protected Endpoints

- `DELETE /api/v1/links/{id}`
- `DELETE /api/v1/quotes/{id}`
- `DELETE /api/v1/cache`
- `PUT /api/v1/kittens/daily`
- `DELETE /api/v1/kittens/daily`

### Public Endpoints

All other endpoints are public, including:
- All GET operations (reading links, quotes, stats, search, daily kitten)
- POST operations for creating links and quotes (user specified in body)

### OpenAPI Security Scheme

```json
{
  "securitySchemes": {
    "apiKey": {
      "type": "apiKey",
      "in": "header",
      "name": "X-API-Key"
    }
  }
}
```

## Endpoint Details

### Links

#### List Links

```http
GET /api/v1/links?limit=50&offset=0&user=alice
```

Query parameters:
- `limit` (default: 50, max: 1000)
- `offset` (default: 0)
- `user` (optional, filter by submitter)

#### Create Link

```http
POST /api/v1/links
Content-Type: application/json

{
  "url": "https://example.com/article",
  "user": "alice"
}
```

Returns 201 with the created link. Includes duplicate information if the URL
was previously submitted.

#### Get Link

```http
GET /api/v1/links/42
```

#### Delete Link

```http
DELETE /api/v1/links/42
X-API-Key: your-secret-key
```

Returns 204 on success.

### Quotes

#### List Quotes

```http
GET /api/v1/quotes?limit=50&offset=0
```

#### Create Quote

```http
POST /api/v1/quotes
Content-Type: application/json

{
  "quote": "The quote text",
  "author": "Author Name",
  "poster": "submitter"
}
```

Only `quote` is required. Returns 201.

#### Get Quote

```http
GET /api/v1/quotes/42
```

#### Delete Quote

```http
DELETE /api/v1/quotes/42
X-API-Key: your-secret-key
```

Returns 204 on success.

### Search

```http
GET /api/v1/search?q=keyword&type=links,quotes&limit=50&offset=0
```

Query parameters:
- `q` (required, min 4 characters)
- `type` (optional, comma-separated: `links`, `quotes`, default: both)
- `limit` (default: 50, applies per type)
- `offset` (default: 0, applies per type)

Response uses separate arrays for each content type:

```json
{
  "links": [
    {"id": 42, "url": "...", "title": "...", "user": "alice", "created_at": "..."}
  ],
  "quotes": [
    {"id": 5, "quote": "...", "author": "...", "poster": "bob", "created_at": "..."}
  ],
  "meta": {
    "total_links": 12,
    "total_quotes": 3,
    "limit": 50,
    "offset": 0
  }
}
```

If `type` parameter limits to one type, the other array will be empty.

### Statistics

#### Site-Wide Stats and Leaderboard

```http
GET /api/v1/stats?limit=50&offset=0
```

Returns site-wide totals and a leaderboard of users by activity.

Query parameters:
- `limit` (default: 50, max: 1000) - limits leaderboard entries
- `offset` (default: 0)

Response:

```json
{
  "site": {
    "total_links": 15000,
    "total_quotes": 3200,
    "total_users": 47
  },
  "leaderboard": [
    {"user": "alice", "link_count": 500, "quote_count": 120},
    {"user": "bob", "link_count": 450, "quote_count": 85}
  ],
  "meta": {
    "total": 47,
    "limit": 50,
    "offset": 0
  }
}
```

#### Per-User Stats

```http
GET /api/v1/users/alice/stats
```

Returns statistics for a specific user.

Response:

```json
{
  "user": "alice",
  "link_count": 500,
  "quote_count": 120
}
```

Returns 404 if user has no submissions.

### Cache

#### Clear All Cache

```http
DELETE /api/v1/cache
X-API-Key: your-secret-key
```

Response:
```json
{
  "cleared": "all",
  "count": 47
}
```

#### Clear Specific URL

```http
DELETE /api/v1/cache?url=https://example.com/article
X-API-Key: your-secret-key
```

Response:
```json
{
  "cleared": "https://example.com/article"
}
```

### Daily Kitten

#### Get Daily Kitten

```http
GET /api/v1/kittens/daily
```

Returns today's kitten without triggering a fetch.

Response:
```json
{
  "url": "https://cataas.com/cat/abc123",
  "date": "2026-02-08"
}
```

Returns 404 if no kitten exists for today.

#### Ensure Daily Kitten Exists

```http
PUT /api/v1/kittens/daily
X-API-Key: your-secret-key
```

Fetches a new kitten if one doesn't exist for today.

Response:
```json
{
  "url": "https://cataas.com/cat/abc123",
  "fetched": true,
  "date": "2026-02-08"
}
```

Where `fetched: true` means a new kitten was fetched, `fetched: false` means
one already existed.

#### Remove Daily Kitten

```http
DELETE /api/v1/kittens/daily
X-API-Key: your-secret-key
```

Use this to remove an undesirable kitten, then PUT to fetch a new one.

Returns 204 on success, 404 if no kitten exists.

## Migration Notes

### Deprecated Endpoints

The following endpoints will be removed:

| Old | New |
|-----|-----|
| `POST /link/` | `POST /api/v1/links` |
| `GET /link/{id}` | `GET /r/{id}` (redirect) or `GET /api/v1/links/{id}` (data) |
| `GET /link/{id}.json` | `GET /api/v1/links/{id}` |
| `DELETE /link/{id}` | `DELETE /api/v1/links/{id}` |
| `GET /quote/` | `GET /api/v1/quotes` |
| `POST /quote/` | `POST /api/v1/quotes` |
| `GET /quote/{id}` | `GET /api/v1/quotes/{id}` |
| `GET /quote/{id}.json` | `GET /api/v1/quotes/{id}` |
| `DELETE /quote/{id}` | `DELETE /api/v1/quotes/{id}` |
| `GET /stats` | `GET /api/v1/stats` (HTML removed) |
| `GET /stats.json` | `GET /api/v1/stats` |
| N/A | `GET /api/v1/users/{user}/stats` (new) |
| `GET /search` | `GET /api/v1/search` |
| `GET /ogpreview` | Removed or moved to `/api/v1/preview` |
| `GET /api/caching/invalidate` | `DELETE /api/v1/cache` |
| `POST /api/kitten/fetch` | `PUT /api/v1/kittens/daily` |

### Breaking Changes

1. Query parameter auth (`?secret=...`) removed; use `X-API-Key` header
2. `source=irc` parameter removed; use `Accept: text/plain` header
3. Response field names changed (see Field Naming section)
4. POST for links uses JSON body instead of query parameters
5. `.json` suffix routes consolidated into main resource endpoints
6. 208 status code replaced with 201 + `is_duplicate` field

### Transition Period

Consider running old and new endpoints in parallel during migration:

1. Deploy new `/api/v1/*` endpoints
2. Update clients to use new endpoints
3. Monitor old endpoint usage
4. Remove old endpoints after clients migrate
