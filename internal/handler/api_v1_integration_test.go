package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tumble/internal/config"
	"tumble/internal/data"
)

// integrationMockStore is a comprehensive mock implementation of data.Store
// for integration tests that need multiple handler types to work together.
type integrationMockStore struct {
	data.Store

	// Links
	links          []data.IRCLink
	linkByID       *data.IRCLink
	insertedLinkID int

	// Quotes
	quotes    []data.Quote
	quoteByID *data.Quote

	// Stats
	userStats []data.UserStat

	// Search
	searchLinks  []data.IRCLink
	searchQuotes []data.Quote

	// Error injection
	err error
}

func (m *integrationMockStore) GetRecentIRCLinks(ctx context.Context, days int, offsetDays int, filter data.ClientFilter) ([]data.IRCLink, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.links, nil
}

func (m *integrationMockStore) GetIRCLinkByID(ctx context.Context, id int) (*data.IRCLink, error) {
	if m.err != nil {
		return nil, m.err
	}
	// Find link by ID in links slice
	for _, link := range m.links {
		if link.ID == id {
			return &link, nil
		}
	}
	return m.linkByID, nil
}

func (m *integrationMockStore) GetIRCLinksByURL(ctx context.Context, url string, filter data.ClientFilter) ([]data.IRCLink, error) {
	if m.err != nil {
		return nil, m.err
	}
	return nil, nil // No duplicates by default
}

func (m *integrationMockStore) InsertIRCLink(ctx context.Context, link *data.IRCLink) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.insertedLinkID, nil
}

func (m *integrationMockStore) DeleteIRCLink(ctx context.Context, id int) error {
	return m.err
}

func (m *integrationMockStore) GetRecentQuotes(ctx context.Context, days int, offsetDays int, filter data.ClientFilter) ([]data.Quote, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.quotes, nil
}

func (m *integrationMockStore) GetQuoteByID(ctx context.Context, id int) (*data.Quote, error) {
	if m.err != nil {
		return nil, m.err
	}
	// Find quote by ID in quotes slice
	for _, quote := range m.quotes {
		if quote.ID == id {
			return &quote, nil
		}
	}
	return m.quoteByID, nil
}

func (m *integrationMockStore) InsertQuote(ctx context.Context, quote *data.Quote) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return 1, nil
}

func (m *integrationMockStore) DeleteQuote(ctx context.Context, id int) error {
	return m.err
}

func (m *integrationMockStore) CountIRCLinks(ctx context.Context) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return int64(len(m.links)), nil
}

func (m *integrationMockStore) CountQuotes(ctx context.Context) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return int64(len(m.quotes)), nil
}

func (m *integrationMockStore) GetUserStats(ctx context.Context, sortBy string, limit int, offset int) ([]data.UserStat, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.userStats, nil
}

func (m *integrationMockStore) SearchIRCLinks(ctx context.Context, query string, filter data.ClientFilter) ([]data.IRCLink, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.searchLinks, nil
}

func (m *integrationMockStore) SearchQuotes(ctx context.Context, query string, filter data.ClientFilter) ([]data.Quote, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.searchQuotes, nil
}

func (m *integrationMockStore) CreateTag(ctx context.Context, tag data.Tag) (*data.Tag, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &tag, nil
}

func (m *integrationMockStore) GetTagsByResource(ctx context.Context, resourceType string, resourceID int) ([]data.Tag, error) {
	if m.err != nil {
		return nil, m.err
	}
	return nil, nil
}

func (m *integrationMockStore) GetTagByID(ctx context.Context, id int) (*data.Tag, error) {
	if m.err != nil {
		return nil, m.err
	}
	return nil, nil
}

func (m *integrationMockStore) DeleteTag(ctx context.Context, id int) error {
	return m.err
}

func (m *integrationMockStore) DeleteTagsByResource(ctx context.Context, resourceType string, resourceID int) error {
	return m.err
}

// setupIntegrationTest creates a handler and mux with all v1 routes registered.
func setupIntegrationTest(store *integrationMockStore) (*http.ServeMux, *Handler) {
	cfg := &config.Config{
		BaseURL:     "http://localhost:8080",
		AdminSecret: "test-secret-key",
	}

	h := &Handler{
		Store:  store,
		Config: cfg,
	}

	mux := http.NewServeMux()

	// Register all v1 routes (mirrors main.go registration)
	mux.HandleFunc("/api/v1/links", h.APIv1LinksHandler)
	mux.HandleFunc("/api/v1/links/", h.APIv1LinksHandler)
	mux.HandleFunc("/api/v1/quotes", h.APIv1QuotesHandler)
	mux.HandleFunc("/api/v1/quotes/", h.APIv1QuotesHandler)
	mux.HandleFunc("/api/v1/stats", h.APIv1StatsHandler)
	mux.HandleFunc("/api/v1/search", h.APIv1SearchHandler)

	return mux, h
}

// TestIntegration_LinksEndToEnd tests the full links workflow.
func TestIntegration_LinksEndToEnd(t *testing.T) {
	now := time.Now()

	store := &integrationMockStore{
		links: []data.IRCLink{
			{ID: 1, Timestamp: now, User: "alice", Title: "Example Site", URL: "https://example.com", Clicks: 10},
			{ID: 2, Timestamp: now.Add(-time.Hour), User: "bob", Title: "Another Site", URL: "https://example.org", Clicks: 5},
		},
		insertedLinkID: 3,
	}

	mux, _ := setupIntegrationTest(store)

	t.Run("GET /api/v1/links returns paginated list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/links", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}

		var resp APILinksResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(resp.Data) != 2 {
			t.Errorf("expected 2 links, got %d", len(resp.Data))
		}
		if resp.Meta.Total != 2 {
			t.Errorf("expected total 2, got %d", resp.Meta.Total)
		}
		if resp.Meta.Limit != 50 {
			t.Errorf("expected default limit 50, got %d", resp.Meta.Limit)
		}
	})

	t.Run("GET /api/v1/links/{id} returns single link", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/links/1", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp APILinkResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.ID != 1 {
			t.Errorf("expected ID 1, got %d", resp.ID)
		}
		if resp.Title != "Example Site" {
			t.Errorf("expected title 'Example Site', got %s", resp.Title)
		}
	})

	t.Run("POST /api/v1/links creates link and returns 201", func(t *testing.T) {
		body := `{"url": "https://newsite.com", "user": "charlie"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/links", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp APILinkCreateResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.ID != 3 {
			t.Errorf("expected ID 3, got %d", resp.ID)
		}
		if resp.User != "charlie" {
			t.Errorf("expected user 'charlie', got %s", resp.User)
		}
	})

	t.Run("GET /api/v1/links/999 returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/links/999", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}

		var resp APIErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal error response: %v", err)
		}

		if resp.Error.Code != "not_found" {
			t.Errorf("expected error code 'not_found', got %s", resp.Error.Code)
		}
	})
}

// TestIntegration_ContentNegotiation tests format suffix and Accept header handling.
func TestIntegration_ContentNegotiation(t *testing.T) {
	now := time.Now()

	store := &integrationMockStore{
		links: []data.IRCLink{
			{ID: 1, Timestamp: now, User: "alice", Title: "Test Site", URL: "https://test.com", Clicks: 5},
		},
	}

	mux, _ := setupIntegrationTest(store)

	t.Run(".json suffix works", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/links/1.json", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}

		var resp APILinkResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.ID != 1 {
			t.Errorf("expected ID 1, got %d", resp.ID)
		}
	})

	t.Run("Accept header application/json works", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/links/1", nil)
		req.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run(".txt suffix returns plain text", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/links/1.txt", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "text/plain" {
			t.Errorf("expected Content-Type text/plain, got %s", contentType)
		}

		body := w.Body.String()
		if body != "Test Site - https://test.com" {
			t.Errorf("unexpected plain text body: %s", body)
		}
	})

	t.Run("Accept text/plain returns plain text", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/links/1", nil)
		req.Header.Set("Accept", "text/plain")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "text/plain" {
			t.Errorf("expected Content-Type text/plain, got %s", contentType)
		}
	})
}

// TestIntegration_Authentication tests that protected endpoints require auth.
func TestIntegration_Authentication(t *testing.T) {
	now := time.Now()

	store := &integrationMockStore{
		links: []data.IRCLink{
			{ID: 1, Timestamp: now, User: "alice", Title: "Test Site", URL: "https://test.com", Clicks: 5},
		},
	}

	mux, _ := setupIntegrationTest(store)

	t.Run("DELETE without API key returns 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/links/1", nil)
		// Simulate a non-localhost request
		req.RemoteAddr = "192.168.1.100:54321"
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d", w.Code)
		}

		var resp APIErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal error response: %v", err)
		}

		if resp.Error.Code != "forbidden" {
			t.Errorf("expected error code 'forbidden', got %s", resp.Error.Code)
		}
		if resp.Error.Message != "Invalid or missing API key" {
			t.Errorf("expected message 'Invalid or missing API key', got %s", resp.Error.Message)
		}
	})

	t.Run("DELETE with wrong API key returns 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/links/1", nil)
		req.RemoteAddr = "192.168.1.100:54321"
		req.Header.Set("X-API-Key", "wrong-key")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d", w.Code)
		}
	})

	t.Run("DELETE with correct API key returns 204", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/links/1", nil)
		req.RemoteAddr = "192.168.1.100:54321"
		req.Header.Set("X-API-Key", "test-secret-key")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("expected status 204, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("DELETE from localhost is allowed without API key", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/links/1", nil)
		req.RemoteAddr = "127.0.0.1:54321"
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("expected status 204, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// TestIntegration_ErrorStructure tests that errors have consistent structure.
func TestIntegration_ErrorStructure(t *testing.T) {
	store := &integrationMockStore{}
	mux, _ := setupIntegrationTest(store)

	t.Run("invalid ID returns proper error structure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/links/invalid", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}

		var resp APIErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal error response: %v", err)
		}

		// Verify error structure has required fields
		if resp.Error.Code == "" {
			t.Error("expected error.code to be non-empty")
		}
		if resp.Error.Message == "" {
			t.Error("expected error.message to be non-empty")
		}
	})

	t.Run("validation error returns proper structure", func(t *testing.T) {
		body := `{"user": "charlie"}` // Missing required url
		req := httptest.NewRequest(http.MethodPost, "/api/v1/links", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status 422, got %d: %s", w.Code, w.Body.String())
		}

		var resp ValidationErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal validation error: %v", err)
		}

		if resp.Error.Code != "validation_error" {
			t.Errorf("expected error code 'validation_error', got %s", resp.Error.Code)
		}
		if resp.Error.Details["url"] == "" {
			t.Error("expected details to contain 'url' field error")
		}
	})

	t.Run("method not allowed returns proper error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/links", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected status 405, got %d", w.Code)
		}

		var resp APIErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal error response: %v", err)
		}

		if resp.Error.Code != "method_not_allowed" {
			t.Errorf("expected error code 'method_not_allowed', got %s", resp.Error.Code)
		}
	})
}

// TestIntegration_SearchWithTypeFilter tests search with type filtering.
func TestIntegration_SearchWithTypeFilter(t *testing.T) {
	now := time.Now()

	store := &integrationMockStore{
		searchLinks: []data.IRCLink{
			{ID: 1, Timestamp: now, User: "alice", Title: "Go Tutorial", URL: "https://go.dev", Clicks: 100},
		},
		searchQuotes: []data.Quote{
			{ID: 2, Timestamp: now, Quote: "Go is great", Author: "pike", Poster: "bob"},
		},
	}

	mux, _ := setupIntegrationTest(store)

	t.Run("search returns both types by default", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=golang", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp APISearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(resp.Links) != 1 {
			t.Errorf("expected 1 link, got %d", len(resp.Links))
		}
		if len(resp.Quotes) != 1 {
			t.Errorf("expected 1 quote, got %d", len(resp.Quotes))
		}
		if resp.Meta.TotalLinks != 1 {
			t.Errorf("expected TotalLinks 1, got %d", resp.Meta.TotalLinks)
		}
		if resp.Meta.TotalQuotes != 1 {
			t.Errorf("expected TotalQuotes 1, got %d", resp.Meta.TotalQuotes)
		}
	})

	t.Run("type=links returns only links", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=golang&type=links", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp APISearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(resp.Links) != 1 {
			t.Errorf("expected 1 link, got %d", len(resp.Links))
		}
		if len(resp.Quotes) != 0 {
			t.Errorf("expected 0 quotes, got %d", len(resp.Quotes))
		}
	})

	t.Run("search requires minimum query length", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=go", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected status 422, got %d", w.Code)
		}

		var resp ValidationErrorResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal error: %v", err)
		}

		if resp.Error.Details["q"] == "" {
			t.Error("expected validation error for 'q' field")
		}
	})
}

// TestIntegration_QuotesEndToEnd tests the full quotes workflow.
func TestIntegration_QuotesEndToEnd(t *testing.T) {
	now := time.Now()

	store := &integrationMockStore{
		quotes: []data.Quote{
			{ID: 1, Timestamp: now, Quote: "Hello World", Author: "test", Poster: "poster1"},
		},
	}

	mux, _ := setupIntegrationTest(store)

	t.Run("GET /api/v1/quotes returns list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/quotes", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp APIQuotesResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(resp.Data) != 1 {
			t.Errorf("expected 1 quote, got %d", len(resp.Data))
		}
	})

	t.Run("GET /api/v1/quotes/{id} returns single quote", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/quotes/1", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp APIQuoteResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.ID != 1 {
			t.Errorf("expected ID 1, got %d", resp.ID)
		}
		if resp.Quote != "Hello World" {
			t.Errorf("expected quote 'Hello World', got %s", resp.Quote)
		}
	})

	t.Run("POST /api/v1/quotes creates quote", func(t *testing.T) {
		body := `{"quote": "New quote", "author": "newauthor"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/quotes", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp APIQuoteResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp.Quote != "New quote" {
			t.Errorf("expected quote 'New quote', got %s", resp.Quote)
		}
	})
}

// TestIntegration_StatsEndpoint tests the stats endpoint.
func TestIntegration_StatsEndpoint(t *testing.T) {
	store := &integrationMockStore{
		userStats: []data.UserStat{
			{User: "alice", LinkCount: 100, QuoteCount: 50},
			{User: "bob", LinkCount: 75, QuoteCount: 25},
		},
	}

	mux, _ := setupIntegrationTest(store)

	t.Run("GET /api/v1/stats returns stats", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp APIStatsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(resp.Leaderboard) != 2 {
			t.Errorf("expected 2 users in leaderboard, got %d", len(resp.Leaderboard))
		}
		if resp.Leaderboard[0].User != "alice" {
			t.Errorf("expected first user 'alice', got %s", resp.Leaderboard[0].User)
		}
	})
}
