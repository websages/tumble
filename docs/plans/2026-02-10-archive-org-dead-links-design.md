# Archive.org Integration for Dead Links

**Date:** 2026-02-10
**Status:** Proposed
**Branch:** tumble-dark

## Problem

Tumble has ~20 years of archived links. Over time, many have gone dead
(404s, domain expirations, content removals). The existing system
detects dead links and shows an error badge, but offers no way to view
the original content. Archive.org's Wayback Machine likely has snapshots
of many of these links.

## Solution

Integrate with the Wayback Machine Availability API to check whether
dead links have archived snapshots. When a snapshot exists, show a
"View on Archive.org" link alongside the error badge so users can still
access the original content.

## Data Model

New `archive_lookups` table, independent of `ircLink` and
`link_previews`:

| Column | Type | Purpose |
|--------|------|---------|
| `url` | string, primary key | The original link URL |
| `archive_url` | string, nullable | Wayback snapshot URL |
| `snapshot_at` | timestamp, nullable | When the snapshot was captured |
| `status` | string | `found`, `not_found`, or `error` |
| `checked_at` | timestamp | When we last queried archive.org |

GORM model:

```go
type ArchiveLookup struct {
    URL        string     `gorm:"column:url;primaryKey"`
    ArchiveURL *string    `gorm:"column:archive_url"`
    SnapshotAt *time.Time `gorm:"column:snapshot_at"`
    Status     string     `gorm:"column:status"`
    CheckedAt  time.Time  `gorm:"column:checked_at"`
}
```

### Key decisions

- **Keyed on URL, not IRCLink ID.** Multiple IRCLinks can share the
  same URL (duplicates). One lookup per unique URL.
- **Three-state status.** `found` = snapshot exists, `not_found` =
  checked but no snapshot, `error` = archive.org was unreachable.
  No row = never checked.
- **No cascade on IRCLink deletion.** Orphaned rows are harmless and
  small. This matches existing `link_previews` behavior.

## Archive.org API Wrapper

Single function wrapping the Wayback Machine Availability API:

- **Endpoint:** `GET https://archive.org/wayback/available?url={url}`
- **Response:** `{"archived_snapshots": {"closest": {"url": "...",
  "timestamp": "20190115120000", "status": "200"}}}`
- Empty `archived_snapshots` or missing `closest` means no snapshot.
- Parses `timestamp` (format `YYYYMMDDHHmmss`) into `time.Time`.
- Returns `(archiveURL, snapshotTime, error)`.
- Uses `golang.org/x/time/rate` limiter at 5 req/s.

## Lookup Strategy: Hybrid

Two paths for checking archive.org:

### Background batch job

Runs alongside the existing scheduler (daily kittens):

- **Trigger:** On app startup, then once daily.
- **Targets:** URLs with a cached error in `link_previews` but no row
  (or `error`-status row) in `archive_lookups`.
- **Rate:** ~5 req/s with sleeps between requests.
- **Logging:** Progress updates (e.g. "checked 50/500 dead links").

Recheck policy:

| Status | Recheck after |
|--------|---------------|
| `found` | Never |
| `not_found` | 30 days |
| `error` | 24 hours |

On first deploy, backfills ~500-1000 dead links in a few minutes.
Subsequent daily runs only process newly dead links and stale
`not_found`/`error` rechecks.

**Restart safety:** Results are persisted. Restarting the app resumes
from where it left off — already-checked URLs are skipped.

### Lazy on-demand lookup

For newly detected dead links:

1. After `cacheErrorPreview()` stores an error, check if an
   `archive_lookups` row exists for the URL.
2. If no row (or stale row), call the Wayback API inline (~200-500ms).
3. Store the result in `archive_lookups`.
4. Return archive data alongside the error response.

If a row already exists and is fresh, include it in the response
without an API call.

## API Response Changes

The existing `GET /api/v1/preview?url=...` error response is extended:

```json
{
  "error": "Not Found",
  "status": "404",
  "archive_url": "https://web.archive.org/web/20190115/...",
  "archive_snapshot_at": "2019-01-15T12:00:00Z"
}
```

No new API endpoints. Archive data piggybacks on the existing preview
flow.

## Frontend Changes

In `handlePreviewData()`, when `data.error` is present and
`data.archive_url` exists, append an archive link after the error
badge:

```
[404] example.com/dead-page  ·  View on Archive.org (Jan 2019)
```

- Archive link opens in a new tab.
- Styled for both the default theme and Scott Mode.
- If no archive snapshot exists, behavior is unchanged from today.

## Store Interface Additions

Four new methods on the Store interface:

```go
GetArchiveLookup(ctx context.Context, url string) (*ArchiveLookup, error)
UpsertArchiveLookup(ctx context.Context, lookup *ArchiveLookup) error
GetUncheckedDeadLinkURLs(ctx context.Context) ([]string, error)
GetStaleArchiveLookups(ctx context.Context, recheckAfter time.Duration) ([]string, error)
```

## Scope

### What changes

- New `ArchiveLookup` model in `internal/data/store.go`
- Four new store methods in `internal/data/gorm_store.go`
- New archive.org API wrapper (new file in `internal/handler/` or
  `internal/service/`)
- New scheduled job in `internal/scheduler/`
- Extended `cacheErrorPreview()` flow in `internal/handler/preview.go`
- Extended error response in preview handler
- Frontend JS in `index.html` (both default and Scott Mode)
- CSS for archive link styling (both themes)

### What does NOT change

- IRCLink model or table
- LinkPreview model or table
- Preview cache TTL logic
- Search exclusion logic
- Non-error preview rendering
- API endpoints or authentication
- OpenAPI spec (archive data is an optional extension of existing
  preview response, not a new endpoint)

## Future Work

- Submit dead links to archive.org's Save Page Now API for links with
  no snapshot
- Show archive availability in search results
- Admin dashboard showing dead link / archive coverage stats
