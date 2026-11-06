package data

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestHotLinksRegression(t *testing.T) {
	// Setup in-memory SQLite DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}

	store := NewGormStore(db)
	if err := store.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Failed to bootstrap db: %v", err)
	}

	ctx := context.Background()
	now := time.Now()

	// Seed Data
	// Hot window is 6-12 days ago.
	// We use 12 for startDays and 6 for endDays in the call, meaning:
	// timestamp >= NOW - 12 days AND timestamp <= NOW - 6 days
	// Offsets use AddDate (calendar days) to match GetTopIRCLinks, so the
	// boundary cases hold across DST transitions.

	fixtures := []struct {
		Title       string
		Timestamp   time.Time
		Clicks      int
		ShouldMatch bool
	}{
		{
			Title:       "Valid Hot Link",
			Timestamp:   now.AddDate(0, 0, -8), // 8 days ago (Inside 6-12 window)
			Clicks:      10,
			ShouldMatch: true,
		},
		{
			Title:       "Too Recent Link",
			Timestamp:   now.AddDate(0, 0, -2), // 2 days ago (Too recent)
			Clicks:      10,
			ShouldMatch: false,
		},
		{
			Title:       "Too Old Link",
			Timestamp:   now.AddDate(0, 0, -15), // 15 days ago (Too old)
			Clicks:      10,
			ShouldMatch: false,
		},
		{
			Title:       "Unpopular Link",
			Timestamp:   now.AddDate(0, 0, -8), // 8 days ago (Inside window)
			Clicks:      1,                     // 1 click -> Should now appear
			ShouldMatch: true,
		},
		{
			Title:       "Zero Click Link",
			Timestamp:   now.AddDate(0, 0, -8), // 8 days ago
			Clicks:      0,                     // 0 clicks -> Should now appear
			ShouldMatch: true,
		},
		{
			Title:       "Edge Case Start (Almost 12 days)",
			Timestamp:   now.AddDate(0, 0, -12).Add(23 * time.Minute), // Just inside the 12 day window
			Clicks:      10,
			ShouldMatch: true,
		},
		{
			Title:       "Edge Case End (6 days)",
			Timestamp:   now.AddDate(0, 0, -6), // Exact boundary? Logic is <= end (which is NOW-6)
			Clicks:      10,
			ShouldMatch: true,
		},
	}

	for _, f := range fixtures {
		_, err := store.InsertIRCLink(ctx, &IRCLink{User: "tester", Title: f.Title, URL: "http://example.com/" + f.Title, ContentType: "text"})
		if err != nil {
			t.Fatalf("Failed to insert link: %v", err)
		}
		// Manually update timestamp and clicks since InsertIRCLink sets time.Now() and clicks=0
		err = db.Model(&IRCLink{}).Where("title = ?", f.Title).Updates(map[string]interface{}{
			"timestamp": f.Timestamp,
			"clicks":    f.Clicks,
		}).Error
		if err != nil {
			t.Fatalf("Failed to update link fixture: %v", err)
		}
	}

	// Test
	// Arguments: startDays=12, endDays=6, limit=5
	links, err := store.GetTopIRCLinks(ctx, 12, 6, 5)
	if err != nil {
		t.Fatalf("GetTopIRCLinks failed: %v", err)
	}

	// Verify
	foundTitles := make(map[string]bool)
	for _, l := range links {
		foundTitles[l.Title] = true
	}

	for _, f := range fixtures {
		if f.ShouldMatch {
			if !foundTitles[f.Title] {
				t.Errorf("Expected to find '%s' but did not. (Timestamp: %v, Clicks: %d)", f.Title, f.Timestamp, f.Clicks)
			}
		} else {
			if foundTitles[f.Title] {
				t.Errorf("Expected NOT to find '%s' but strictly did. (Timestamp: %v, Clicks: %d)", f.Title, f.Timestamp, f.Clicks)
			}
		}
	}
}
