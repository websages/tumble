# Daily Kitten Feature Design

## Overview

A background scheduler that fetches a cat image from cataas.com every day at
10 AM Central, captures the redirect URL, and stores it as an image entry
attributed to "cat AAS". The image displays inline in the feed like any other
image, without showing authorship on the frontend.

## Key Behaviors

- Runs at 10:00 AM Central Time daily
- On app startup, checks if today's kitten exists; fetches immediately if missing
- If cataas.com is unreachable, retries at 11 AM, 12 PM, and 1 PM before giving up
- Uses the existing `image` table with user field set to "cat AAS"

## Scheduler Implementation

### Dependencies

Add `github.com/robfig/cron/v3` for scheduling. This is a well-maintained
library that supports timezone-aware cron expressions.

### Scheduler Setup

```go
c := cron.New(cron.WithLocation(centralTime))
c.AddFunc("0 10 * * *", fetchDailyCat)  // 10:00 AM Central
c.Start()
```

### Startup Check

During app initialization, after the scheduler starts, call a function that:

1. Queries the `image` table for entries from "cat AAS" with today's date
2. If none exist, trigger `fetchDailyCat()` immediately

### Retry Logic

If the initial fetch fails, schedule one-off retries at 11 AM, 12 PM, and 1 PM.
After three failures, log at WARN level and give up for that day.

## Fetching the Cat

### HTTP Request to Capture Redirect

Make a request to `https://cataas.com/cat` but don't follow redirects
automatically. Capture the `Location` header to get the specific image URL.

```go
client := &http.Client{
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        return http.ErrUseLastResponse  // Don't follow redirects
    },
}
resp, err := client.Get("https://cataas.com/cat")
imageURL := resp.Header.Get("Location")
```

### Fallback

If cataas returns the image directly (no redirect), use the original URL with
a unique query param like `?ts=<timestamp>` to ensure we get a distinct image.

### Store in Database

Create an `image` record:

- `user`: "cat AAS"
- `url`: the captured redirect URL
- `title`: "Daily Kitten"
- `created_at`: current timestamp

## Frontend Display

No changes needed to image rendering logic. The existing image display will
show the kitten inline since it's just a regular image entry.

### Hide Author for cat AAS Entries

In the template where images are rendered, add a condition to skip displaying
the author when the user is "cat AAS":

```html
{{ if ne .User "cat AAS" }}
  <span class="author">{{ .User }}</span>
{{ end }}
```

## File Structure

### New Files

- `internal/scheduler/scheduler.go` - Cron setup, start/stop lifecycle
- `internal/scheduler/dailycat.go` - Fetch logic, retry handling, database insert

### Integration Points

- Initialize scheduler in `cmd/` main function after database is ready
- Graceful shutdown: call `scheduler.Stop()` on app termination

## Error Handling

- Log all fetch attempts; WARN level for failures, standard level for success
- On network errors or non-2xx/3xx responses, mark as failed and schedule retry
- After 3 failed retries, log a warning and skip that day
- Don't create duplicate entries: check if today's kitten exists before inserting

## Edge Cases

- **Timezone**: Use `America/Chicago` for Central Time (handles DST automatically)
- **Duplicate prevention**: Query by user="cat AAS" and date before inserting
