# Daily Kitten Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a scheduled daily kitten image from cataas.com at 10 AM Central.

**Architecture:** A new `internal/scheduler` package handles cron scheduling and the daily cat fetch logic. On startup, it checks if today's kitten exists; if not, fetches immediately. Uses `robfig/cron/v3` for timezone-aware scheduling.

**Tech Stack:** Go, robfig/cron/v3, GORM, existing Image model

---

## Task 1: Add cron dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add the cron library**

Run:
```bash
cd /Users/stahnma/development/personal/tumble/daily-kitten && go get github.com/robfig/cron/v3
```

**Step 2: Verify dependency added**

Run:
```bash
grep robfig go.mod
```

Expected: Line containing `github.com/robfig/cron/v3`

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add robfig/cron for scheduling"
```

---

## Task 2: Add InsertImage to Store interface and implementation

**Files:**
- Modify: `internal/data/store.go:80-113`
- Modify: `internal/data/gorm_store.go`
- Create: `internal/data/image_test.go`

**Step 1: Write the failing test**

Create `internal/data/image_test.go`:

```go
package data

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestInsertImage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	store := NewGormStore(db)
	if err := store.Bootstrap(context.Background()); err != nil {
		t.Fatalf("failed to bootstrap: %v", err)
	}

	// Insert an image
	id, err := store.InsertImage(context.Background(), "Daily Kitten", "cat AAS", "https://cataas.com/cat/abc123")
	if err != nil {
		t.Fatalf("InsertImage failed: %v", err)
	}

	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}

	// Verify it's in the database via GetRecentImages
	images, err := store.GetRecentImages(context.Background(), 1, 0)
	if err != nil {
		t.Fatalf("GetRecentImages failed: %v", err)
	}

	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}

	img := images[0]
	if img.Title != "Daily Kitten" {
		t.Errorf("expected title 'Daily Kitten', got '%s'", img.Title)
	}
	if img.Link != "cat AAS" {
		t.Errorf("expected link 'cat AAS', got '%s'", img.Link)
	}
	if img.URL != "https://cataas.com/cat/abc123" {
		t.Errorf("expected URL 'https://cataas.com/cat/abc123', got '%s'", img.URL)
	}
}

func TestGetTodayImageByLink(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	store := NewGormStore(db)
	if err := store.Bootstrap(context.Background()); err != nil {
		t.Fatalf("failed to bootstrap: %v", err)
	}

	// No image yet
	exists, err := store.GetTodayImageByLink(context.Background(), "cat AAS")
	if err != nil {
		t.Fatalf("GetTodayImageByLink failed: %v", err)
	}
	if exists != nil {
		t.Error("expected nil when no image exists")
	}

	// Insert an image
	_, err = store.InsertImage(context.Background(), "Daily Kitten", "cat AAS", "https://cataas.com/cat/abc123")
	if err != nil {
		t.Fatalf("InsertImage failed: %v", err)
	}

	// Now it should exist
	exists, err = store.GetTodayImageByLink(context.Background(), "cat AAS")
	if err != nil {
		t.Fatalf("GetTodayImageByLink failed: %v", err)
	}
	if exists == nil {
		t.Error("expected image to exist")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/stahnma/development/personal/tumble/daily-kitten && go test ./internal/data/... -run TestInsertImage -v
```

Expected: FAIL with compilation error about undefined InsertImage

**Step 3: Add methods to Store interface**

In `internal/data/store.go`, add these methods to the Store interface (around line 97, before Bootstrap):

```go
	// Image operations
	InsertImage(ctx context.Context, title, link, url string) (int, error)
	GetTodayImageByLink(ctx context.Context, link string) (*Image, error)
```

**Step 4: Implement methods in GormStore**

Add to `internal/data/gorm_store.go` (after GetRecentImages, around line 58):

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

func (s *GormStore) GetTodayImageByLink(ctx context.Context, link string) (*Image, error) {
	var img Image
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	err := s.db.WithContext(ctx).
		Where("link = ? AND timestamp >= ? AND timestamp < ?", link, startOfDay, endOfDay).
		First(&img).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &img, nil
}
```

**Step 5: Run tests to verify they pass**

Run:
```bash
cd /Users/stahnma/development/personal/tumble/daily-kitten && go test ./internal/data/... -v
```

Expected: All tests PASS

**Step 6: Commit**

```bash
git add internal/data/
git commit -m "feat(data): add InsertImage and GetTodayImageByLink methods"
```

---

## Task 3: Create scheduler package with cat fetcher

**Files:**
- Create: `internal/scheduler/scheduler.go`
- Create: `internal/scheduler/dailycat.go`
- Create: `internal/scheduler/dailycat_test.go`

**Step 1: Write the failing test for cat URL fetching**

Create `internal/scheduler/dailycat_test.go`:

```go
package scheduler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchCatURL_Redirect(t *testing.T) {
	// Mock server that redirects
	redirectURL := "https://cataas.com/cat/abc123.jpg"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectURL, http.StatusFound)
	}))
	defer server.Close()

	url, err := fetchCatURL(server.URL)
	if err != nil {
		t.Fatalf("fetchCatURL failed: %v", err)
	}

	if url != redirectURL {
		t.Errorf("expected %s, got %s", redirectURL, url)
	}
}

func TestFetchCatURL_DirectImage(t *testing.T) {
	// Mock server that returns image directly (no redirect)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake image data"))
	}))
	defer server.Close()

	url, err := fetchCatURL(server.URL)
	if err != nil {
		t.Fatalf("fetchCatURL failed: %v", err)
	}

	// Should return the original URL since no redirect
	if url != server.URL {
		t.Errorf("expected %s, got %s", server.URL, url)
	}
}

func TestFetchCatURL_Error(t *testing.T) {
	// Mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := fetchCatURL(server.URL)
	if err == nil {
		t.Error("expected error for 500 response")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
cd /Users/stahnma/development/personal/tumble/daily-kitten && go test ./internal/scheduler/... -v
```

Expected: FAIL with compilation error about undefined fetchCatURL

**Step 3: Create dailycat.go with fetch logic**

Create `internal/scheduler/dailycat.go`:

```go
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"tumble/internal/data"
)

const (
	cataasURL    = "https://cataas.com/cat"
	catAASUser   = "cat AAS"
	kittenTitle  = "Daily Kitten"
)

// fetchCatURL fetches the cataas URL and returns the redirect location.
// If no redirect occurs, returns the original URL.
func fetchCatURL(url string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch cat URL: %w", err)
	}
	defer resp.Body.Close()

	// Check for redirect
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		if location != "" {
			return location, nil
		}
	}

	// If no redirect, return original URL if successful
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return url, nil
	}

	return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// FetchAndStoreDailyCat fetches a cat image and stores it in the database.
// Returns true if a new cat was stored, false if today's cat already exists.
func FetchAndStoreDailyCat(ctx context.Context, store data.Store) (bool, error) {
	// Check if today's cat already exists
	existing, err := store.GetTodayImageByLink(ctx, catAASUser)
	if err != nil {
		return false, fmt.Errorf("failed to check for existing cat: %w", err)
	}
	if existing != nil {
		slog.Info("Daily kitten already exists for today", "imageID", existing.ID)
		return false, nil
	}

	// Fetch the cat URL
	catURL, err := fetchCatURL(cataasURL)
	if err != nil {
		return false, fmt.Errorf("failed to fetch cat URL: %w", err)
	}

	// Store the image
	id, err := store.InsertImage(ctx, kittenTitle, catAASUser, catURL)
	if err != nil {
		return false, fmt.Errorf("failed to insert cat image: %w", err)
	}

	slog.Info("Daily kitten fetched and stored", "imageID", id, "url", catURL)
	return true, nil
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
cd /Users/stahnma/development/personal/tumble/daily-kitten && go test ./internal/scheduler/... -v
```

Expected: All tests PASS

**Step 5: Commit**

```bash
git add internal/scheduler/
git commit -m "feat(scheduler): add daily cat fetch logic"
```

---

## Task 4: Add scheduler with cron and retry logic

**Files:**
- Create: `internal/scheduler/scheduler.go`
- Modify: `internal/scheduler/dailycat_test.go`

**Step 1: Create scheduler.go**

Create `internal/scheduler/scheduler.go`:

```go
package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"tumble/internal/data"
)

const (
	maxRetries     = 3
	retryInterval  = 1 * time.Hour
)

// Scheduler manages scheduled tasks for the application.
type Scheduler struct {
	cron       *cron.Cron
	store      data.Store
	retryCount int
	retryMu    sync.Mutex
	stopRetry  chan struct{}
}

// New creates a new Scheduler with the given store.
func New(store data.Store) *Scheduler {
	// Use America/Chicago for Central Time
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		slog.Warn("Failed to load America/Chicago timezone, using local time", "error", err)
		loc = time.Local
	}

	return &Scheduler{
		cron:      cron.New(cron.WithLocation(loc)),
		store:     store,
		stopRetry: make(chan struct{}),
	}
}

// Start begins the scheduler and runs any startup tasks.
func (s *Scheduler) Start(ctx context.Context) error {
	// Schedule daily cat at 10 AM Central
	_, err := s.cron.AddFunc("0 10 * * *", func() {
		s.fetchDailyCatWithRetry(ctx)
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	slog.Info("Scheduler started", "nextRun", s.cron.Entries()[0].Next)

	// Check if we need to fetch today's cat on startup
	go s.checkStartupCat(ctx)

	return nil
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	close(s.stopRetry)
	ctx := s.cron.Stop()
	<-ctx.Done()
	slog.Info("Scheduler stopped")
}

// checkStartupCat checks if today's cat needs to be fetched on startup.
func (s *Scheduler) checkStartupCat(ctx context.Context) {
	existing, err := s.store.GetTodayImageByLink(ctx, catAASUser)
	if err != nil {
		slog.Warn("Failed to check for existing daily cat on startup", "error", err)
		return
	}

	if existing == nil {
		slog.Info("No daily kitten for today, fetching now")
		s.fetchDailyCatWithRetry(ctx)
	} else {
		slog.Info("Daily kitten already exists", "imageID", existing.ID)
	}
}

// fetchDailyCatWithRetry attempts to fetch the daily cat with retry logic.
func (s *Scheduler) fetchDailyCatWithRetry(ctx context.Context) {
	s.retryMu.Lock()
	s.retryCount = 0
	s.retryMu.Unlock()

	s.attemptFetch(ctx)
}

func (s *Scheduler) attemptFetch(ctx context.Context) {
	stored, err := FetchAndStoreDailyCat(ctx, s.store)
	if err != nil {
		s.retryMu.Lock()
		s.retryCount++
		count := s.retryCount
		s.retryMu.Unlock()

		if count < maxRetries {
			slog.Warn("Failed to fetch daily kitten, will retry",
				"error", err,
				"attempt", count,
				"nextRetry", time.Now().Add(retryInterval))

			// Schedule retry
			go func() {
				select {
				case <-time.After(retryInterval):
					s.attemptFetch(ctx)
				case <-s.stopRetry:
					return
				case <-ctx.Done():
					return
				}
			}()
		} else {
			slog.Warn("Failed to fetch daily kitten after all retries",
				"error", err,
				"attempts", count)
		}
		return
	}

	if stored {
		slog.Info("Daily kitten successfully fetched")
	}
}
```

**Step 2: Run all tests**

Run:
```bash
cd /Users/stahnma/development/personal/tumble/daily-kitten && go test ./internal/scheduler/... -v
```

Expected: All tests PASS

**Step 3: Commit**

```bash
git add internal/scheduler/
git commit -m "feat(scheduler): add cron scheduler with retry logic"
```

---

## Task 5: Integrate scheduler into main.go

**Files:**
- Modify: `cmd/tumble/main.go`

**Step 1: Add scheduler import and initialization**

In `cmd/tumble/main.go`, add the import:

```go
"tumble/internal/scheduler"
```

**Step 2: Initialize scheduler after store creation**

After the store initialization (around line 198, after `defer store.Close()`), add:

```go
	// Init Scheduler
	sched := scheduler.New(store)
	if err := sched.Start(context.Background()); err != nil {
		slog.Error("Failed to start scheduler", "error", err)
		os.Exit(1)
	}
	defer sched.Stop()
```

**Step 3: Run make build to verify compilation**

Run:
```bash
cd /Users/stahnma/development/personal/tumble/daily-kitten && make build
```

Expected: Build succeeds

**Step 4: Run all tests**

Run:
```bash
cd /Users/stahnma/development/personal/tumble/daily-kitten && make test
```

Expected: All tests PASS

**Step 5: Commit**

```bash
git add cmd/tumble/main.go
git commit -m "feat: integrate daily kitten scheduler into main"
```

---

## Task 6: Verify end-to-end functionality

**Step 1: Start the application**

Run:
```bash
cd /Users/stahnma/development/personal/tumble/daily-kitten && make restart
```

**Step 2: Check logs for scheduler startup**

Run:
```bash
tail -20 /Users/stahnma/development/personal/tumble/daily-kitten/tumble.log | grep -i "scheduler\|kitten"
```

Expected: Log lines showing scheduler started and daily kitten fetch (or already exists)

**Step 3: Stop the application**

Run:
```bash
cd /Users/stahnma/development/personal/tumble/daily-kitten && make kill
```

**Step 4: Final commit for any cleanup**

If any changes were needed:
```bash
git add -A
git commit -m "fix: address integration issues"
```

---

## Summary of Changes

1. **New dependency:** `github.com/robfig/cron/v3`
2. **New Store methods:** `InsertImage`, `GetTodayImageByLink`
3. **New package:** `internal/scheduler/` with:
   - `scheduler.go` - Cron management, startup check, retry logic
   - `dailycat.go` - Cat URL fetching and storage
   - `dailycat_test.go` - Unit tests
4. **Modified:** `cmd/tumble/main.go` - Scheduler integration

The daily kitten appears in the feed as a regular image entry with:
- Title: "Daily Kitten"
- Link: "cat AAS" (stored in the link field, not displayed in templates)
- URL: The captured redirect URL from cataas.com
