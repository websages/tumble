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

// mockAPIStore is a mock implementation of data.Store for testing API handlers.
// It embeds a nil data.Store to satisfy the interface, and we override the
// methods we need for testing.
type mockAPIStore struct {
	data.Store
	links []data.IRCLink
	err   error
}

func (m *mockAPIStore) GetRecentIRCLinks(ctx context.Context, days int, offsetDays int) ([]data.IRCLink, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.links, nil
}

func TestAPIv1_ListLinks(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		method         string
		path           string
		links          []data.IRCLink
		storeErr       error
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:   "returns empty list",
			method: http.MethodGet,
			path:   "/api/v1/links",
			links:  []data.IRCLink{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APILinksResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Data) != 0 {
					t.Errorf("expected 0 links, got %d", len(resp.Data))
				}
				if resp.Meta.Total != 0 {
					t.Errorf("expected total 0, got %d", resp.Meta.Total)
				}
				if resp.Meta.Limit != 50 {
					t.Errorf("expected limit 50, got %d", resp.Meta.Limit)
				}
				if resp.Meta.Offset != 0 {
					t.Errorf("expected offset 0, got %d", resp.Meta.Offset)
				}
			},
		},
		{
			name:   "returns links with proper structure",
			method: http.MethodGet,
			path:   "/api/v1/links",
			links: []data.IRCLink{
				{
					ID:          1,
					Timestamp:   now,
					User:        "testuser",
					Title:       "Test Link",
					URL:         "https://example.com",
					Clicks:      10,
					ContentType: "",
				},
				{
					ID:          2,
					Timestamp:   now.Add(-time.Hour),
					User:        "anotheruser",
					Title:       "Another Link",
					URL:         "https://example.org",
					Clicks:      5,
					ContentType: "image",
				},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APILinksResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Data) != 2 {
					t.Fatalf("expected 2 links, got %d", len(resp.Data))
				}
				if resp.Meta.Total != 2 {
					t.Errorf("expected total 2, got %d", resp.Meta.Total)
				}

				// Check first link structure
				link := resp.Data[0]
				if link.ID != 1 {
					t.Errorf("expected ID 1, got %d", link.ID)
				}
				if link.User != "testuser" {
					t.Errorf("expected user testuser, got %s", link.User)
				}
				if link.Title != "Test Link" {
					t.Errorf("expected title Test Link, got %s", link.Title)
				}
				if link.URL != "https://example.com" {
					t.Errorf("expected URL https://example.com, got %s", link.URL)
				}
				if link.Clicks != 10 {
					t.Errorf("expected clicks 10, got %d", link.Clicks)
				}
			},
		},
		{
			name:   "respects limit parameter",
			method: http.MethodGet,
			path:   "/api/v1/links?limit=1",
			links: []data.IRCLink{
				{ID: 1, Timestamp: now, User: "user1", Title: "Link 1", URL: "https://example.com/1"},
				{ID: 2, Timestamp: now, User: "user2", Title: "Link 2", URL: "https://example.com/2"},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APILinksResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Data) != 1 {
					t.Errorf("expected 1 link, got %d", len(resp.Data))
				}
				if resp.Meta.Total != 2 {
					t.Errorf("expected total 2, got %d", resp.Meta.Total)
				}
				if resp.Meta.Limit != 1 {
					t.Errorf("expected limit 1, got %d", resp.Meta.Limit)
				}
			},
		},
		{
			name:   "respects offset parameter",
			method: http.MethodGet,
			path:   "/api/v1/links?offset=1",
			links: []data.IRCLink{
				{ID: 1, Timestamp: now, User: "user1", Title: "Link 1", URL: "https://example.com/1"},
				{ID: 2, Timestamp: now, User: "user2", Title: "Link 2", URL: "https://example.com/2"},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APILinksResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Data) != 1 {
					t.Errorf("expected 1 link, got %d", len(resp.Data))
				}
				if resp.Meta.Offset != 1 {
					t.Errorf("expected offset 1, got %d", resp.Meta.Offset)
				}
				// First item should be the second link
				if resp.Data[0].ID != 2 {
					t.Errorf("expected link ID 2, got %d", resp.Data[0].ID)
				}
			},
		},
		{
			name:   "limit capped at 1000",
			method: http.MethodGet,
			path:   "/api/v1/links?limit=5000",
			links:  []data.IRCLink{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APILinksResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Meta.Limit != 1000 {
					t.Errorf("expected limit capped at 1000, got %d", resp.Meta.Limit)
				}
			},
		},
		{
			name:   "invalid limit uses default",
			method: http.MethodGet,
			path:   "/api/v1/links?limit=abc",
			links:  []data.IRCLink{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APILinksResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Meta.Limit != 50 {
					t.Errorf("expected default limit 50, got %d", resp.Meta.Limit)
				}
			},
		},
		{
			name:   "offset beyond range returns empty",
			method: http.MethodGet,
			path:   "/api/v1/links?offset=100",
			links: []data.IRCLink{
				{ID: 1, Timestamp: now, User: "user1", Title: "Link 1", URL: "https://example.com/1"},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APILinksResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Data) != 0 {
					t.Errorf("expected 0 links, got %d", len(resp.Data))
				}
				if resp.Meta.Total != 1 {
					t.Errorf("expected total 1, got %d", resp.Meta.Total)
				}
			},
		},
		{
			name:           "method not allowed for PUT",
			method:         http.MethodPut,
			path:           "/api/v1/links",
			links:          []data.IRCLink{},
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
			name:   "works with .json suffix",
			method: http.MethodGet,
			path:   "/api/v1/links.json",
			links: []data.IRCLink{
				{ID: 1, Timestamp: now, User: "user1", Title: "Link 1", URL: "https://example.com/1"},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APILinksResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Data) != 1 {
					t.Errorf("expected 1 link, got %d", len(resp.Data))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockAPIStore{links: tt.links, err: tt.storeErr}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.RemoteAddr = "127.0.0.1:12345" // Localhost for auth bypass
			w := httptest.NewRecorder()

			handler.APIv1LinksHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
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

func TestAPIv1_ListLinks_StoreError(t *testing.T) {
	store := &mockAPIStore{
		links: nil,
		err:   context.DeadlineExceeded,
	}
	handler := &Handler{
		Store:  store,
		Config: &config.Config{},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/links", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.APIv1LinksHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp APIErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error.Code != "internal_error" {
		t.Errorf("expected code internal_error, got %s", resp.Error.Code)
	}
}
