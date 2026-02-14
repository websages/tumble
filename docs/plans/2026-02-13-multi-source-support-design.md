# Multi-Client Support

**Date:** 2026-02-13
**Status:** Proposed

## Problem

Tumble currently serves a single community with no tracking of where
posts originate. As usage expands to multiple IRC channels, Slack
teams, and potentially Discord servers, posts need metadata about
their client. This enables per-client duplicate detection, per-client
filtering, and eventually client-scoped UI views.

## Solution

Add client metadata columns to all three content tables (links,
quotes, images). Duplicate detection becomes scoped per client.
API endpoints accept optional client fields on creation and support
filtering by client on reads. No UI changes in this phase.

## Data Model

Five nullable columns added to `ircLink`, `quote`, and `image`:

| Column | Type | Purpose |
|--------|------|---------|
| `client_type` | VARCHAR(50) | Platform: "irc", "slack", "discord", "api", "web" |
| `client_network` | VARCHAR(255) | IRC server, Slack team ID, Discord guild ID |
| `client_channel` | VARCHAR(255) | IRC channel, Slack/Discord channel ID |
| `client_user_id` | VARCHAR(255) | Platform-specific user ID (null for IRC) |
| `client_user_name` | VARCHAR(255) | Display/mention name at time of post (null for IRC) |

Composite index on `(client_type, client_network, client_channel)`
on each table.

The existing `user` column is unchanged. It continues to store the
poster's name. The new `client_user_id` and `client_user_name` fields
provide platform-specific identity alongside it.

Platform mapping:

| Field | IRC | Slack | Discord |
|-------|-----|-------|---------|
| `client_type` | "irc" | "slack" | "discord" |
| `client_network` | server hostname | team ID | guild ID |
| `client_channel` | channel name | channel ID | channel ID |
| `client_user_id` | null | Slack user ID | Discord user ID |
| `client_user_name` | null | mention name | display name |

GORM model additions (same for all three structs):

```go
ClientType     *string `json:"client_type,omitempty" gorm:"column:client_type"`
ClientNetwork  *string `json:"client_network,omitempty" gorm:"column:client_network"`
ClientChannel  *string `json:"client_channel,omitempty" gorm:"column:client_channel"`
ClientUserID   *string `json:"client_user_id,omitempty" gorm:"column:client_user_id"`
ClientUserName *string `json:"client_user_name,omitempty" gorm:"column:client_user_name"`
```

Pointer types for correct null handling and `omitempty` serialization.

## DDL

GORM AutoMigrate handles schema changes from the struct definitions.
No manual DDL required. Columns are added automatically on next
application startup.

## Data Backfill

All existing rows predate multi-client support and originate from IRC.
A standalone SQL script (run once, manually) backfills them:

```sql
UPDATE ircLink SET client_type = 'irc', client_network = 'jameswhite.org', client_channel = '#soggies' WHERE client_type IS NULL;
UPDATE quote SET client_type = 'irc', client_network = 'jameswhite.org', client_channel = '#soggies' WHERE client_type IS NULL;
UPDATE image SET client_type = 'irc', client_network = 'jameswhite.org', client_channel = '#soggies' WHERE client_type IS NULL;
```

No backfill code in the application. No dead code paths.

## Duplicate Detection

Currently global: any POST of an existing URL returns 208. Changes to
be scoped by `client_type + client_network + client_channel`.

**Rules:**
- POST with client fields: check duplicates within that client tuple
  only. Same URL from a different client is a new link (201).
- POST with null/omitted client fields: fall back to global duplicate
  check (backward compatibility).
- Same URL within same client: 208 Already Reported with previous
  submission info, as before.

**Example:**
1. `https://example.com` from `slack / T12345 / C67890` -> 201
2. Same URL from `irc / jameswhite.org / #soggies` -> 201
3. Same URL from `slack / T12345 / C67890` again -> 208

Images: same scoping applied to MD5 deduplication.

Quotes: no duplicate detection currently, no change.

## API Changes

### POST Endpoints

`POST /api/v1/links`, `POST /api/v1/quotes` accept new optional
fields:

```json
{
  "url": "https://example.com",
  "user": "stahnma",
  "client_type": "slack",
  "client_network": "T12345",
  "client_channel": "C67890",
  "client_user_id": "U99999",
  "client_user_name": "stahnma"
}
```

All client fields are optional. Omitting them stores nulls.

### GET Endpoints

`GET /api/v1/links`, `GET /api/v1/quotes`, `GET /api/v1/search` gain
three optional query parameters:

- `client_type` - filter by platform
- `client_network` - filter by team/server (requires client_type)
- `client_channel` - filter by channel (requires client_type +
  client_network)

Cumulative filtering. Omitting all returns everything (current
behavior).

### Response Payloads

Client fields included with `omitempty`. Null client fields are
omitted from responses:

```json
{
  "id": 1234,
  "url": "https://example.com",
  "user": "stahnma",
  "title": "Example",
  "client_type": "slack",
  "client_network": "T12345",
  "client_channel": "C67890",
  "client_user_id": "U99999",
  "client_user_name": "stahnma"
}
```

## Store Interface Changes

`GetLinkByURL` becomes client-scoped. New filter struct for query
methods:

```go
type ClientFilter struct {
    ClientType    *string
    ClientNetwork *string
    ClientChannel *string
}
```

Used by `GetLinks`, `GetQuotes`, `GetImages`, and the duplicate
detection lookup. When all fields are nil, behaves identically to
current implementation.

`TimelineItem` gains client fields for future frontend use.

## Testing

- Duplicate detection: same URL + same client = 208; same URL +
  different client = 201; null client = global fallback
- Filtering: client params return correct subset; params combine to
  narrow results; no params returns everything
- Backfill: existing rows have correct values after running script
- Serialization: client fields omitted from JSON when null; present
  when populated
- Round-trip: POST with client fields, GET returns them; POST
  without, GET omits them

## API Documentation

Update `internal/assets/openapi.json` to reflect all API changes:

- Add `client_type`, `client_network`, `client_channel`,
  `client_user_id`, and `client_user_name` to request schemas for
  `POST /api/v1/links` and `POST /api/v1/quotes`
- Add `client_type`, `client_network`, `client_channel` as optional
  query parameters on `GET /api/v1/links`, `GET /api/v1/quotes`, and
  `GET /api/v1/search`
- Add client fields to all response schemas (links, quotes, images)
- Document the scoped duplicate detection behavior (208 is now
  per-client when client fields are provided)

## Out of Scope

- UI changes
- Authentication/authorization per client
- User identity lookup table for name history tracking
- Slack, Discord, or other client/bot implementations
- Changes to the existing `source` request parameter (controls
  response format for irc/api/html -- separate concept)
