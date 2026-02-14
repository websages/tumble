# Multi-Source Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add source metadata to links, quotes, and images so Tumble can track where posts originate and scope duplicate detection per source.

**Architecture:** Add five nullable source columns to all three content tables via GORM struct changes. Modify the Store interface to accept a `SourceFilter` for queries and duplicate detection. Update API handlers to parse, pass through, and return source fields.

**Tech Stack:** Go, GORM, SQLite/MySQL, `net/http`, `httptest` for testing

---

### Task 1: Add source fields to data models

**Files:**
- Modify: `internal/data/store.go`

**Step 1: Add source fields to IRCLink struct**

In `internal/data/store.go`, add five fields to the `IRCLink` struct (after the `ContentType` field, line 15):

```go
type IRCLink struct {
	ID             int       `json:"ircLinkID" gorm:"column:ircLinkID;primaryKey"`
	Timestamp      time.Time `json:"timestamp" gorm:"column:timestamp"`
	User           string    `json:"user" gorm:"column:user;index"`
	Title          string    `json:"title" gorm:"column:title"`
	URL            string    `json:"url" gorm:"column:url"`
	Clicks         int       `json:"clicks" gorm:"column:clicks;default:0"`
	ContentType    string    `json:"content_type" gorm:"column:content_type"`
	SourceType     *string   `json:"source_type,omitempty" gorm:"column:source_type;type:varchar(50);index:idx_source,priority:1"`
	SourceNetwork  *string   `json:"source_network,omitempty" gorm:"column:source_network;type:varchar(255);index:idx_source,priority:2"`
	SourceChannel  *string   `json:"source_channel,omitempty" gorm:"column:source_channel;type:varchar(255);index:idx_source,priority:3"`
	SourceUserID   *string   `json:"source_user_id,omitempty" gorm:"column:source_user_id;type:varchar(255)"`
	SourceUserName *string   `json:"source_user_name,omitempty" gorm:"column:source_user_name;type:varchar(255)"`
}
```

**Step 2: Add source fields to Image struct**

Same five fields added to `Image` (after `MD5Sum`, line 29):

```go
type Image struct {
	ID             int       `json:"imageID" gorm:"column:imageID;primaryKey"`
	Timestamp      time.Time `json:"timestamp" gorm:"column:timestamp"`
	Title          string    `json:"title" gorm:"column:title"`
	Link           string    `json:"link" gorm:"column:link"`
	URL            string    `json:"url" gorm:"column:url"`
	MD5Sum         string    `json:"md5sum" gorm:"column:md5sum"`
	SourceType     *string   `json:"source_type,omitempty" gorm:"column:source_type;type:varchar(50);index:idx_source,priority:1"`
	SourceNetwork  *string   `json:"source_network,omitempty" gorm:"column:source_network;type:varchar(255);index:idx_source,priority:2"`
	SourceChannel  *string   `json:"source_channel,omitempty" gorm:"column:source_channel;type:varchar(255);index:idx_source,priority:3"`
	SourceUserID   *string   `json:"source_user_id,omitempty" gorm:"column:source_user_id;type:varchar(255)"`
	SourceUserName *string   `json:"source_user_name,omitempty" gorm:"column:source_user_name;type:varchar(255)"`
}
```

**Step 3: Add source fields to Quote struct**

Same five fields added to `Quote` (after `Poster`, line 42):

```go
type Quote struct {
	ID             int       `json:"quoteID" gorm:"column:quoteID;primaryKey"`
	Timestamp      time.Time `json:"timestamp" gorm:"column:timestamp"`
	Quote          string    `json:"quote" gorm:"column:quote"`
	Author         string    `json:"author" gorm:"column:author;type:varchar(255);index"`
	Poster         string    `json:"poster,omitempty" gorm:"column:poster;type:varchar(255);index"`
	SourceType     *string   `json:"source_type,omitempty" gorm:"column:source_type;type:varchar(50);index:idx_source,priority:1"`
	SourceNetwork  *string   `json:"source_network,omitempty" gorm:"column:source_network;type:varchar(255);index:idx_source,priority:2"`
	SourceChannel  *string   `json:"source_channel,omitempty" gorm:"column:source_channel;type:varchar(255);index:idx_source,priority:3"`
	SourceUserID   *string   `json:"source_user_id,omitempty" gorm:"column:source_user_id;type:varchar(255)"`
	SourceUserName *string   `json:"source_user_name,omitempty" gorm:"column:source_user_name;type:varchar(255)"`
}
```

**Step 4: Add source fields to TimelineItem struct**

Add source fields to `TimelineItem` (after `ContentType`, line 65):

```go
type TimelineItem struct {
	Type           string    `json:"type"`
	ID             int       `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	Title          string    `json:"title"`
	URL            string    `json:"url"`
	Content        string    `json:"content"`
	Author         string    `json:"author"`
	MD5Sum         string    `json:"md5sum"`
	ContentType    string    `json:"contentType" gorm:"column:content_type"`
	SourceType     *string   `json:"source_type,omitempty"`
	SourceNetwork  *string   `json:"source_network,omitempty"`
	SourceChannel  *string   `json:"source_channel,omitempty"`
	SourceUserID   *string   `json:"source_user_id,omitempty"`
	SourceUserName *string   `json:"source_user_name,omitempty"`
}
```

**Step 5: Add SourceFilter type**

Add a new type after the `TimelineItem` struct:

```go
// SourceFilter is used to filter queries by source metadata.
// When all fields are nil, no source filtering is applied.
type SourceFilter struct {
	SourceType    *string
	SourceNetwork *string
	SourceChannel *string
}
```

**Step 6: Run tests to verify nothing broke**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && make test`
Expected: All existing tests pass (struct additions are backward compatible)

**Step 7: Commit**

```bash
git add internal/data/store.go
git commit -m "feat: add source metadata fields to data models"
```

---

### Task 2: Update Store interface and GormStore for source-aware queries

**Files:**
- Modify: `internal/data/store.go` (interface)
- Modify: `internal/data/gorm_store.go` (implementation)

**Step 1: Write failing test for source-scoped InsertIRCLink**

Create test in a new file `internal/data/source_filter_test.go`:

```go
package data

import (
	"testing"
)

func TestSourceFilter_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		filter   SourceFilter
		expected bool
	}{
		{"all nil", SourceFilter{}, true},
		{"type set", SourceFilter{SourceType: strPtr("irc")}, false},
		{"network set", SourceFilter{SourceNetwork: strPtr("server")}, false},
		{"channel set", SourceFilter{SourceChannel: strPtr("#chan")}, false},
		{"all set", SourceFilter{
			SourceType:    strPtr("irc"),
			SourceNetwork: strPtr("server"),
			SourceChannel: strPtr("#chan"),
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter.IsEmpty()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && go test -v ./internal/data/ -run TestSourceFilter`
Expected: FAIL — `SourceFilter` has no `IsEmpty` method

**Step 3: Implement IsEmpty on SourceFilter**

In `internal/data/store.go`, add method after the `SourceFilter` struct:

```go
// IsEmpty returns true if no source filter fields are set.
func (f SourceFilter) IsEmpty() bool {
	return f.SourceType == nil && f.SourceNetwork == nil && f.SourceChannel == nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && go test -v ./internal/data/ -run TestSourceFilter`
Expected: PASS

**Step 5: Update Store interface signatures**

In `internal/data/store.go`, change these method signatures:

Old:
```go
InsertIRCLink(ctx context.Context, user, title, url, contentType string) (int, error)
GetIRCLinksByURL(ctx context.Context, url string) ([]IRCLink, error)
GetRecentIRCLinks(ctx context.Context, days int, offsetDays int) ([]IRCLink, error)
GetRecentQuotes(ctx context.Context, days int, offsetDays int) ([]Quote, error)
GetRecentImages(ctx context.Context, days int, offsetDays int) ([]Image, error)
InsertQuote(ctx context.Context, quote, author, poster string) (int, error)
SearchIRCLinks(ctx context.Context, query string) ([]IRCLink, error)
SearchQuotes(ctx context.Context, query string) ([]Quote, error)
```

New:
```go
InsertIRCLink(ctx context.Context, link *IRCLink) (int, error)
GetIRCLinksByURL(ctx context.Context, url string, filter SourceFilter) ([]IRCLink, error)
GetRecentIRCLinks(ctx context.Context, days int, offsetDays int, filter SourceFilter) ([]IRCLink, error)
GetRecentQuotes(ctx context.Context, days int, offsetDays int, filter SourceFilter) ([]Quote, error)
GetRecentImages(ctx context.Context, days int, offsetDays int, filter SourceFilter) ([]Image, error)
InsertQuote(ctx context.Context, quote *Quote) (int, error)
InsertImage(ctx context.Context, image *Image) (int, error)
SearchIRCLinks(ctx context.Context, query string, filter SourceFilter) ([]IRCLink, error)
SearchQuotes(ctx context.Context, query string, filter SourceFilter) ([]Quote, error)
```

**Step 6: Update GormStore.InsertIRCLink**

In `internal/data/gorm_store.go`, change `InsertIRCLink` (line 204):

Old:
```go
func (s *GormStore) InsertIRCLink(ctx context.Context, user, title, url, contentType string) (int, error) {
	link := IRCLink{
		User:        user,
		Title:       title,
		URL:         url,
		ContentType: contentType,
		Timestamp:   time.Now(),
		Clicks:      0,
	}
	err := s.db.WithContext(ctx).Create(&link).Error
	return link.ID, err
}
```

New:
```go
func (s *GormStore) InsertIRCLink(ctx context.Context, link *IRCLink) (int, error) {
	link.Timestamp = time.Now()
	link.Clicks = 0
	err := s.db.WithContext(ctx).Create(link).Error
	return link.ID, err
}
```

**Step 7: Update GormStore.InsertQuote**

Find the `InsertQuote` method in `gorm_store.go` and update similarly:

Old:
```go
func (s *GormStore) InsertQuote(ctx context.Context, quote, author, poster string) (int, error) {
	q := Quote{
		Quote:     quote,
		Author:    author,
		Poster:    poster,
		Timestamp: time.Now(),
	}
	err := s.db.WithContext(ctx).Create(&q).Error
	return q.ID, err
}
```

New:
```go
func (s *GormStore) InsertQuote(ctx context.Context, quote *Quote) (int, error) {
	quote.Timestamp = time.Now()
	err := s.db.WithContext(ctx).Create(quote).Error
	return quote.ID, err
}
```

**Step 8: Update GormStore.InsertImage**

Old:
```go
func (s *GormStore) InsertImage(ctx context.Context, title, link, url string) (int, error) {
	img := Image{
		Title:     title,
		Link:      link,
		URL:       url,
		Timestamp: time.Now(),
	}
	err := s.db.WithContext(ctx).Create(&img).Error
	return img.ID, err
}
```

New:
```go
func (s *GormStore) InsertImage(ctx context.Context, image *Image) (int, error) {
	image.Timestamp = time.Now()
	err := s.db.WithContext(ctx).Create(image).Error
	return image.ID, err
}
```

**Step 9: Update GormStore.GetIRCLinksByURL for source-scoped duplicate detection**

In `gorm_store.go` (line 191):

Old:
```go
func (s *GormStore) GetIRCLinksByURL(ctx context.Context, url string) ([]IRCLink, error) {
	var links []IRCLink
	err := s.db.WithContext(ctx).
		Where("url = ?", url).
		Order("timestamp DESC").
		Find(&links).Error
	return links, err
}
```

New:
```go
func (s *GormStore) GetIRCLinksByURL(ctx context.Context, url string, filter SourceFilter) ([]IRCLink, error) {
	var links []IRCLink
	query := s.db.WithContext(ctx).Where("url = ?", url)
	query = applySourceFilter(query, filter)
	err := query.Order("timestamp DESC").Find(&links).Error
	return links, err
}
```

**Step 10: Update GormStore.GetRecentIRCLinks for source filtering**

In `gorm_store.go` (line 32):

Old:
```go
func (s *GormStore) GetRecentIRCLinks(ctx context.Context, startDays int, endDays int) ([]IRCLink, error) {
	var links []IRCLink
	now := time.Now()
	startDate := now.AddDate(0, 0, -startDays)
	endDate := now.AddDate(0, 0, -endDays)

	err := s.db.WithContext(ctx).
		Where("timestamp >= ? AND timestamp <= ?", startDate, endDate).
		Order("timestamp DESC").
		Find(&links).Error
	return links, err
}
```

New:
```go
func (s *GormStore) GetRecentIRCLinks(ctx context.Context, startDays int, endDays int, filter SourceFilter) ([]IRCLink, error) {
	var links []IRCLink
	now := time.Now()
	startDate := now.AddDate(0, 0, -startDays)
	endDate := now.AddDate(0, 0, -endDays)

	query := s.db.WithContext(ctx).
		Where("timestamp >= ? AND timestamp <= ?", startDate, endDate)
	query = applySourceFilter(query, filter)
	err := query.Order("timestamp DESC").Find(&links).Error
	return links, err
}
```

**Step 11: Apply same pattern to GetRecentQuotes, GetRecentImages, SearchIRCLinks, SearchQuotes**

Add `filter SourceFilter` parameter and `applySourceFilter(query, filter)` call to each. Follow exact same pattern as Step 10.

**Step 12: Add applySourceFilter helper**

Add this function to `gorm_store.go`:

```go
// applySourceFilter adds WHERE clauses for source metadata fields.
// When filter is empty, no clauses are added (backward compatible).
func applySourceFilter(query *gorm.DB, filter SourceFilter) *gorm.DB {
	if filter.SourceType != nil {
		query = query.Where("source_type = ?", *filter.SourceType)
	}
	if filter.SourceNetwork != nil {
		query = query.Where("source_network = ?", *filter.SourceNetwork)
	}
	if filter.SourceChannel != nil {
		query = query.Where("source_channel = ?", *filter.SourceChannel)
	}
	return query
}
```

**Step 13: Update all other callers of changed methods**

Search the codebase for all calls to `InsertIRCLink`, `InsertQuote`, `InsertImage`, `GetIRCLinksByURL`, `GetRecentIRCLinks`, `GetRecentQuotes`, `GetRecentImages`, `SearchIRCLinks`, `SearchQuotes`. Update each call site to pass the new parameters. For existing callers that don't have source context, pass `data.SourceFilter{}` (empty filter).

Key callers to update:
- `internal/handler/api_v1_links.go` — `apiV1CreateLink`, `apiV1ListLinks`
- `internal/handler/api_v1_quotes.go` — `apiV1CreateQuote`, `apiV1ListQuotes`
- `internal/handler/api_v1_search.go` — `APIv1SearchHandler`
- `internal/handler/irclink.go` — legacy handler (if it exists)
- `internal/handler/handlers.go` — any frontend handlers calling these
- `internal/scheduler/` — any background jobs
- All test mock implementations

**Step 14: Update mock stores in test files**

Update `mockAPIStore` in `internal/handler/api_v1_links_test.go` to match new signatures:

```go
func (m *mockAPIStore) GetRecentIRCLinks(ctx context.Context, days int, offsetDays int, filter data.SourceFilter) ([]data.IRCLink, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.links, nil
}

func (m *mockAPIStore) GetIRCLinksByURL(ctx context.Context, url string, filter data.SourceFilter) ([]data.IRCLink, error) {
	if m.linksByURLFn != nil {
		return m.linksByURLFn(url)
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.linksByURL, nil
}

func (m *mockAPIStore) InsertIRCLink(ctx context.Context, link *data.IRCLink) (int, error) {
	if m.insertLinkFn != nil {
		return m.insertLinkFn(link.User, link.Title, link.URL, link.ContentType)
	}
	if m.err != nil {
		return 0, m.err
	}
	return m.insertedLinkID, nil
}
```

Do the same for `mockQuoteStore` in `api_v1_quotes_test.go` and any other mock stores.

**Step 15: Run tests**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && make test`
Expected: All tests pass. Compilation succeeds with updated signatures.

**Step 16: Commit**

```bash
git add internal/data/store.go internal/data/gorm_store.go internal/data/source_filter_test.go
git add internal/handler/
git commit -m "feat: update Store interface for source-aware queries

Change Insert methods to accept struct pointers instead of
individual parameters. Add SourceFilter to Get and Search
methods. Add applySourceFilter helper for GORM queries."
```

---

### Task 3: Update API handlers to accept and return source fields

**Files:**
- Modify: `internal/handler/api_v1_types.go`
- Modify: `internal/handler/api_v1_links.go`
- Modify: `internal/handler/api_v1_quotes.go`
- Modify: `internal/handler/api_v1_search.go`

**Step 1: Write failing test for source fields in link creation**

Add to `internal/handler/api_v1_links_test.go`, in the `TestAPIv1_CreateLink` function's test table:

```go
{
	name: "valid link with source fields",
	body: `{"url":"https://example.com","user":"testuser","source_type":"slack","source_network":"T12345","source_channel":"C67890","source_user_id":"U99999","source_user_name":"testuser"}`,
	store: &mockAPIStore{insertedLinkID: 42},
	expectedStatus: http.StatusCreated,
	checkBody: func(t *testing.T, body []byte) {
		var resp APILinkCreateResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.SourceType == nil || *resp.SourceType != "slack" {
			t.Errorf("expected source_type 'slack', got %v", resp.SourceType)
		}
		if resp.SourceNetwork == nil || *resp.SourceNetwork != "T12345" {
			t.Errorf("expected source_network 'T12345', got %v", resp.SourceNetwork)
		}
	},
},
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && go test -v ./internal/handler/ -run TestAPIv1_CreateLink/valid_link_with_source_fields`
Expected: FAIL — `APILinkCreateResponse` has no `SourceType` field

**Step 3: Add source fields to API response types**

In `internal/handler/api_v1_types.go`, add source fields to `APILinkResponse`:

```go
type APILinkResponse struct {
	ID             int       `json:"id"`
	URL            string    `json:"url"`
	Title          string    `json:"title"`
	User           string    `json:"user"`
	Clicks         int       `json:"clicks"`
	CreatedAt      time.Time `json:"created_at"`
	Tags           []string  `json:"tags,omitempty"`
	SourceType     *string   `json:"source_type,omitempty"`
	SourceNetwork  *string   `json:"source_network,omitempty"`
	SourceChannel  *string   `json:"source_channel,omitempty"`
	SourceUserID   *string   `json:"source_user_id,omitempty"`
	SourceUserName *string   `json:"source_user_name,omitempty"`
}
```

Add source fields to `APIQuoteResponse`:

```go
type APIQuoteResponse struct {
	ID             int       `json:"id"`
	Quote          string    `json:"quote"`
	Author         string    `json:"author"`
	Poster         string    `json:"poster,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	Tags           []string  `json:"tags,omitempty"`
	SourceType     *string   `json:"source_type,omitempty"`
	SourceNetwork  *string   `json:"source_network,omitempty"`
	SourceChannel  *string   `json:"source_channel,omitempty"`
	SourceUserID   *string   `json:"source_user_id,omitempty"`
	SourceUserName *string   `json:"source_user_name,omitempty"`
}
```

**Step 4: Add source fields to request types**

In `internal/handler/api_v1_links.go`, update `APILinkCreateRequest`:

```go
type APILinkCreateRequest struct {
	URL            string   `json:"url"`
	User           string   `json:"user"`
	Tags           []string `json:"tags,omitempty"`
	SourceType     *string  `json:"source_type,omitempty"`
	SourceNetwork  *string  `json:"source_network,omitempty"`
	SourceChannel  *string  `json:"source_channel,omitempty"`
	SourceUserID   *string  `json:"source_user_id,omitempty"`
	SourceUserName *string  `json:"source_user_name,omitempty"`
}
```

In `internal/handler/api_v1_quotes.go`, update `APIQuoteCreateRequest`:

```go
type APIQuoteCreateRequest struct {
	Quote          string   `json:"quote"`
	Author         string   `json:"author"`
	Poster         string   `json:"poster"`
	Tags           []string `json:"tags,omitempty"`
	SourceType     *string  `json:"source_type,omitempty"`
	SourceNetwork  *string  `json:"source_network,omitempty"`
	SourceChannel  *string  `json:"source_channel,omitempty"`
	SourceUserID   *string  `json:"source_user_id,omitempty"`
	SourceUserName *string  `json:"source_user_name,omitempty"`
}
```

**Step 5: Update apiV1CreateLink to pass source fields through**

In `internal/handler/api_v1_links.go`, update `apiV1CreateLink` to build an `IRCLink` struct and pass source fields:

```go
// Build source filter for duplicate detection
sourceFilter := data.SourceFilter{
	SourceType:    req.SourceType,
	SourceNetwork: req.SourceNetwork,
	SourceChannel: req.SourceChannel,
}

// Check for duplicates (scoped by source when provided)
existingLinks, err := h.Store.GetIRCLinksByURL(ctx, req.URL, sourceFilter)

// Build the link struct with source metadata
link := &data.IRCLink{
	User:           req.User,
	Title:          req.URL,
	URL:            req.URL,
	ContentType:    "",
	SourceType:     req.SourceType,
	SourceNetwork:  req.SourceNetwork,
	SourceChannel:  req.SourceChannel,
	SourceUserID:   req.SourceUserID,
	SourceUserName: req.SourceUserName,
}
linkID, err := h.Store.InsertIRCLink(ctx, link)
```

Update the response builder to include source fields:

```go
resp := APILinkCreateResponse{
	APILinkResponse: APILinkResponse{
		ID:             linkID,
		URL:            req.URL,
		Title:          req.URL,
		User:           req.User,
		Clicks:         0,
		CreatedAt:      time.Now(),
		Tags:           tagStrings,
		SourceType:     req.SourceType,
		SourceNetwork:  req.SourceNetwork,
		SourceChannel:  req.SourceChannel,
		SourceUserID:   req.SourceUserID,
		SourceUserName: req.SourceUserName,
	},
	IsDuplicate:         isDuplicate,
	PreviousSubmissions: previousSubmissions,
}
```

**Step 6: Update apiV1CreateQuote similarly**

Pass source fields through to `InsertQuote` and include in response.

**Step 7: Update apiV1ListLinks to parse source query params and pass filter**

In `internal/handler/api_v1_links.go`, in `apiV1ListLinks`:

```go
// Parse source filter parameters
var sourceFilter data.SourceFilter
if st := r.URL.Query().Get("source_type"); st != "" {
	sourceFilter.SourceType = &st
}
if sn := r.URL.Query().Get("source_network"); sn != "" {
	sourceFilter.SourceNetwork = &sn
}
if sc := r.URL.Query().Get("source_channel"); sc != "" {
	sourceFilter.SourceChannel = &sc
}

links, err := h.Store.GetRecentIRCLinks(ctx, 365, 0, sourceFilter)
```

Update the response conversion loop to include source fields from each link:

```go
data = append(data, APILinkResponse{
	ID:             link.ID,
	URL:            link.URL,
	Title:          link.Title,
	User:           link.User,
	Clicks:         link.Clicks,
	CreatedAt:      link.Timestamp,
	Tags:           h.getTagStrings(ctx, "link", link.ID),
	SourceType:     link.SourceType,
	SourceNetwork:  link.SourceNetwork,
	SourceChannel:  link.SourceChannel,
	SourceUserID:   link.SourceUserID,
	SourceUserName: link.SourceUserName,
})
```

**Step 8: Update apiV1ListQuotes the same way**

Parse source query params, pass filter to `GetRecentQuotes`, include source fields in response.

**Step 9: Update apiV1GetLink and apiV1GetQuote responses**

Include source fields from the fetched link/quote in the response structs.

**Step 10: Update APIv1SearchHandler**

In `internal/handler/api_v1_search.go`, parse source query params and pass filter to `SearchIRCLinks` and `SearchQuotes`. Include source fields in the response conversion loops.

**Step 11: Run tests**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && make test`
Expected: All tests pass including the new source fields test

**Step 12: Commit**

```bash
git add internal/handler/
git commit -m "feat: accept and return source fields in API endpoints

POST links/quotes accepts optional source_type, source_network,
source_channel, source_user_id, source_user_name fields.
GET links/quotes/search supports source_type, source_network,
source_channel query parameters for filtering.
All responses include source fields with omitempty."
```

---

### Task 4: Write comprehensive tests for source-scoped duplicate detection

**Files:**
- Modify: `internal/handler/api_v1_links_test.go`

**Step 1: Write test cases for scoped duplicate detection**

Add a new test function `TestAPIv1_CreateLink_SourceDuplicates`:

```go
func TestAPIv1_CreateLink_SourceDuplicates(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name               string
		body               string
		existingLinks      []data.IRCLink
		expectedStatus     int
		expectedDuplicate  bool
	}{
		{
			name: "same URL different source is not duplicate",
			body: `{"url":"https://example.com","user":"alice","source_type":"slack","source_network":"T111","source_channel":"C222"}`,
			existingLinks: nil, // source-scoped query returns nothing
			expectedStatus: http.StatusCreated,
			expectedDuplicate: false,
		},
		{
			name: "same URL same source is duplicate",
			body: `{"url":"https://example.com","user":"bob","source_type":"slack","source_network":"T111","source_channel":"C222"}`,
			existingLinks: []data.IRCLink{
				{ID: 10, User: "alice", URL: "https://example.com", Timestamp: now},
			},
			expectedStatus: http.StatusCreated,
			expectedDuplicate: true,
		},
		{
			name: "no source fields uses global duplicate check",
			body: `{"url":"https://example.com","user":"charlie"}`,
			existingLinks: []data.IRCLink{
				{ID: 10, User: "alice", URL: "https://example.com", Timestamp: now},
			},
			expectedStatus: http.StatusCreated,
			expectedDuplicate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockAPIStore{
				linksByURL:     tt.existingLinks,
				insertedLinkID: 42,
			}
			handler := NewHandler(store, &config.Config{})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/links", strings.NewReader(tt.body))
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()

			handler.APIv1LinksHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var resp APILinkCreateResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if resp.IsDuplicate != tt.expectedDuplicate {
				t.Errorf("expected is_duplicate=%v, got %v", tt.expectedDuplicate, resp.IsDuplicate)
			}
		})
	}
}
```

**Step 2: Run tests**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && go test -v ./internal/handler/ -run TestAPIv1_CreateLink_SourceDuplicates`
Expected: PASS

**Step 3: Write test for source filtering on GET**

Add `TestAPIv1_ListLinks_SourceFiltering`:

```go
func TestAPIv1_ListLinks_SourceFiltering(t *testing.T) {
	slackType := "slack"
	slackNetwork := "T12345"
	slackChannel := "C67890"

	tests := []struct {
		name           string
		queryParams    string
		links          []data.IRCLink
		expectedTotal  int
	}{
		{
			name:        "no filter returns all",
			queryParams: "",
			links: []data.IRCLink{
				{ID: 1, User: "alice", URL: "https://a.com", Timestamp: time.Now()},
				{ID: 2, User: "bob", URL: "https://b.com", Timestamp: time.Now(), SourceType: &slackType},
			},
			expectedTotal: 2,
		},
		{
			name:        "filter by source_type",
			queryParams: "?source_type=slack",
			links: []data.IRCLink{
				{ID: 2, User: "bob", URL: "https://b.com", Timestamp: time.Now(), SourceType: &slackType},
			},
			expectedTotal: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockAPIStore{links: tt.links}
			handler := NewHandler(store, &config.Config{})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/links"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			handler.APIv1LinksHandler(w, req)

			var resp APILinksResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if resp.Meta.Total != tt.expectedTotal {
				t.Errorf("expected total %d, got %d", tt.expectedTotal, resp.Meta.Total)
			}
		})
	}
}
```

**Step 4: Write test for omitempty serialization**

Add `TestAPIv1_LinkResponse_SourceOmitEmpty`:

```go
func TestAPIv1_LinkResponse_SourceOmitEmpty(t *testing.T) {
	t.Run("null source fields omitted from JSON", func(t *testing.T) {
		store := &mockAPIStore{
			links: []data.IRCLink{
				{ID: 1, User: "alice", URL: "https://a.com", Timestamp: time.Now()},
			},
		}
		handler := NewHandler(store, &config.Config{})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/links", nil)
		w := httptest.NewRecorder()

		handler.APIv1LinksHandler(w, req)

		body := w.Body.String()
		if strings.Contains(body, "source_type") {
			t.Error("expected source_type to be omitted when null")
		}
	})

	t.Run("source fields present when set", func(t *testing.T) {
		slackType := "slack"
		store := &mockAPIStore{
			links: []data.IRCLink{
				{ID: 1, User: "alice", URL: "https://a.com", Timestamp: time.Now(), SourceType: &slackType},
			},
		}
		handler := NewHandler(store, &config.Config{})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/links", nil)
		w := httptest.NewRecorder()

		handler.APIv1LinksHandler(w, req)

		body := w.Body.String()
		if !strings.Contains(body, `"source_type":"slack"`) {
			t.Errorf("expected source_type in response, got: %s", body)
		}
	})
}
```

**Step 5: Run all tests**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && make test`
Expected: All tests pass

**Step 6: Commit**

```bash
git add internal/handler/api_v1_links_test.go
git commit -m "test: add tests for source-scoped duplicate detection and filtering"
```

---

### Task 5: Write backfill SQL script

**Files:**
- Create: `sql/backfill_sources.sql`

**Step 1: Create the backfill script**

```sql
-- One-time backfill: set source metadata on all existing rows.
-- All existing data originates from IRC, #soggies channel on jameswhite.org.
-- Run this manually after deploying the source fields migration.
--
-- Usage (SQLite):  sqlite3 tumble.db < sql/backfill_sources.sql
-- Usage (MySQL):   mysql -u user -p tumble < sql/backfill_sources.sql

UPDATE ircLink
SET source_type = 'irc',
    source_network = 'jameswhite.org',
    source_channel = '#soggies'
WHERE source_type IS NULL;

UPDATE quote
SET source_type = 'irc',
    source_network = 'jameswhite.org',
    source_channel = '#soggies'
WHERE source_type IS NULL;

UPDATE image
SET source_type = 'irc',
    source_network = 'jameswhite.org',
    source_channel = '#soggies'
WHERE source_type IS NULL;
```

**Step 2: Commit**

```bash
git add sql/backfill_sources.sql
git commit -m "feat: add one-time backfill script for source metadata"
```

---

### Task 6: Update OpenAPI specification

**Files:**
- Modify: `internal/assets/openapi.json`

**Step 1: Add source fields to LinkCreateRequest schema**

Find the `LinkCreateRequest` schema in `openapi.json` and add:

```json
"source_type": {
  "type": "string",
  "description": "Source platform (e.g., irc, slack, discord, api, web)",
  "example": "slack"
},
"source_network": {
  "type": "string",
  "description": "Source network identifier (e.g., IRC server, Slack team ID)",
  "example": "T12345"
},
"source_channel": {
  "type": "string",
  "description": "Source channel identifier (e.g., IRC channel, Slack channel ID)",
  "example": "C67890"
},
"source_user_id": {
  "type": "string",
  "description": "Platform-specific user ID",
  "example": "U99999"
},
"source_user_name": {
  "type": "string",
  "description": "Display/mention name at time of post",
  "example": "stahnma"
}
```

**Step 2: Add same fields to QuoteCreateRequest schema**

Same five fields.

**Step 3: Add source fields to APILinkResponse and APIQuoteResponse schemas**

Same five fields added to response schemas.

**Step 4: Add source query parameters to GET /api/v1/links**

Add three optional query parameters:

```json
{
  "name": "source_type",
  "in": "query",
  "required": false,
  "schema": { "type": "string" },
  "description": "Filter by source platform (e.g., irc, slack, discord)"
},
{
  "name": "source_network",
  "in": "query",
  "required": false,
  "schema": { "type": "string" },
  "description": "Filter by source network (requires source_type)"
},
{
  "name": "source_channel",
  "in": "query",
  "required": false,
  "schema": { "type": "string" },
  "description": "Filter by source channel (requires source_type and source_network)"
}
```

**Step 5: Add same query parameters to GET /api/v1/quotes and GET /api/v1/search**

**Step 6: Update 208 response description**

Update the duplicate detection documentation for POST /api/v1/links to note that duplicate detection is scoped per source when source fields are provided.

**Step 7: Run the app to verify docs render**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && make restart`
Visit: `http://localhost:8080/api/docs` and verify the new fields appear.
Then: `make kill`

**Step 8: Commit**

```bash
git add internal/assets/openapi.json
git commit -m "docs: update OpenAPI spec with source metadata fields

Add source_type, source_network, source_channel, source_user_id,
and source_user_name to request/response schemas. Add source
filter query parameters to GET endpoints."
```

---

### Task 7: Final integration verification

**Step 1: Run full test suite**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && make test`
Expected: All tests pass

**Step 2: Run API integration tests**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && make test-api`
Expected: All integration tests pass

**Step 3: Manual smoke test**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && make restart`

Test creating a link with source fields:
```bash
curl -s -X POST http://localhost:8080/api/v1/links \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/test","user":"testuser","source_type":"slack","source_network":"T12345","source_channel":"C67890","source_user_id":"U99999","source_user_name":"testuser"}' | jq .
```

Verify source fields in response. Then test filtering:
```bash
curl -s "http://localhost:8080/api/v1/links?source_type=slack" | jq .
```

Test that creating same URL without source fields still works:
```bash
curl -s -X POST http://localhost:8080/api/v1/links \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/test","user":"otheruser"}' | jq .
```

Then: `make kill`

**Step 4: Run go fmt**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && go fmt ./...`

**Step 5: Check for trailing whitespace**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && git diff --check`

**Step 6: Final commit if any formatting fixes needed**

```bash
git add -A
git commit -m "chore: format and clean up"
```
