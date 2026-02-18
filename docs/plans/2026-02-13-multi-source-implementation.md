# Multi-Client Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add client metadata to links, quotes, and images so Tumble can track where posts originate and scope duplicate detection per client.

**Architecture:** Add five nullable client columns to all three content tables via GORM struct changes. Modify the Store interface to accept a `ClientFilter` for queries and duplicate detection. Update API handlers to parse, pass through, and return client fields.

**Tech Stack:** Go, GORM, SQLite/MySQL, `net/http`, `httptest` for testing

---

### Task 1: Add client fields to data models

**Files:**
- Modify: `internal/data/store.go`

**Step 1: Add client fields to IRCLink struct**

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
	ClientType     *string   `json:"client_type,omitempty" gorm:"column:client_type;type:varchar(50);index:idx_client,priority:1"`
	ClientNetwork  *string   `json:"client_network,omitempty" gorm:"column:client_network;type:varchar(255);index:idx_client,priority:2"`
	ClientChannel  *string   `json:"client_channel,omitempty" gorm:"column:client_channel;type:varchar(255);index:idx_client,priority:3"`
	ClientUserID   *string   `json:"client_user_id,omitempty" gorm:"column:client_user_id;type:varchar(255)"`
	ClientUserName *string   `json:"client_user_name,omitempty" gorm:"column:client_user_name;type:varchar(255)"`
}
```

**Step 2: Add client fields to Image struct**

Same five fields added to `Image` (after `MD5Sum`, line 29):

```go
type Image struct {
	ID             int       `json:"imageID" gorm:"column:imageID;primaryKey"`
	Timestamp      time.Time `json:"timestamp" gorm:"column:timestamp"`
	Title          string    `json:"title" gorm:"column:title"`
	Link           string    `json:"link" gorm:"column:link"`
	URL            string    `json:"url" gorm:"column:url"`
	MD5Sum         string    `json:"md5sum" gorm:"column:md5sum"`
	ClientType     *string   `json:"client_type,omitempty" gorm:"column:client_type;type:varchar(50);index:idx_client,priority:1"`
	ClientNetwork  *string   `json:"client_network,omitempty" gorm:"column:client_network;type:varchar(255);index:idx_client,priority:2"`
	ClientChannel  *string   `json:"client_channel,omitempty" gorm:"column:client_channel;type:varchar(255);index:idx_client,priority:3"`
	ClientUserID   *string   `json:"client_user_id,omitempty" gorm:"column:client_user_id;type:varchar(255)"`
	ClientUserName *string   `json:"client_user_name,omitempty" gorm:"column:client_user_name;type:varchar(255)"`
}
```

**Step 3: Add client fields to Quote struct**

Same five fields added to `Quote` (after `Poster`, line 42):

```go
type Quote struct {
	ID             int       `json:"quoteID" gorm:"column:quoteID;primaryKey"`
	Timestamp      time.Time `json:"timestamp" gorm:"column:timestamp"`
	Quote          string    `json:"quote" gorm:"column:quote"`
	Author         string    `json:"author" gorm:"column:author;type:varchar(255);index"`
	Poster         string    `json:"poster,omitempty" gorm:"column:poster;type:varchar(255);index"`
	ClientType     *string   `json:"client_type,omitempty" gorm:"column:client_type;type:varchar(50);index:idx_client,priority:1"`
	ClientNetwork  *string   `json:"client_network,omitempty" gorm:"column:client_network;type:varchar(255);index:idx_client,priority:2"`
	ClientChannel  *string   `json:"client_channel,omitempty" gorm:"column:client_channel;type:varchar(255);index:idx_client,priority:3"`
	ClientUserID   *string   `json:"client_user_id,omitempty" gorm:"column:client_user_id;type:varchar(255)"`
	ClientUserName *string   `json:"client_user_name,omitempty" gorm:"column:client_user_name;type:varchar(255)"`
}
```

**Step 4: Add client fields to TimelineItem struct**

Add client fields to `TimelineItem` (after `ContentType`, line 65):

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
	ClientType     *string   `json:"client_type,omitempty"`
	ClientNetwork  *string   `json:"client_network,omitempty"`
	ClientChannel  *string   `json:"client_channel,omitempty"`
	ClientUserID   *string   `json:"client_user_id,omitempty"`
	ClientUserName *string   `json:"client_user_name,omitempty"`
}
```

**Step 5: Add ClientFilter type**

Add a new type after the `TimelineItem` struct:

```go
// ClientFilter is used to filter queries by client metadata.
// When all fields are nil, no client filtering is applied.
type ClientFilter struct {
	ClientType    *string
	ClientNetwork *string
	ClientChannel *string
}
```

**Step 6: Run tests to verify nothing broke**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && make test`
Expected: All existing tests pass (struct additions are backward compatible)

**Step 7: Commit**

```bash
git add internal/data/store.go
git commit -m "feat: add client metadata fields to data models"
```

---

### Task 2: Update Store interface and GormStore for client-aware queries

**Files:**
- Modify: `internal/data/store.go` (interface)
- Modify: `internal/data/gorm_store.go` (implementation)

**Step 1: Write failing test for client-scoped InsertIRCLink**

Create test in a new file `internal/data/client_filter_test.go`:

```go
package data

import (
	"testing"
)

func TestClientFilter_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		filter   ClientFilter
		expected bool
	}{
		{"all nil", ClientFilter{}, true},
		{"type set", ClientFilter{ClientType: strPtr("irc")}, false},
		{"network set", ClientFilter{ClientNetwork: strPtr("server")}, false},
		{"channel set", ClientFilter{ClientChannel: strPtr("#chan")}, false},
		{"all set", ClientFilter{
			ClientType:    strPtr("irc"),
			ClientNetwork: strPtr("server"),
			ClientChannel: strPtr("#chan"),
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

Run: `cd /Users/stahnma/development/personal/tumble/tumble && go test -v ./internal/data/ -run TestClientFilter`
Expected: FAIL -- `ClientFilter` has no `IsEmpty` method

**Step 3: Implement IsEmpty on ClientFilter**

In `internal/data/store.go`, add method after the `ClientFilter` struct:

```go
// IsEmpty returns true if no client filter fields are set.
func (f ClientFilter) IsEmpty() bool {
	return f.ClientType == nil && f.ClientNetwork == nil && f.ClientChannel == nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && go test -v ./internal/data/ -run TestClientFilter`
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
GetIRCLinksByURL(ctx context.Context, url string, filter ClientFilter) ([]IRCLink, error)
GetRecentIRCLinks(ctx context.Context, days int, offsetDays int, filter ClientFilter) ([]IRCLink, error)
GetRecentQuotes(ctx context.Context, days int, offsetDays int, filter ClientFilter) ([]Quote, error)
GetRecentImages(ctx context.Context, days int, offsetDays int, filter ClientFilter) ([]Image, error)
InsertQuote(ctx context.Context, quote *Quote) (int, error)
InsertImage(ctx context.Context, image *Image) (int, error)
SearchIRCLinks(ctx context.Context, query string, filter ClientFilter) ([]IRCLink, error)
SearchQuotes(ctx context.Context, query string, filter ClientFilter) ([]Quote, error)
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

**Step 9: Update GormStore.GetIRCLinksByURL for client-scoped duplicate detection**

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
func (s *GormStore) GetIRCLinksByURL(ctx context.Context, url string, filter ClientFilter) ([]IRCLink, error) {
	var links []IRCLink
	query := s.db.WithContext(ctx).Where("url = ?", url)
	query = applyClientFilter(query, filter)
	err := query.Order("timestamp DESC").Find(&links).Error
	return links, err
}
```

**Step 10: Update GormStore.GetRecentIRCLinks for client filtering**

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
func (s *GormStore) GetRecentIRCLinks(ctx context.Context, startDays int, endDays int, filter ClientFilter) ([]IRCLink, error) {
	var links []IRCLink
	now := time.Now()
	startDate := now.AddDate(0, 0, -startDays)
	endDate := now.AddDate(0, 0, -endDays)

	query := s.db.WithContext(ctx).
		Where("timestamp >= ? AND timestamp <= ?", startDate, endDate)
	query = applyClientFilter(query, filter)
	err := query.Order("timestamp DESC").Find(&links).Error
	return links, err
}
```

**Step 11: Apply same pattern to GetRecentQuotes, GetRecentImages, SearchIRCLinks, SearchQuotes**

Add `filter ClientFilter` parameter and `applyClientFilter(query, filter)` call to each. Follow exact same pattern as Step 10.

**Step 12: Add applyClientFilter helper**

Add this function to `gorm_store.go`:

```go
// applyClientFilter adds WHERE clauses for client metadata fields.
// When filter is empty, no clauses are added (backward compatible).
func applyClientFilter(query *gorm.DB, filter ClientFilter) *gorm.DB {
	if filter.ClientType != nil {
		query = query.Where("client_type = ?", *filter.ClientType)
	}
	if filter.ClientNetwork != nil {
		query = query.Where("client_network = ?", *filter.ClientNetwork)
	}
	if filter.ClientChannel != nil {
		query = query.Where("client_channel = ?", *filter.ClientChannel)
	}
	return query
}
```

**Step 13: Update all other callers of changed methods**

Search the codebase for all calls to `InsertIRCLink`, `InsertQuote`, `InsertImage`, `GetIRCLinksByURL`, `GetRecentIRCLinks`, `GetRecentQuotes`, `GetRecentImages`, `SearchIRCLinks`, `SearchQuotes`. Update each call site to pass the new parameters. For existing callers that don't have client context, pass `data.ClientFilter{}` (empty filter).

Key callers to update:
- `internal/handler/api_v1_links.go` -- `apiV1CreateLink`, `apiV1ListLinks`
- `internal/handler/api_v1_quotes.go` -- `apiV1CreateQuote`, `apiV1ListQuotes`
- `internal/handler/api_v1_search.go` -- `APIv1SearchHandler`
- `internal/handler/irclink.go` -- legacy handler (if it exists)
- `internal/handler/handlers.go` -- any frontend handlers calling these
- `internal/scheduler/` -- any background jobs
- All test mock implementations

**Step 14: Update mock stores in test files**

Update `mockAPIStore` in `internal/handler/api_v1_links_test.go` to match new signatures:

```go
func (m *mockAPIStore) GetRecentIRCLinks(ctx context.Context, days int, offsetDays int, filter data.ClientFilter) ([]data.IRCLink, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.links, nil
}

func (m *mockAPIStore) GetIRCLinksByURL(ctx context.Context, url string, filter data.ClientFilter) ([]data.IRCLink, error) {
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
git add internal/data/store.go internal/data/gorm_store.go internal/data/client_filter_test.go
git add internal/handler/
git commit -m "feat: update Store interface for client-aware queries

Change Insert methods to accept struct pointers instead of
individual parameters. Add ClientFilter to Get and Search
methods. Add applyClientFilter helper for GORM queries."
```

---

### Task 3: Update API handlers to accept and return client fields

**Files:**
- Modify: `internal/handler/api_v1_types.go`
- Modify: `internal/handler/api_v1_links.go`
- Modify: `internal/handler/api_v1_quotes.go`
- Modify: `internal/handler/api_v1_search.go`

**Step 1: Write failing test for client fields in link creation**

Add to `internal/handler/api_v1_links_test.go`, in the `TestAPIv1_CreateLink` function's test table:

```go
{
	name: "valid link with client fields",
	body: `{"url":"https://example.com","user":"testuser","client_type":"slack","client_network":"T12345","client_channel":"C67890","client_user_id":"U99999","client_user_name":"testuser"}`,
	store: &mockAPIStore{insertedLinkID: 42},
	expectedStatus: http.StatusCreated,
	checkBody: func(t *testing.T, body []byte) {
		var resp APILinkCreateResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.ClientType == nil || *resp.ClientType != "slack" {
			t.Errorf("expected client_type 'slack', got %v", resp.ClientType)
		}
		if resp.ClientNetwork == nil || *resp.ClientNetwork != "T12345" {
			t.Errorf("expected client_network 'T12345', got %v", resp.ClientNetwork)
		}
	},
},
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && go test -v ./internal/handler/ -run TestAPIv1_CreateLink/valid_link_with_client_fields`
Expected: FAIL -- `APILinkCreateResponse` has no `ClientType` field

**Step 3: Add client fields to API response types**

In `internal/handler/api_v1_types.go`, add client fields to `APILinkResponse`:

```go
type APILinkResponse struct {
	ID             int       `json:"id"`
	URL            string    `json:"url"`
	Title          string    `json:"title"`
	User           string    `json:"user"`
	Clicks         int       `json:"clicks"`
	CreatedAt      time.Time `json:"created_at"`
	Tags           []string  `json:"tags,omitempty"`
	ClientType     *string   `json:"client_type,omitempty"`
	ClientNetwork  *string   `json:"client_network,omitempty"`
	ClientChannel  *string   `json:"client_channel,omitempty"`
	ClientUserID   *string   `json:"client_user_id,omitempty"`
	ClientUserName *string   `json:"client_user_name,omitempty"`
}
```

Add client fields to `APIQuoteResponse`:

```go
type APIQuoteResponse struct {
	ID             int       `json:"id"`
	Quote          string    `json:"quote"`
	Author         string    `json:"author"`
	Poster         string    `json:"poster,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	Tags           []string  `json:"tags,omitempty"`
	ClientType     *string   `json:"client_type,omitempty"`
	ClientNetwork  *string   `json:"client_network,omitempty"`
	ClientChannel  *string   `json:"client_channel,omitempty"`
	ClientUserID   *string   `json:"client_user_id,omitempty"`
	ClientUserName *string   `json:"client_user_name,omitempty"`
}
```

**Step 4: Add client fields to request types**

In `internal/handler/api_v1_links.go`, update `APILinkCreateRequest`:

```go
type APILinkCreateRequest struct {
	URL            string   `json:"url"`
	User           string   `json:"user"`
	Tags           []string `json:"tags,omitempty"`
	ClientType     *string  `json:"client_type,omitempty"`
	ClientNetwork  *string  `json:"client_network,omitempty"`
	ClientChannel  *string  `json:"client_channel,omitempty"`
	ClientUserID   *string  `json:"client_user_id,omitempty"`
	ClientUserName *string  `json:"client_user_name,omitempty"`
}
```

In `internal/handler/api_v1_quotes.go`, update `APIQuoteCreateRequest`:

```go
type APIQuoteCreateRequest struct {
	Quote          string   `json:"quote"`
	Author         string   `json:"author"`
	Poster         string   `json:"poster"`
	Tags           []string `json:"tags,omitempty"`
	ClientType     *string  `json:"client_type,omitempty"`
	ClientNetwork  *string  `json:"client_network,omitempty"`
	ClientChannel  *string  `json:"client_channel,omitempty"`
	ClientUserID   *string  `json:"client_user_id,omitempty"`
	ClientUserName *string  `json:"client_user_name,omitempty"`
}
```

**Step 5: Update apiV1CreateLink to pass client fields through**

In `internal/handler/api_v1_links.go`, update `apiV1CreateLink` to build an `IRCLink` struct and pass client fields:

```go
// Build client filter for duplicate detection
clientFilter := data.ClientFilter{
	ClientType:    req.ClientType,
	ClientNetwork: req.ClientNetwork,
	ClientChannel: req.ClientChannel,
}

// Check for duplicates (scoped by client when provided)
existingLinks, err := h.Store.GetIRCLinksByURL(ctx, req.URL, clientFilter)

// Build the link struct with client metadata
link := &data.IRCLink{
	User:           req.User,
	Title:          req.URL,
	URL:            req.URL,
	ContentType:    "",
	ClientType:     req.ClientType,
	ClientNetwork:  req.ClientNetwork,
	ClientChannel:  req.ClientChannel,
	ClientUserID:   req.ClientUserID,
	ClientUserName: req.ClientUserName,
}
linkID, err := h.Store.InsertIRCLink(ctx, link)
```

Update the response builder to include client fields:

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
		ClientType:     req.ClientType,
		ClientNetwork:  req.ClientNetwork,
		ClientChannel:  req.ClientChannel,
		ClientUserID:   req.ClientUserID,
		ClientUserName: req.ClientUserName,
	},
	IsDuplicate:         isDuplicate,
	PreviousSubmissions: previousSubmissions,
}
```

**Step 6: Update apiV1CreateQuote similarly**

Pass client fields through to `InsertQuote` and include in response.

**Step 7: Update apiV1ListLinks to parse client query params and pass filter**

In `internal/handler/api_v1_links.go`, in `apiV1ListLinks`:

```go
// Parse client filter parameters
var clientFilter data.ClientFilter
if st := r.URL.Query().Get("client_type"); st != "" {
	clientFilter.ClientType = &st
}
if sn := r.URL.Query().Get("client_network"); sn != "" {
	clientFilter.ClientNetwork = &sn
}
if sc := r.URL.Query().Get("client_channel"); sc != "" {
	clientFilter.ClientChannel = &sc
}

links, err := h.Store.GetRecentIRCLinks(ctx, 365, 0, clientFilter)
```

Update the response conversion loop to include client fields from each link:

```go
data = append(data, APILinkResponse{
	ID:             link.ID,
	URL:            link.URL,
	Title:          link.Title,
	User:           link.User,
	Clicks:         link.Clicks,
	CreatedAt:      link.Timestamp,
	Tags:           h.getTagStrings(ctx, "link", link.ID),
	ClientType:     link.ClientType,
	ClientNetwork:  link.ClientNetwork,
	ClientChannel:  link.ClientChannel,
	ClientUserID:   link.ClientUserID,
	ClientUserName: link.ClientUserName,
})
```

**Step 8: Update apiV1ListQuotes the same way**

Parse client query params, pass filter to `GetRecentQuotes`, include client fields in response.

**Step 9: Update apiV1GetLink and apiV1GetQuote responses**

Include client fields from the fetched link/quote in the response structs.

**Step 10: Update APIv1SearchHandler**

In `internal/handler/api_v1_search.go`, parse client query params and pass filter to `SearchIRCLinks` and `SearchQuotes`. Include client fields in the response conversion loops.

**Step 11: Run tests**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && make test`
Expected: All tests pass including the new client fields test

**Step 12: Commit**

```bash
git add internal/handler/
git commit -m "feat: accept and return client fields in API endpoints

POST links/quotes accepts optional client_type, client_network,
client_channel, client_user_id, client_user_name fields.
GET links/quotes/search supports client_type, client_network,
client_channel query parameters for filtering.
All responses include client fields with omitempty."
```

---

### Task 4: Write comprehensive tests for client-scoped duplicate detection

**Files:**
- Modify: `internal/handler/api_v1_links_test.go`

**Step 1: Write test cases for scoped duplicate detection**

Add a new test function `TestAPIv1_CreateLink_ClientDuplicates`:

```go
func TestAPIv1_CreateLink_ClientDuplicates(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name               string
		body               string
		existingLinks      []data.IRCLink
		expectedStatus     int
		expectedDuplicate  bool
	}{
		{
			name: "same URL different client is not duplicate",
			body: `{"url":"https://example.com","user":"alice","client_type":"slack","client_network":"T111","client_channel":"C222"}`,
			existingLinks: nil, // client-scoped query returns nothing
			expectedStatus: http.StatusCreated,
			expectedDuplicate: false,
		},
		{
			name: "same URL same client is duplicate",
			body: `{"url":"https://example.com","user":"bob","client_type":"slack","client_network":"T111","client_channel":"C222"}`,
			existingLinks: []data.IRCLink{
				{ID: 10, User: "alice", URL: "https://example.com", Timestamp: now},
			},
			expectedStatus: http.StatusCreated,
			expectedDuplicate: true,
		},
		{
			name: "no client fields uses global duplicate check",
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

Run: `cd /Users/stahnma/development/personal/tumble/tumble && go test -v ./internal/handler/ -run TestAPIv1_CreateLink_ClientDuplicates`
Expected: PASS

**Step 3: Write test for client filtering on GET**

Add `TestAPIv1_ListLinks_ClientFiltering`:

```go
func TestAPIv1_ListLinks_ClientFiltering(t *testing.T) {
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
				{ID: 2, User: "bob", URL: "https://b.com", Timestamp: time.Now(), ClientType: &slackType},
			},
			expectedTotal: 2,
		},
		{
			name:        "filter by client_type",
			queryParams: "?client_type=slack",
			links: []data.IRCLink{
				{ID: 2, User: "bob", URL: "https://b.com", Timestamp: time.Now(), ClientType: &slackType},
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

Add `TestAPIv1_LinkResponse_ClientOmitEmpty`:

```go
func TestAPIv1_LinkResponse_ClientOmitEmpty(t *testing.T) {
	t.Run("null client fields omitted from JSON", func(t *testing.T) {
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
		if strings.Contains(body, "client_type") {
			t.Error("expected client_type to be omitted when null")
		}
	})

	t.Run("client fields present when set", func(t *testing.T) {
		slackType := "slack"
		store := &mockAPIStore{
			links: []data.IRCLink{
				{ID: 1, User: "alice", URL: "https://a.com", Timestamp: time.Now(), ClientType: &slackType},
			},
		}
		handler := NewHandler(store, &config.Config{})
		req := httptest.NewRequest(http.MethodGet, "/api/v1/links", nil)
		w := httptest.NewRecorder()

		handler.APIv1LinksHandler(w, req)

		body := w.Body.String()
		if !strings.Contains(body, `"client_type":"slack"`) {
			t.Errorf("expected client_type in response, got: %s", body)
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
git commit -m "test: add tests for client-scoped duplicate detection and filtering"
```

---

### Task 5: Write backfill SQL script

**Files:**
- Create: `sql/backfill_clients.sql`

**Step 1: Create the backfill script**

```sql
-- One-time backfill: set client metadata on all existing rows.
-- All existing data originates from IRC, #soggies channel on jameswhite.org.
-- Run this manually after deploying the client fields migration.
--
-- Usage (SQLite):  sqlite3 tumble.db < sql/backfill_clients.sql
-- Usage (MySQL):   mysql -u user -p tumble < sql/backfill_clients.sql

UPDATE ircLink
SET client_type = 'irc',
    client_network = 'jameswhite.org',
    client_channel = '#soggies'
WHERE client_type IS NULL;

UPDATE quote
SET client_type = 'irc',
    client_network = 'jameswhite.org',
    client_channel = '#soggies'
WHERE client_type IS NULL;

UPDATE image
SET client_type = 'irc',
    client_network = 'jameswhite.org',
    client_channel = '#soggies'
WHERE client_type IS NULL;
```

**Step 2: Commit**

```bash
git add sql/backfill_clients.sql
git commit -m "feat: add one-time backfill script for client metadata"
```

---

### Task 6: Update OpenAPI specification

**Files:**
- Modify: `internal/assets/openapi.json`

**Step 1: Add client fields to LinkCreateRequest schema**

Find the `LinkCreateRequest` schema in `openapi.json` and add:

```json
"client_type": {
  "type": "string",
  "description": "Client platform (e.g., irc, slack, discord, api, web)",
  "example": "slack"
},
"client_network": {
  "type": "string",
  "description": "Client network identifier (e.g., IRC server, Slack team ID)",
  "example": "T12345"
},
"client_channel": {
  "type": "string",
  "description": "Client channel identifier (e.g., IRC channel, Slack channel ID)",
  "example": "C67890"
},
"client_user_id": {
  "type": "string",
  "description": "Platform-specific user ID",
  "example": "U99999"
},
"client_user_name": {
  "type": "string",
  "description": "Display/mention name at time of post",
  "example": "stahnma"
}
```

**Step 2: Add same fields to QuoteCreateRequest schema**

Same five fields.

**Step 3: Add client fields to APILinkResponse and APIQuoteResponse schemas**

Same five fields added to response schemas.

**Step 4: Add client query parameters to GET /api/v1/links**

Add three optional query parameters:

```json
{
  "name": "client_type",
  "in": "query",
  "required": false,
  "schema": { "type": "string" },
  "description": "Filter by client platform (e.g., irc, slack, discord)"
},
{
  "name": "client_network",
  "in": "query",
  "required": false,
  "schema": { "type": "string" },
  "description": "Filter by client network (requires client_type)"
},
{
  "name": "client_channel",
  "in": "query",
  "required": false,
  "schema": { "type": "string" },
  "description": "Filter by client channel (requires client_type and client_network)"
}
```

**Step 5: Add same query parameters to GET /api/v1/quotes and GET /api/v1/search**

**Step 6: Update 208 response description**

Update the duplicate detection documentation for POST /api/v1/links to note that duplicate detection is scoped per client when client fields are provided.

**Step 7: Run the app to verify docs render**

Run: `cd /Users/stahnma/development/personal/tumble/tumble && make restart`
Visit: `http://localhost:8080/api/docs` and verify the new fields appear.
Then: `make kill`

**Step 8: Commit**

```bash
git add internal/assets/openapi.json
git commit -m "docs: update OpenAPI spec with client metadata fields

Add client_type, client_network, client_channel, client_user_id,
and client_user_name to request/response schemas. Add client
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

Test creating a link with client fields:
```bash
curl -s -X POST http://localhost:8080/api/v1/links \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/test","user":"testuser","client_type":"slack","client_network":"T12345","client_channel":"C67890","client_user_id":"U99999","client_user_name":"testuser"}' | jq .
```

Verify client fields in response. Then test filtering:
```bash
curl -s "http://localhost:8080/api/v1/links?client_type=slack" | jq .
```

Test that creating same URL without client fields still works:
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
