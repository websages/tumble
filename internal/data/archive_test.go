package data

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// newTestStore creates an in-memory SQLite store for testing.
func newTestStore(t *testing.T) *GormStore {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	store := NewGormStore(db)
	if err := store.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Failed to bootstrap db: %v", err)
	}
	return store
}

func TestArchiveLookupCRUD(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Get non-existent lookup -> returns nil, nil
	lookup, err := store.GetArchiveLookup(ctx, "http://example.com/missing")
	if err != nil {
		t.Fatalf("GetArchiveLookup returned error for missing key: %v", err)
	}
	if lookup != nil {
		t.Fatalf("GetArchiveLookup should return nil for missing key, got %+v", lookup)
	}

	// Upsert a "found" lookup with archive_url and snapshot_at
	archiveURL := "https://web.archive.org/web/20240101/http://example.com/page"
	snapshotAt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	foundLookup := &ArchiveLookup{
		URL:        "http://example.com/page",
		ArchiveURL: &archiveURL,
		SnapshotAt: &snapshotAt,
		Status:     "found",
		CheckedAt:  time.Now(),
	}
	if err := store.UpsertArchiveLookup(ctx, foundLookup); err != nil {
		t.Fatalf("UpsertArchiveLookup (insert) failed: %v", err)
	}

	// Read it back and verify
	lookup, err = store.GetArchiveLookup(ctx, "http://example.com/page")
	if err != nil {
		t.Fatalf("GetArchiveLookup failed: %v", err)
	}
	if lookup == nil {
		t.Fatalf("GetArchiveLookup returned nil after upsert")
	}
	if lookup.URL != "http://example.com/page" {
		t.Errorf("Expected URL %q, got %q", "http://example.com/page", lookup.URL)
	}
	if lookup.Status != "found" {
		t.Errorf("Expected Status %q, got %q", "found", lookup.Status)
	}
	if lookup.ArchiveURL == nil || *lookup.ArchiveURL != archiveURL {
		t.Errorf("Expected ArchiveURL %q, got %v", archiveURL, lookup.ArchiveURL)
	}
	if lookup.SnapshotAt == nil || !lookup.SnapshotAt.Equal(snapshotAt) {
		t.Errorf("Expected SnapshotAt %v, got %v", snapshotAt, lookup.SnapshotAt)
	}

	// Upsert again to update same URL to "not_found"
	updatedLookup := &ArchiveLookup{
		URL:        "http://example.com/page",
		ArchiveURL: nil,
		SnapshotAt: nil,
		Status:     "not_found",
		CheckedAt:  time.Now(),
	}
	if err := store.UpsertArchiveLookup(ctx, updatedLookup); err != nil {
		t.Fatalf("UpsertArchiveLookup (update) failed: %v", err)
	}

	lookup, err = store.GetArchiveLookup(ctx, "http://example.com/page")
	if err != nil {
		t.Fatalf("GetArchiveLookup failed after update: %v", err)
	}
	if lookup == nil {
		t.Fatalf("GetArchiveLookup returned nil after update")
	}
	if lookup.Status != "not_found" {
		t.Errorf("Expected Status %q after update, got %q", "not_found", lookup.Status)
	}
}

func TestGetUncheckedDeadLinkURLs(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Insert error previews for 2 URLs (simulating dead links)
	deadURL1 := "http://example.com/dead1"
	deadURL2 := "http://example.com/dead2"
	errorData := []byte(`{"error":"status 404"}`)

	if err := store.InsertLinkPreview(ctx, deadURL1, errorData); err != nil {
		t.Fatalf("InsertLinkPreview failed: %v", err)
	}
	if err := store.InsertLinkPreview(ctx, deadURL2, errorData); err != nil {
		t.Fatalf("InsertLinkPreview failed: %v", err)
	}

	// Insert a successful preview for a live URL
	liveURL := "http://example.com/alive"
	liveData := []byte(`{"title":"Works fine"}`)
	if err := store.InsertLinkPreview(ctx, liveURL, liveData); err != nil {
		t.Fatalf("InsertLinkPreview failed: %v", err)
	}

	// Mark dead1 as already checked via UpsertArchiveLookup
	if err := store.UpsertArchiveLookup(ctx, &ArchiveLookup{
		URL:       deadURL1,
		Status:    "not_found",
		CheckedAt: time.Now(),
	}); err != nil {
		t.Fatalf("UpsertArchiveLookup failed: %v", err)
	}

	// Call GetUncheckedDeadLinkURLs -> expect only deadURL2
	urls, err := store.GetUncheckedDeadLinkURLs(ctx)
	if err != nil {
		t.Fatalf("GetUncheckedDeadLinkURLs failed: %v", err)
	}
	if len(urls) != 1 {
		t.Fatalf("Expected 1 unchecked dead URL, got %d: %v", len(urls), urls)
	}
	if urls[0] != deadURL2 {
		t.Errorf("Expected unchecked URL %q, got %q", deadURL2, urls[0])
	}
}

func TestGetStaleArchiveLookups(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	recheckAfter := 30 * 24 * time.Hour

	// Insert a stale "not_found" lookup (checked_at 31 days ago)
	staleURL := "http://example.com/stale"
	if err := store.UpsertArchiveLookup(ctx, &ArchiveLookup{
		URL:       staleURL,
		Status:    "not_found",
		CheckedAt: time.Now().Add(-31 * 24 * time.Hour),
	}); err != nil {
		t.Fatalf("UpsertArchiveLookup (stale) failed: %v", err)
	}

	// Insert a fresh "not_found" lookup (checked_at 1 day ago)
	freshURL := "http://example.com/fresh"
	if err := store.UpsertArchiveLookup(ctx, &ArchiveLookup{
		URL:       freshURL,
		Status:    "not_found",
		CheckedAt: time.Now().Add(-1 * 24 * time.Hour),
	}); err != nil {
		t.Fatalf("UpsertArchiveLookup (fresh) failed: %v", err)
	}

	// Insert a "found" lookup with old checked_at (should never appear)
	foundURL := "http://example.com/found-old"
	archiveURL := "https://web.archive.org/web/20230101/http://example.com/found-old"
	if err := store.UpsertArchiveLookup(ctx, &ArchiveLookup{
		URL:        foundURL,
		ArchiveURL: &archiveURL,
		Status:     "found",
		CheckedAt:  time.Now().Add(-60 * 24 * time.Hour),
	}); err != nil {
		t.Fatalf("UpsertArchiveLookup (found-old) failed: %v", err)
	}

	// Call GetStaleArchiveLookups for not_found with 30-day recheck -> expect only staleURL
	urls, err := store.GetStaleArchiveLookups(ctx, "not_found", recheckAfter)
	if err != nil {
		t.Fatalf("GetStaleArchiveLookups failed: %v", err)
	}
	if len(urls) != 1 {
		t.Fatalf("Expected 1 stale URL, got %d: %v", len(urls), urls)
	}
	if urls[0] != staleURL {
		t.Errorf("Expected stale URL %q, got %q", staleURL, urls[0])
	}
}
