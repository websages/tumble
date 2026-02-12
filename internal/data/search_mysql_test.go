//go:build mysql

package data

import (
	"context"
	"os"
	"testing"
	"time"

	"tumble/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// newMySQLTestStore creates a GormStore backed by a real MySQL instance.
// Reads connection info from conf/config-test-mysql.yaml.
// Set MYSQL_TEST_DSN to override the DSN entirely.
func newMySQLTestStore(t *testing.T) *GormStore {
	t.Helper()
	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		cfg, err := config.Load("../../conf/config-test-mysql.yaml")
		if err != nil {
			t.Fatalf("Failed to load MySQL test config: %v", err)
		}
		dsn = cfg.DSN()
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to MySQL: %v", err)
	}
	store := NewGormStore(db)
	if err := store.Bootstrap(context.Background()); err != nil {
		t.Fatalf("Failed to bootstrap MySQL db: %v", err)
	}
	t.Cleanup(func() {
		db.Exec("DELETE FROM tags")
		db.Exec("DELETE FROM link_previews")
		db.Exec("DELETE FROM quote")
		db.Exec("DELETE FROM ircLink")
	})
	return store
}

// --------------- SearchIRCLinks MySQL tests ---------------

func TestSearchIRCLinks_ByTitle_MySQL(t *testing.T) {
	store := newMySQLTestStore(t)
	ctx := context.Background()

	_, err := store.InsertIRCLink(ctx, "alice", "Golang Tutorial", "http://example.com/go-mysql", "text/html")
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}
	_, err = store.InsertIRCLink(ctx, "bob", "Rust Guide", "http://example.com/rust-mysql", "text/html")
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	links, err := store.SearchIRCLinks(ctx, "Golang")
	if err != nil {
		t.Fatalf("SearchIRCLinks failed: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 result, got %d", len(links))
	}
	if links[0].Title != "Golang Tutorial" {
		t.Errorf("expected title 'Golang Tutorial', got %q", links[0].Title)
	}
}

func TestSearchIRCLinks_ByURL_MySQL(t *testing.T) {
	store := newMySQLTestStore(t)
	ctx := context.Background()

	_, err := store.InsertIRCLink(ctx, "alice", "Some Page", "http://example.com/unique-mysql-path", "text/html")
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	links, err := store.SearchIRCLinks(ctx, "unique-mysql-path")
	if err != nil {
		t.Fatalf("SearchIRCLinks failed: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 result, got %d", len(links))
	}
	if links[0].URL != "http://example.com/unique-mysql-path" {
		t.Errorf("expected URL with 'unique-mysql-path', got %q", links[0].URL)
	}
}

func TestSearchIRCLinks_ByTag_MySQL(t *testing.T) {
	store := newMySQLTestStore(t)
	ctx := context.Background()

	id, err := store.InsertIRCLink(ctx, "alice", "Tagged Link", "http://example.com/tagged-mysql", "text/html")
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	_, err = store.CreateTag(ctx, Tag{
		Tag:          "special-mysql-topic",
		ResourceType: "link",
		ResourceID:   id,
		CreatedBy:    "alice",
	})
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	links, err := store.SearchIRCLinks(ctx, "special-mysql-topic")
	if err != nil {
		t.Fatalf("SearchIRCLinks failed: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 result, got %d", len(links))
	}
	if links[0].Title != "Tagged Link" {
		t.Errorf("expected 'Tagged Link', got %q", links[0].Title)
	}
}

func TestSearchIRCLinks_NoMatch_MySQL(t *testing.T) {
	store := newMySQLTestStore(t)
	ctx := context.Background()

	_, err := store.InsertIRCLink(ctx, "alice", "Something", "http://example.com/a-mysql", "text/html")
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	links, err := store.SearchIRCLinks(ctx, "nonexistent-xyzzy-mysql")
	if err != nil {
		t.Fatalf("SearchIRCLinks failed: %v", err)
	}
	if len(links) != 0 {
		t.Errorf("expected 0 results, got %d", len(links))
	}
}

func TestSearchIRCLinks_OrderedByClicks_MySQL(t *testing.T) {
	store := newMySQLTestStore(t)
	ctx := context.Background()
	db := store.db

	_, err := store.InsertIRCLink(ctx, "alice", "SearchM Low", "http://example.com/searchm-low", "text/html")
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}
	_, err = store.InsertIRCLink(ctx, "bob", "SearchM High", "http://example.com/searchm-high", "text/html")
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	db.Model(&IRCLink{}).Where("title = ?", "SearchM Low").Update("clicks", 5)
	db.Model(&IRCLink{}).Where("title = ?", "SearchM High").Update("clicks", 50)

	links, err := store.SearchIRCLinks(ctx, "SearchM")
	if err != nil {
		t.Fatalf("SearchIRCLinks failed: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 results, got %d", len(links))
	}
	if links[0].Title != "SearchM High" {
		t.Errorf("expected first result 'SearchM High' (most clicks), got %q", links[0].Title)
	}
	if links[1].Title != "SearchM Low" {
		t.Errorf("expected second result 'SearchM Low', got %q", links[1].Title)
	}
}

func TestSearchIRCLinks_ExcludesErrorPreviews_MySQL(t *testing.T) {
	store := newMySQLTestStore(t)
	ctx := context.Background()

	_, err := store.InsertIRCLink(ctx, "alice", "Good Link", "http://example.com/good-mysql", "text/html")
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}
	_, err = store.InsertIRCLink(ctx, "bob", "Bad Link", "http://example.com/bad-mysql", "text/html")
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	errorData := []byte(`{"error":"status 404"}`)
	if err := store.InsertLinkPreview(ctx, "http://example.com/bad-mysql", errorData); err != nil {
		t.Fatalf("InsertLinkPreview failed: %v", err)
	}

	links, err := store.SearchIRCLinks(ctx, "Link")
	if err != nil {
		t.Fatalf("SearchIRCLinks failed: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 result (error link excluded), got %d", len(links))
	}
	if links[0].Title != "Good Link" {
		t.Errorf("expected 'Good Link', got %q", links[0].Title)
	}
}

func TestSearchIRCLinks_ExpiredErrorCacheIncluded_MySQL(t *testing.T) {
	store := newMySQLTestStore(t)
	ctx := context.Background()
	db := store.db

	_, err := store.InsertIRCLink(ctx, "alice", "Recoverable Link", "http://example.com/recover-mysql", "text/html")
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	errorData := []byte(`{"error":"status 503"}`)
	if err := store.InsertLinkPreview(ctx, "http://example.com/recover-mysql", errorData); err != nil {
		t.Fatalf("InsertLinkPreview failed: %v", err)
	}
	twoDaysAgo := time.Now().Add(-48 * time.Hour)
	db.Model(&LinkPreview{}).Where("url = ?", "http://example.com/recover-mysql").Update("updated_at", twoDaysAgo)

	links, err := store.SearchIRCLinks(ctx, "Recoverable")
	if err != nil {
		t.Fatalf("SearchIRCLinks failed: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 result (expired error cache should be included), got %d", len(links))
	}
	if links[0].Title != "Recoverable Link" {
		t.Errorf("expected 'Recoverable Link', got %q", links[0].Title)
	}
}

// --------------- SearchQuotes MySQL tests ---------------

func TestSearchQuotes_ByQuoteText_MySQL(t *testing.T) {
	store := newMySQLTestStore(t)
	ctx := context.Background()

	_, err := store.InsertQuote(ctx, "To be or not to be", "Shakespeare", "alice")
	if err != nil {
		t.Fatalf("InsertQuote failed: %v", err)
	}
	_, err = store.InsertQuote(ctx, "I think therefore I am", "Descartes", "bob")
	if err != nil {
		t.Fatalf("InsertQuote failed: %v", err)
	}

	quotes, err := store.SearchQuotes(ctx, "not to be")
	if err != nil {
		t.Fatalf("SearchQuotes failed: %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("expected 1 result, got %d", len(quotes))
	}
	if quotes[0].Author != "Shakespeare" {
		t.Errorf("expected author 'Shakespeare', got %q", quotes[0].Author)
	}
}

func TestSearchQuotes_ByAuthor_MySQL(t *testing.T) {
	store := newMySQLTestStore(t)
	ctx := context.Background()

	_, err := store.InsertQuote(ctx, "Some quote", "UniqueAuthor42", "alice")
	if err != nil {
		t.Fatalf("InsertQuote failed: %v", err)
	}

	quotes, err := store.SearchQuotes(ctx, "UniqueAuthor42")
	if err != nil {
		t.Fatalf("SearchQuotes failed: %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("expected 1 result, got %d", len(quotes))
	}
	if quotes[0].Quote != "Some quote" {
		t.Errorf("expected quote 'Some quote', got %q", quotes[0].Quote)
	}
}

func TestSearchQuotes_ByTag_MySQL(t *testing.T) {
	store := newMySQLTestStore(t)
	ctx := context.Background()

	id, err := store.InsertQuote(ctx, "A tagged quote", "someone", "alice")
	if err != nil {
		t.Fatalf("InsertQuote failed: %v", err)
	}

	_, err = store.CreateTag(ctx, Tag{
		Tag:          "philosophy",
		ResourceType: "quote",
		ResourceID:   id,
		CreatedBy:    "alice",
	})
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	quotes, err := store.SearchQuotes(ctx, "philosophy")
	if err != nil {
		t.Fatalf("SearchQuotes failed: %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("expected 1 result, got %d", len(quotes))
	}
	if quotes[0].Quote != "A tagged quote" {
		t.Errorf("expected 'A tagged quote', got %q", quotes[0].Quote)
	}
}

func TestSearchQuotes_NoMatch_MySQL(t *testing.T) {
	store := newMySQLTestStore(t)
	ctx := context.Background()

	_, err := store.InsertQuote(ctx, "Hello world", "author1", "poster1")
	if err != nil {
		t.Fatalf("InsertQuote failed: %v", err)
	}

	quotes, err := store.SearchQuotes(ctx, "nonexistent-xyzzy-mysql")
	if err != nil {
		t.Fatalf("SearchQuotes failed: %v", err)
	}
	if len(quotes) != 0 {
		t.Errorf("expected 0 results, got %d", len(quotes))
	}
}

func TestSearchQuotes_OrderedByTimestamp_MySQL(t *testing.T) {
	store := newMySQLTestStore(t)
	ctx := context.Background()
	db := store.db

	_, err := store.InsertQuote(ctx, "SearchM older quote", "auth1", "poster")
	if err != nil {
		t.Fatalf("InsertQuote failed: %v", err)
	}
	_, err = store.InsertQuote(ctx, "SearchM newer quote", "auth2", "poster")
	if err != nil {
		t.Fatalf("InsertQuote failed: %v", err)
	}

	now := time.Now()
	db.Model(&Quote{}).Where("quote = ?", "SearchM older quote").Update("timestamp", now.Add(-48*time.Hour))
	db.Model(&Quote{}).Where("quote = ?", "SearchM newer quote").Update("timestamp", now.Add(-1*time.Hour))

	quotes, err := store.SearchQuotes(ctx, "SearchM")
	if err != nil {
		t.Fatalf("SearchQuotes failed: %v", err)
	}
	if len(quotes) != 2 {
		t.Fatalf("expected 2 results, got %d", len(quotes))
	}
	if quotes[0].Quote != "SearchM newer quote" {
		t.Errorf("expected first result 'SearchM newer quote' (most recent), got %q", quotes[0].Quote)
	}
	if quotes[1].Quote != "SearchM older quote" {
		t.Errorf("expected second result 'SearchM older quote', got %q", quotes[1].Quote)
	}
}
