package data

import (
	"context"
	"testing"
	"time"
)

// --------------- SearchIRCLinks tests ---------------

func TestSearchIRCLinks_ByTitle(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.InsertIRCLink(ctx, &IRCLink{User: "alice", Title: "Golang Tutorial", URL: "http://example.com/go", ContentType: "text/html"})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}
	_, err = store.InsertIRCLink(ctx, &IRCLink{User: "bob", Title: "Rust Guide", URL: "http://example.com/rust", ContentType: "text/html"})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	links, err := store.SearchIRCLinks(ctx, "Golang", ClientFilter{})
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

func TestSearchIRCLinks_ByURL(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.InsertIRCLink(ctx, &IRCLink{User: "alice", Title: "Some Page", URL: "http://example.com/unique-path", ContentType: "text/html"})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	links, err := store.SearchIRCLinks(ctx, "unique-path", ClientFilter{})
	if err != nil {
		t.Fatalf("SearchIRCLinks failed: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 result, got %d", len(links))
	}
	if links[0].URL != "http://example.com/unique-path" {
		t.Errorf("expected URL with 'unique-path', got %q", links[0].URL)
	}
}

func TestSearchIRCLinks_ByTag(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	id, err := store.InsertIRCLink(ctx, &IRCLink{User: "alice", Title: "Tagged Link", URL: "http://example.com/tagged", ContentType: "text/html"})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	_, err = store.CreateTag(ctx, Tag{
		Tag:          "special-topic",
		ResourceType: "link",
		ResourceID:   id,
		CreatedBy:    "alice",
	})
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}

	links, err := store.SearchIRCLinks(ctx, "special-topic", ClientFilter{})
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

func TestSearchIRCLinks_NoMatch(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.InsertIRCLink(ctx, &IRCLink{User: "alice", Title: "Something", URL: "http://example.com/a", ContentType: "text/html"})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	links, err := store.SearchIRCLinks(ctx, "nonexistent-xyzzy", ClientFilter{})
	if err != nil {
		t.Fatalf("SearchIRCLinks failed: %v", err)
	}
	if len(links) != 0 {
		t.Errorf("expected 0 results, got %d", len(links))
	}
}

func TestSearchIRCLinks_OrderedByClicks(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	db := store.db

	_, err := store.InsertIRCLink(ctx, &IRCLink{User: "alice", Title: "Search Low", URL: "http://example.com/search-low", ContentType: "text/html"})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}
	_, err = store.InsertIRCLink(ctx, &IRCLink{User: "bob", Title: "Search High", URL: "http://example.com/search-high", ContentType: "text/html"})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	// Set different click counts
	db.Model(&IRCLink{}).Where("title = ?", "Search Low").Update("clicks", 5)
	db.Model(&IRCLink{}).Where("title = ?", "Search High").Update("clicks", 50)

	links, err := store.SearchIRCLinks(ctx, "Search", ClientFilter{})
	if err != nil {
		t.Fatalf("SearchIRCLinks failed: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("expected 2 results, got %d", len(links))
	}
	if links[0].Title != "Search High" {
		t.Errorf("expected first result 'Search High' (most clicks), got %q", links[0].Title)
	}
	if links[1].Title != "Search Low" {
		t.Errorf("expected second result 'Search Low', got %q", links[1].Title)
	}
}

func TestSearchIRCLinks_ExcludesErrorPreviews(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Insert two links — one will have an error preview, one won't
	_, err := store.InsertIRCLink(ctx, &IRCLink{User: "alice", Title: "Good Link", URL: "http://example.com/good", ContentType: "text/html"})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}
	_, err = store.InsertIRCLink(ctx, &IRCLink{User: "bob", Title: "Bad Link", URL: "http://example.com/bad", ContentType: "text/html"})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	// Insert an error preview for the bad link (recent, so it's within the cache TTL)
	errorData := []byte(`{"error":"status 404"}`)
	if err := store.InsertLinkPreview(ctx, "http://example.com/bad", errorData); err != nil {
		t.Fatalf("InsertLinkPreview failed: %v", err)
	}

	links, err := store.SearchIRCLinks(ctx, "Link", ClientFilter{})
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

func TestSearchIRCLinks_ExpiredErrorCacheIncluded(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	db := store.db

	// Insert link that is "recent" (< 10 days old) — its error cache TTL is 24h
	_, err := store.InsertIRCLink(ctx, &IRCLink{User: "alice", Title: "Recoverable Link", URL: "http://example.com/recover", ContentType: "text/html"})
	if err != nil {
		t.Fatalf("InsertIRCLink failed: %v", err)
	}

	// Insert an error preview and backdate its updated_at to >24h ago
	errorData := []byte(`{"error":"status 503"}`)
	if err := store.InsertLinkPreview(ctx, "http://example.com/recover", errorData); err != nil {
		t.Fatalf("InsertLinkPreview failed: %v", err)
	}
	// Backdate the preview's updated_at to 2 days ago (beyond the 24h TTL for recent links)
	twoDaysAgo := time.Now().Add(-48 * time.Hour)
	db.Model(&LinkPreview{}).Where("url = ?", "http://example.com/recover").Update("updated_at", twoDaysAgo)

	links, err := store.SearchIRCLinks(ctx, "Recoverable", ClientFilter{})
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

// --------------- SearchQuotes tests ---------------

func TestSearchQuotes_ByQuoteText(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.InsertQuote(ctx, &Quote{Quote: "To be or not to be", Author: "Shakespeare", Poster: "alice"})
	if err != nil {
		t.Fatalf("InsertQuote failed: %v", err)
	}
	_, err = store.InsertQuote(ctx, &Quote{Quote: "I think therefore I am", Author: "Descartes", Poster: "bob"})
	if err != nil {
		t.Fatalf("InsertQuote failed: %v", err)
	}

	quotes, err := store.SearchQuotes(ctx, "not to be", ClientFilter{})
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

func TestSearchQuotes_ByAuthor(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.InsertQuote(ctx, &Quote{Quote: "Some quote", Author: "UniqueAuthor42", Poster: "alice"})
	if err != nil {
		t.Fatalf("InsertQuote failed: %v", err)
	}

	quotes, err := store.SearchQuotes(ctx, "UniqueAuthor42", ClientFilter{})
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

func TestSearchQuotes_ByTag(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	id, err := store.InsertQuote(ctx, &Quote{Quote: "A tagged quote", Author: "someone", Poster: "alice"})
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

	quotes, err := store.SearchQuotes(ctx, "philosophy", ClientFilter{})
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

func TestSearchQuotes_NoMatch(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.InsertQuote(ctx, &Quote{Quote: "Hello world", Author: "author1", Poster: "poster1"})
	if err != nil {
		t.Fatalf("InsertQuote failed: %v", err)
	}

	quotes, err := store.SearchQuotes(ctx, "nonexistent-xyzzy", ClientFilter{})
	if err != nil {
		t.Fatalf("SearchQuotes failed: %v", err)
	}
	if len(quotes) != 0 {
		t.Errorf("expected 0 results, got %d", len(quotes))
	}
}

func TestSearchQuotes_OrderedByTimestamp(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	db := store.db

	_, err := store.InsertQuote(ctx, &Quote{Quote: "Search older quote", Author: "auth1", Poster: "poster"})
	if err != nil {
		t.Fatalf("InsertQuote failed: %v", err)
	}
	_, err = store.InsertQuote(ctx, &Quote{Quote: "Search newer quote", Author: "auth2", Poster: "poster"})
	if err != nil {
		t.Fatalf("InsertQuote failed: %v", err)
	}

	// Set timestamps: newer quote gets a more recent time
	now := time.Now()
	db.Model(&Quote{}).Where("quote = ?", "Search older quote").Update("timestamp", now.Add(-48*time.Hour))
	db.Model(&Quote{}).Where("quote = ?", "Search newer quote").Update("timestamp", now.Add(-1*time.Hour))

	quotes, err := store.SearchQuotes(ctx, "Search", ClientFilter{})
	if err != nil {
		t.Fatalf("SearchQuotes failed: %v", err)
	}
	if len(quotes) != 2 {
		t.Fatalf("expected 2 results, got %d", len(quotes))
	}
	if quotes[0].Quote != "Search newer quote" {
		t.Errorf("expected first result 'Search newer quote' (most recent), got %q", quotes[0].Quote)
	}
	if quotes[1].Quote != "Search older quote" {
		t.Errorf("expected second result 'Search older quote', got %q", quotes[1].Quote)
	}
}
