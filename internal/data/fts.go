package data

import (
	"context"
	"fmt"
	"strings"
)

// bootstrapFTS creates FTS5 virtual tables and triggers for SQLite.
// All statements are idempotent (IF NOT EXISTS).
func (s *GormStore) bootstrapFTS(ctx context.Context) error {
	// Check which FTS tables need to be created (and thus need initial population).
	// We check before creating because CREATE VIRTUAL TABLE IF NOT EXISTS
	// doesn't tell us whether it actually created the table.
	ftsTables := []struct {
		name    string
		content string
	}{
		{"ircLink_fts", "ircLink"},
		{"quote_fts", "quote"},
	}
	needsRebuild := make(map[string]bool)
	for _, t := range ftsTables {
		var exists int64
		if err := s.db.WithContext(ctx).Raw(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", t.name,
		).Scan(&exists).Error; err != nil {
			return fmt.Errorf("checking FTS table %s: %w", t.name, err)
		}
		needsRebuild[t.name] = (exists == 0)
	}

	statements := []string{
		// FTS5 virtual tables (external content — no data duplication)
		`CREATE VIRTUAL TABLE IF NOT EXISTS ircLink_fts USING fts5(
			title, url,
			content='ircLink',
			content_rowid='ircLinkID'
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS quote_fts USING fts5(
			quote, author,
			content='quote',
			content_rowid='quoteID'
		)`,

		// ircLink triggers
		`CREATE TRIGGER IF NOT EXISTS ircLink_fts_ai AFTER INSERT ON ircLink BEGIN
			INSERT INTO ircLink_fts(rowid, title, url) VALUES (new.ircLinkID, new.title, new.url);
		END`,
		`CREATE TRIGGER IF NOT EXISTS ircLink_fts_ad AFTER DELETE ON ircLink BEGIN
			INSERT INTO ircLink_fts(ircLink_fts, rowid, title, url) VALUES('delete', old.ircLinkID, old.title, old.url);
		END`,
		`CREATE TRIGGER IF NOT EXISTS ircLink_fts_au AFTER UPDATE ON ircLink BEGIN
			INSERT INTO ircLink_fts(ircLink_fts, rowid, title, url) VALUES('delete', old.ircLinkID, old.title, old.url);
			INSERT INTO ircLink_fts(rowid, title, url) VALUES (new.ircLinkID, new.title, new.url);
		END`,

		// quote triggers
		`CREATE TRIGGER IF NOT EXISTS quote_fts_ai AFTER INSERT ON quote BEGIN
			INSERT INTO quote_fts(rowid, quote, author) VALUES (new.quoteID, new.quote, new.author);
		END`,
		`CREATE TRIGGER IF NOT EXISTS quote_fts_ad AFTER DELETE ON quote BEGIN
			INSERT INTO quote_fts(quote_fts, rowid, quote, author) VALUES('delete', old.quoteID, old.quote, old.author);
		END`,
		`CREATE TRIGGER IF NOT EXISTS quote_fts_au AFTER UPDATE ON quote BEGIN
			INSERT INTO quote_fts(quote_fts, rowid, quote, author) VALUES('delete', old.quoteID, old.quote, old.author);
			INSERT INTO quote_fts(rowid, quote, author) VALUES (new.quoteID, new.quote, new.author);
		END`,
	}

	for _, stmt := range statements {
		if err := s.db.WithContext(ctx).Exec(stmt).Error; err != nil {
			return fmt.Errorf("FTS5 setup failed: %w", err)
		}
	}

	// Rebuild FTS indexes for any newly created tables that have existing content data.
	for _, t := range ftsTables {
		if !needsRebuild[t.name] {
			continue
		}
		var contentCount int64
		if err := s.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM " + t.content).Scan(&contentCount).Error; err != nil {
			return fmt.Errorf("counting %s: %w", t.content, err)
		}
		if contentCount == 0 {
			continue
		}
		if err := s.db.WithContext(ctx).Exec(
			"INSERT INTO " + t.name + "(" + t.name + ") VALUES('rebuild')",
		).Error; err != nil {
			return fmt.Errorf("rebuilding %s: %w", t.name, err)
		}
	}

	return nil
}

// buildFTSQuery converts user input into a safe FTS5 query string.
// Each word is double-quoted to escape FTS5 special characters.
// Multiple words use implicit AND semantics.
func buildFTSQuery(input string) string {
	words := strings.Fields(input)
	if len(words) == 0 {
		return ""
	}
	quoted := make([]string, len(words))
	for i, w := range words {
		w = strings.ReplaceAll(w, `"`, `""`)
		quoted[i] = `"` + w + `"`
	}
	return strings.Join(quoted, " ")
}
