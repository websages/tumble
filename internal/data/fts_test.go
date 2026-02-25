package data

import (
	"context"
	"testing"
)

func TestBuildFTSQuery(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", `"hello" "world"`},
		{"", ""},
		{"single", `"single"`},
		{`has "quotes" inside`, `"has" """quotes""" "inside"`},
		{"special*chars(here)", `"special*chars(here)"`},
		{"  extra   spaces  ", `"extra" "spaces"`},
		{"UPPER lower MiXeD", `"UPPER" "lower" "MiXeD"`},
	}

	for _, tt := range tests {
		result := buildFTSQuery(tt.input)
		if result != tt.expected {
			t.Errorf("buildFTSQuery(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFTS5_BootstrapCreatesTables(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Verify FTS tables exist by querying them
	var count int64
	err := store.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM ircLink_fts`).Scan(&count).Error
	if err != nil {
		t.Fatalf("ircLink_fts table should exist after Bootstrap: %v", err)
	}

	err = store.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM quote_fts`).Scan(&count).Error
	if err != nil {
		t.Fatalf("quote_fts table should exist after Bootstrap: %v", err)
	}
}

func TestFTS5_TriggersPopulateOnInsert(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.InsertIRCLink(ctx, &IRCLink{
		User: "alice", Title: "FTS Trigger Test", URL: "http://fts.example.com", ContentType: "text/html",
	})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	var count int64
	err = store.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM ircLink_fts WHERE ircLink_fts MATCH '"Trigger"'`).Scan(&count).Error
	if err != nil {
		t.Fatalf("FTS query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 FTS match after insert, got %d", count)
	}
}

func TestFTS5_TriggersRemoveOnDelete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	id, err := store.InsertIRCLink(ctx, &IRCLink{
		User: "alice", Title: "Deletable Link", URL: "http://delete.example.com", ContentType: "text/html",
	})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	if err := store.DeleteIRCLink(ctx, id); err != nil {
		t.Fatalf("DeleteIRCLink failed: %v", err)
	}

	var count int64
	err = store.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM ircLink_fts WHERE ircLink_fts MATCH '"Deletable"'`).Scan(&count).Error
	if err != nil {
		t.Fatalf("FTS query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 FTS matches after delete, got %d", count)
	}
}

func TestFTS5_QuoteTriggersPopulateOnInsert(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.InsertQuote(ctx, &Quote{Quote: "FTS quote trigger test", Author: "tester", Poster: "poster"})
	if err != nil {
		t.Fatalf("InsertQuote failed: %v", err)
	}

	var count int64
	err = store.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM quote_fts WHERE quote_fts MATCH '"trigger"'`).Scan(&count).Error
	if err != nil {
		t.Fatalf("FTS query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 FTS match after insert, got %d", count)
	}
}

func TestFTS5_WordBoundaryMatching(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.InsertIRCLink(ctx, &IRCLink{
		User: "alice", Title: "Golang Tutorial Guide", URL: "http://example.com/golang", ContentType: "text/html",
	})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	// Whole word matches
	links, err := store.SearchIRCLinks(ctx, "Golang", ClientFilter{})
	if err != nil {
		t.Fatalf("SearchIRCLinks failed: %v", err)
	}
	if len(links) != 1 {
		t.Errorf("expected 1 result for whole word 'Golang', got %d", len(links))
	}

	// Substring should NOT match with FTS5
	links, err = store.SearchIRCLinks(ctx, "olan", ClientFilter{})
	if err != nil {
		t.Fatalf("SearchIRCLinks failed: %v", err)
	}
	if len(links) != 0 {
		t.Errorf("expected 0 results for substring 'olan' with FTS5, got %d", len(links))
	}
}

func TestFTS5_MultiWordSearch(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.InsertIRCLink(ctx, &IRCLink{
		User: "alice", Title: "Golang Tutorial Guide", URL: "http://example.com/go", ContentType: "text/html",
	})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}
	_, err = store.InsertIRCLink(ctx, &IRCLink{
		User: "bob", Title: "Rust Tutorial", URL: "http://example.com/rust", ContentType: "text/html",
	})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	// Multi-word AND: both words must appear
	links, err := store.SearchIRCLinks(ctx, "Golang Tutorial", ClientFilter{})
	if err != nil {
		t.Fatalf("SearchIRCLinks failed: %v", err)
	}
	if len(links) != 1 {
		t.Errorf("expected 1 result for 'Golang Tutorial', got %d", len(links))
	}
	if len(links) > 0 && links[0].Title != "Golang Tutorial Guide" {
		t.Errorf("expected 'Golang Tutorial Guide', got %q", links[0].Title)
	}
}

func TestFTS5_RebuildFromExistingData(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Insert data (triggers will populate FTS)
	_, err := store.InsertIRCLink(ctx, &IRCLink{
		User: "alice", Title: "Pre-existing Link", URL: "http://example.com/old", ContentType: "text/html",
	})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	// Drop FTS tables and triggers to simulate a pre-FTS database
	store.db.Exec("DROP TABLE IF EXISTS ircLink_fts")
	store.db.Exec("DROP TABLE IF EXISTS quote_fts")
	store.db.Exec("DROP TRIGGER IF EXISTS ircLink_fts_ai")
	store.db.Exec("DROP TRIGGER IF EXISTS ircLink_fts_ad")
	store.db.Exec("DROP TRIGGER IF EXISTS ircLink_fts_au")
	store.db.Exec("DROP TRIGGER IF EXISTS quote_fts_ai")
	store.db.Exec("DROP TRIGGER IF EXISTS quote_fts_ad")
	store.db.Exec("DROP TRIGGER IF EXISTS quote_fts_au")

	// Re-run bootstrapFTS — should detect missing tables and rebuild from existing data
	if err := store.bootstrapFTS(ctx); err != nil {
		t.Fatalf("bootstrapFTS failed: %v", err)
	}

	// Verify FTS index was populated from existing content
	var count int64
	store.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM ircLink_fts WHERE ircLink_fts MATCH '"Pre-existing"'`).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 FTS match after rebuild, got %d", count)
	}
}
