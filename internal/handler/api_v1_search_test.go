package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tumble/internal/config"
	"tumble/internal/data"
)

// mockSearchStore is a mock implementation of data.Store for testing search API handlers.
type mockSearchStore struct {
	data.Store
	links          []data.IRCLink
	quotes         []data.Quote
	searchLinksFn  func(query string) ([]data.IRCLink, error)
	searchQuotesFn func(query string) ([]data.Quote, error)
	err            error
}

func (m *mockSearchStore) SearchIRCLinks(ctx context.Context, query string, filter data.SourceFilter) ([]data.IRCLink, error) {
	if m.searchLinksFn != nil {
		return m.searchLinksFn(query)
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.links, nil
}

func (m *mockSearchStore) SearchQuotes(ctx context.Context, query string, filter data.SourceFilter) ([]data.Quote, error) {
	if m.searchQuotesFn != nil {
		return m.searchQuotesFn(query)
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.quotes, nil
}

func TestAPIv1_Search(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		path           string
		links          []data.IRCLink
		quotes         []data.Quote
		storeErr       error
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name: "search returns both links and quotes",
			path: "/api/v1/search?q=test",
			links: []data.IRCLink{
				{ID: 1, Timestamp: now, User: "alice", Title: "Test Link", URL: "https://example.com/test", Clicks: 5},
			},
			quotes: []data.Quote{
				{ID: 2, Timestamp: now, Quote: "Test quote", Author: "bob", Poster: "charlie"},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APISearchResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Links) != 1 {
					t.Errorf("expected 1 link, got %d", len(resp.Links))
				}
				if len(resp.Quotes) != 1 {
					t.Errorf("expected 1 quote, got %d", len(resp.Quotes))
				}
				if resp.Meta.TotalLinks != 1 {
					t.Errorf("expected total_links 1, got %d", resp.Meta.TotalLinks)
				}
				if resp.Meta.TotalQuotes != 1 {
					t.Errorf("expected total_quotes 1, got %d", resp.Meta.TotalQuotes)
				}
				if resp.Meta.Limit != 50 {
					t.Errorf("expected limit 50, got %d", resp.Meta.Limit)
				}
				if resp.Meta.Offset != 0 {
					t.Errorf("expected offset 0, got %d", resp.Meta.Offset)
				}

				// Check link structure
				link := resp.Links[0]
				if link.ID != 1 {
					t.Errorf("expected link ID 1, got %d", link.ID)
				}
				if link.User != "alice" {
					t.Errorf("expected user alice, got %s", link.User)
				}
				if link.Title != "Test Link" {
					t.Errorf("expected title 'Test Link', got %s", link.Title)
				}

				// Check quote structure
				quote := resp.Quotes[0]
				if quote.ID != 2 {
					t.Errorf("expected quote ID 2, got %d", quote.ID)
				}
				if quote.Author != "bob" {
					t.Errorf("expected author bob, got %s", quote.Author)
				}
				if quote.Poster != "charlie" {
					t.Errorf("expected poster charlie, got %s", quote.Poster)
				}
			},
		},
		{
			name: "type=links returns only links",
			path: "/api/v1/search?q=test&type=links",
			links: []data.IRCLink{
				{ID: 1, Timestamp: now, User: "alice", Title: "Test Link", URL: "https://example.com/test"},
			},
			quotes: []data.Quote{
				{ID: 2, Timestamp: now, Quote: "Test quote", Author: "bob"},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APISearchResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Links) != 1 {
					t.Errorf("expected 1 link, got %d", len(resp.Links))
				}
				if len(resp.Quotes) != 0 {
					t.Errorf("expected 0 quotes, got %d", len(resp.Quotes))
				}
				if resp.Meta.TotalLinks != 1 {
					t.Errorf("expected total_links 1, got %d", resp.Meta.TotalLinks)
				}
				if resp.Meta.TotalQuotes != 0 {
					t.Errorf("expected total_quotes 0, got %d", resp.Meta.TotalQuotes)
				}
			},
		},
		{
			name: "type=quotes returns only quotes",
			path: "/api/v1/search?q=test&type=quotes",
			links: []data.IRCLink{
				{ID: 1, Timestamp: now, User: "alice", Title: "Test Link", URL: "https://example.com/test"},
			},
			quotes: []data.Quote{
				{ID: 2, Timestamp: now, Quote: "Test quote", Author: "bob"},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APISearchResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Links) != 0 {
					t.Errorf("expected 0 links, got %d", len(resp.Links))
				}
				if len(resp.Quotes) != 1 {
					t.Errorf("expected 1 quote, got %d", len(resp.Quotes))
				}
				if resp.Meta.TotalLinks != 0 {
					t.Errorf("expected total_links 0, got %d", resp.Meta.TotalLinks)
				}
				if resp.Meta.TotalQuotes != 1 {
					t.Errorf("expected total_quotes 1, got %d", resp.Meta.TotalQuotes)
				}
			},
		},
		{
			name:           "missing q returns 422",
			path:           "/api/v1/search",
			expectedStatus: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body []byte) {
				var resp ValidationErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "validation_error" {
					t.Errorf("expected code validation_error, got %s", resp.Error.Code)
				}
				if resp.Error.Details["q"] == "" {
					t.Errorf("expected q validation error")
				}
			},
		},
		{
			name:           "q too short returns 422",
			path:           "/api/v1/search?q=abc",
			expectedStatus: http.StatusUnprocessableEntity,
			checkBody: func(t *testing.T, body []byte) {
				var resp ValidationErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "validation_error" {
					t.Errorf("expected code validation_error, got %s", resp.Error.Code)
				}
				if resp.Error.Details["q"] == "" {
					t.Errorf("expected q validation error")
				}
			},
		},
		{
			name:           "q exactly 4 characters is valid",
			path:           "/api/v1/search?q=test",
			links:          []data.IRCLink{},
			quotes:         []data.Quote{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APISearchResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Links) != 0 {
					t.Errorf("expected 0 links, got %d", len(resp.Links))
				}
				if len(resp.Quotes) != 0 {
					t.Errorf("expected 0 quotes, got %d", len(resp.Quotes))
				}
			},
		},
		{
			name: "respects limit parameter",
			path: "/api/v1/search?q=test&limit=1",
			links: []data.IRCLink{
				{ID: 1, Timestamp: now, User: "alice", Title: "Link 1", URL: "https://example.com/1"},
				{ID: 2, Timestamp: now, User: "bob", Title: "Link 2", URL: "https://example.com/2"},
			},
			quotes: []data.Quote{
				{ID: 3, Timestamp: now, Quote: "Quote 1", Author: "charlie"},
				{ID: 4, Timestamp: now, Quote: "Quote 2", Author: "david"},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APISearchResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Links) != 1 {
					t.Errorf("expected 1 link, got %d", len(resp.Links))
				}
				if len(resp.Quotes) != 1 {
					t.Errorf("expected 1 quote, got %d", len(resp.Quotes))
				}
				if resp.Meta.Limit != 1 {
					t.Errorf("expected limit 1, got %d", resp.Meta.Limit)
				}
				// Total should reflect all results, not just paginated
				if resp.Meta.TotalLinks != 2 {
					t.Errorf("expected total_links 2, got %d", resp.Meta.TotalLinks)
				}
				if resp.Meta.TotalQuotes != 2 {
					t.Errorf("expected total_quotes 2, got %d", resp.Meta.TotalQuotes)
				}
			},
		},
		{
			name: "respects offset parameter",
			path: "/api/v1/search?q=test&offset=1",
			links: []data.IRCLink{
				{ID: 1, Timestamp: now, User: "alice", Title: "Link 1", URL: "https://example.com/1"},
				{ID: 2, Timestamp: now, User: "bob", Title: "Link 2", URL: "https://example.com/2"},
			},
			quotes: []data.Quote{
				{ID: 3, Timestamp: now, Quote: "Quote 1", Author: "charlie"},
				{ID: 4, Timestamp: now, Quote: "Quote 2", Author: "david"},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APISearchResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Links) != 1 {
					t.Errorf("expected 1 link, got %d", len(resp.Links))
				}
				if len(resp.Quotes) != 1 {
					t.Errorf("expected 1 quote, got %d", len(resp.Quotes))
				}
				if resp.Meta.Offset != 1 {
					t.Errorf("expected offset 1, got %d", resp.Meta.Offset)
				}
				// First link should be the second one due to offset
				if len(resp.Links) > 0 && resp.Links[0].ID != 2 {
					t.Errorf("expected first link ID 2, got %d", resp.Links[0].ID)
				}
				// First quote should be the second one due to offset
				if len(resp.Quotes) > 0 && resp.Quotes[0].ID != 4 {
					t.Errorf("expected first quote ID 4, got %d", resp.Quotes[0].ID)
				}
			},
		},
		{
			name: "type=links,quotes returns both",
			path: "/api/v1/search?q=test&type=links,quotes",
			links: []data.IRCLink{
				{ID: 1, Timestamp: now, User: "alice", Title: "Test Link", URL: "https://example.com/test"},
			},
			quotes: []data.Quote{
				{ID: 2, Timestamp: now, Quote: "Test quote", Author: "bob"},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APISearchResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Links) != 1 {
					t.Errorf("expected 1 link, got %d", len(resp.Links))
				}
				if len(resp.Quotes) != 1 {
					t.Errorf("expected 1 quote, got %d", len(resp.Quotes))
				}
			},
		},
		{
			name:           "method not allowed for POST",
			path:           "/api/v1/search?q=test",
			links:          []data.IRCLink{},
			quotes:         []data.Quote{},
			expectedStatus: http.StatusMethodNotAllowed,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "method_not_allowed" {
					t.Errorf("expected code method_not_allowed, got %s", resp.Error.Code)
				}
			},
		},
		{
			name: "works with .json suffix",
			path: "/api/v1/search.json?q=test",
			links: []data.IRCLink{
				{ID: 1, Timestamp: now, User: "alice", Title: "Test Link", URL: "https://example.com/test"},
			},
			quotes:         []data.Quote{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APISearchResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Links) != 1 {
					t.Errorf("expected 1 link, got %d", len(resp.Links))
				}
			},
		},
		{
			name:           "limit capped at 1000",
			path:           "/api/v1/search?q=test&limit=5000",
			links:          []data.IRCLink{},
			quotes:         []data.Quote{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APISearchResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Meta.Limit != 1000 {
					t.Errorf("expected limit capped at 1000, got %d", resp.Meta.Limit)
				}
			},
		},
		{
			name:           "invalid limit uses default",
			path:           "/api/v1/search?q=test&limit=abc",
			links:          []data.IRCLink{},
			quotes:         []data.Quote{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APISearchResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Meta.Limit != 50 {
					t.Errorf("expected default limit 50, got %d", resp.Meta.Limit)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockSearchStore{links: tt.links, quotes: tt.quotes, err: tt.storeErr}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			method := http.MethodGet
			if tt.name == "method not allowed for POST" {
				method = http.MethodPost
			}

			req := httptest.NewRequest(method, tt.path, nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()

			handler.APIv1SearchHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

func TestAPIv1_Search_StoreError(t *testing.T) {
	tests := []struct {
		name           string
		searchLinksFn  func(query string) ([]data.IRCLink, error)
		searchQuotesFn func(query string) ([]data.Quote, error)
		expectedStatus int
		expectedCode   string
	}{
		{
			name: "SearchIRCLinks error returns 500",
			searchLinksFn: func(query string) ([]data.IRCLink, error) {
				return nil, context.DeadlineExceeded
			},
			searchQuotesFn: func(query string) ([]data.Quote, error) {
				return []data.Quote{}, nil
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "internal_error",
		},
		{
			name: "SearchQuotes error returns 500",
			searchLinksFn: func(query string) ([]data.IRCLink, error) {
				return []data.IRCLink{}, nil
			},
			searchQuotesFn: func(query string) ([]data.Quote, error) {
				return nil, context.DeadlineExceeded
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockSearchStore{
				searchLinksFn:  tt.searchLinksFn,
				searchQuotesFn: tt.searchQuotesFn,
			}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=test", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()

			handler.APIv1SearchHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var resp APIErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if resp.Error.Code != tt.expectedCode {
				t.Errorf("expected code %s, got %s", tt.expectedCode, resp.Error.Code)
			}
		})
	}
}
