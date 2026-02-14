# Multi-Source Support

**Date:** 2026-02-13
**Status:** Proposed

## Problem

Tumble currently serves a single community with no tracking of where
posts originate. As usage expands to multiple IRC channels, Slack
teams, and potentially Discord servers, posts need metadata about
their source. This enables per-source duplicate detection, per-source
filtering, and eventually source-scoped UI views.

## Solution

Add source metadata columns to all three content tables (links,
quotes, images). Duplicate detection becomes scoped per source.
API endpoints accept optional source fields on creation and support
filtering by source on reads. No UI changes in this phase.

## Data Model

Five nullable columns added to `ircLink`, `quote`, and `image`:

| Column | Type | Purpose |
|--------|------|---------|
| `source_type` | VARCHAR(50) | Platform: "irc", "slack", "discord", "api", "web" |
| `source_network` | VARCHAR(255) | IRC server, Slack team ID, Discord guild ID |
| `source_channel` | VARCHAR(255) | IRC channel, Slack/Discord channel ID |
| `source_user_id` | VARCHAR(255) | Platform-specific user ID (null for IRC) |
| `source_user_name` | VARCHAR(255) | Display/mention name at time of post (null for IRC) |

Composite index on `(source_type, source_network, source_channel)`
on each table.

The existing `user` column is unchanged. It continues to store the
poster's name. The new `source_user_id` and `source_user_name` fields
provide platform-specific identity alongside it.

Platform mapping:

| Field | IRC | Slack | Discord |
|-------|-----|-------|---------|
| `source_type` | "irc" | "slack" | "discord" |
| `source_network` | server hostname | team ID | guild ID |
| `source_channel` | channel name | channel ID | channel ID |
| `source_user_id` | null | Slack user ID | Discord user ID |
| `source_user_name` | null | mention name | display name |

GORM model additions (same for all three structs):

```go
SourceType     *string `json:"source_type,omitempty" gorm:"column:source_type"`
SourceNetwork  *string `json:"source_network,omitempty" gorm:"column:source_network"`
SourceChannel  *string `json:"source_channel,omitempty" gorm:"column:source_channel"`
SourceUserID   *string `json:"source_user_id,omitempty" gorm:"column:source_user_id"`
SourceUserName *string `json:"source_user_name,omitempty" gorm:"column:source_user_name"`
```

Pointer types for correct null handling and `omitempty` serialization.

## DDL

GORM AutoMigrate handles schema changes from the struct definitions.
No manual DDL required. Columns are added automatically on next
application startup.

## Data Backfill

All existing rows predate multi-source support and originate from IRC.
A standalone SQL script (run once, manually) backfills them:

```sql
UPDATE ircLink SET source_type = 'irc', source_network = 'jameswhite.org', source_channel = '#soggies' WHERE source_type IS NULL;
UPDATE quote SET source_type = 'irc', source_network = 'jameswhite.org', source_channel = '#soggies' WHERE source_type IS NULL;
UPDATE image SET source_type = 'irc', source_network = 'jameswhite.org', source_channel = '#soggies' WHERE source_type IS NULL;
```

No backfill code in the application. No dead code paths.

## Duplicate Detection

Currently global: any POST of an existing URL returns 208. Changes to
be scoped by `source_type + source_network + source_channel`.

**Rules:**
- POST with source fields: check duplicates within that source tuple
  only. Same URL from a different source is a new link (201).
- POST with null/omitted source fields: fall back to global duplicate
  check (backward compatibility).
- Same URL within same source: 208 Already Reported with previous
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
  "source_type": "slack",
  "source_network": "T12345",
  "source_channel": "C67890",
  "source_user_id": "U99999",
  "source_user_name": "stahnma"
}
```

All source fields are optional. Omitting them stores nulls.

### GET Endpoints

`GET /api/v1/links`, `GET /api/v1/quotes`, `GET /api/v1/search` gain
three optional query parameters:

- `source_type` - filter by platform
- `source_network` - filter by team/server (requires source_type)
- `source_channel` - filter by channel (requires source_type +
  source_network)

Cumulative filtering. Omitting all returns everything (current
behavior).

### Response Payloads

Source fields included with `omitempty`. Null source fields are
omitted from responses:

```json
{
  "id": 1234,
  "url": "https://example.com",
  "user": "stahnma",
  "title": "Example",
  "source_type": "slack",
  "source_network": "T12345",
  "source_channel": "C67890",
  "source_user_id": "U99999",
  "source_user_name": "stahnma"
}
```

## Store Interface Changes

`GetLinkByURL` becomes source-scoped. New filter struct for query
methods:

```go
type SourceFilter struct {
    SourceType    *string
    SourceNetwork *string
    SourceChannel *string
}
```

Used by `GetLinks`, `GetQuotes`, `GetImages`, and the duplicate
detection lookup. When all fields are nil, behaves identically to
current implementation.

`TimelineItem` gains source fields for future frontend use.

## Testing

- Duplicate detection: same URL + same source = 208; same URL +
  different source = 201; null source = global fallback
- Filtering: source params return correct subset; params combine to
  narrow results; no params returns everything
- Backfill: existing rows have correct values after running script
- Serialization: source fields omitted from JSON when null; present
  when populated
- Round-trip: POST with source fields, GET returns them; POST
  without, GET omits them

## API Documentation

Update `internal/assets/openapi.json` to reflect all API changes:

- Add `source_type`, `source_network`, `source_channel`,
  `source_user_id`, and `source_user_name` to request schemas for
  `POST /api/v1/links` and `POST /api/v1/quotes`
- Add `source_type`, `source_network`, `source_channel` as optional
  query parameters on `GET /api/v1/links`, `GET /api/v1/quotes`, and
  `GET /api/v1/search`
- Add source fields to all response schemas (links, quotes, images)
- Document the scoped duplicate detection behavior (208 is now
  per-source when source fields are provided)

## Out of Scope

- UI changes
- Authentication/authorization per source
- User identity lookup table for name history tracking
- Slack, Discord, or other client/bot implementations
- Changes to the existing `source` request parameter (controls
  response format for irc/api/html — separate concept)
