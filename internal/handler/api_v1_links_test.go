package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
	links          []data.IRCLink
	linkByID       *data.IRCLink
	linkByIDFn     func(id int) (*data.IRCLink, error)
	linksByURL     []data.IRCLink
	linksByURLFn   func(url string) ([]data.IRCLink, error)
	insertedLinkID int
	insertLinkFn   func(user, title, url, contentType string) (int, error)
	deleteLinkFn   func(id int) error
	err            error
}

func (m *mockAPIStore) GetRecentIRCLinks(ctx context.Context, days int, offsetDays int) ([]data.IRCLink, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.links, nil
}

func (m *mockAPIStore) GetIRCLinkByID(ctx context.Context, id int) (*data.IRCLink, error) {
	if m.linkByIDFn != nil {
		return m.linkByIDFn(id)
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.linkByID, nil
}

func (m *mockAPIStore) GetIRCLinksByURL(ctx context.Context, url string) ([]data.IRCLink, error) {
	if m.linksByURLFn != nil {
		return m.linksByURLFn(url)
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.linksByURL, nil
}

func (m *mockAPIStore) InsertIRCLink(ctx context.Context, user, title, url, contentType string) (int, error) {
	if m.insertLinkFn != nil {
		return m.insertLinkFn(user, title, url, contentType)
	}
	if m.err != nil {
		return 0, m.err
	}
	return m.insertedLinkID, nil
}

func (m *mockAPIStore) DeleteIRCLink(ctx context.Context, id int) error {
	if m.deleteLinkFn != nil {
		return m.deleteLinkFn(id)
	}
	if m.err != nil {
		return m.err
	}
	return nil
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
			name:           "returns empty list",
			method:         http.MethodGet,
			path:           "/api/v1/links",
			links:          []data.IRCLink{},
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
			name:           "limit capped at 1000",
			method:         http.MethodGet,
			path:           "/api/v1/links?limit=5000",
			links:          []data.IRCLink{},
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
			name:           "invalid limit uses default",
			method:         http.MethodGet,
			path:           "/api/v1/links?limit=abc",
			links:          []data.IRCLink{},
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

func TestAPIv1_GetLink(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		path           string
		linkByID       *data.IRCLink
		storeErr       error
		acceptHeader   string
		expectedStatus int
		expectedType   string
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name: "returns existing link",
			path: "/api/v1/links/1",
			linkByID: &data.IRCLink{
				ID:        1,
				Timestamp: now,
				User:      "testuser",
				Title:     "Test Link",
				URL:       "https://example.com",
				Clicks:    10,
			},
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APILinkResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID != 1 {
					t.Errorf("expected ID 1, got %d", resp.ID)
				}
				if resp.User != "testuser" {
					t.Errorf("expected user testuser, got %s", resp.User)
				}
				if resp.Title != "Test Link" {
					t.Errorf("expected title Test Link, got %s", resp.Title)
				}
				if resp.URL != "https://example.com" {
					t.Errorf("expected URL https://example.com, got %s", resp.URL)
				}
				if resp.Clicks != 10 {
					t.Errorf("expected clicks 10, got %d", resp.Clicks)
				}
			},
		},
		{
			name:           "returns 404 for non-existent link",
			path:           "/api/v1/links/999",
			linkByID:       nil,
			expectedStatus: http.StatusNotFound,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "not_found" {
					t.Errorf("expected code not_found, got %s", resp.Error.Code)
				}
				if resp.Error.Message != "Link not found" {
					t.Errorf("expected message 'Link not found', got %s", resp.Error.Message)
				}
			},
		},
		{
			name:           "returns 500 on store error",
			path:           "/api/v1/links/1",
			linkByID:       nil,
			storeErr:       context.DeadlineExceeded,
			expectedStatus: http.StatusInternalServerError,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "internal_error" {
					t.Errorf("expected code internal_error, got %s", resp.Error.Code)
				}
			},
		},
		{
			name: "returns plain text with Accept header",
			path: "/api/v1/links/1",
			linkByID: &data.IRCLink{
				ID:        1,
				Timestamp: now,
				User:      "testuser",
				Title:     "Test Link",
				URL:       "https://example.com",
				Clicks:    10,
			},
			acceptHeader:   "text/plain",
			expectedStatus: http.StatusOK,
			expectedType:   "text/plain",
			checkBody: func(t *testing.T, body []byte) {
				expected := "Test Link - https://example.com"
				if string(body) != expected {
					t.Errorf("expected %q, got %q", expected, string(body))
				}
			},
		},
		{
			name: "returns plain text with .txt suffix",
			path: "/api/v1/links/1.txt",
			linkByID: &data.IRCLink{
				ID:        1,
				Timestamp: now,
				User:      "testuser",
				Title:     "Test Link",
				URL:       "https://example.com",
				Clicks:    10,
			},
			expectedStatus: http.StatusOK,
			expectedType:   "text/plain",
			checkBody: func(t *testing.T, body []byte) {
				expected := "Test Link - https://example.com"
				if string(body) != expected {
					t.Errorf("expected %q, got %q", expected, string(body))
				}
			},
		},
		{
			name:           "returns 400 for invalid ID",
			path:           "/api/v1/links/abc",
			expectedStatus: http.StatusBadRequest,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "invalid_id" {
					t.Errorf("expected code invalid_id, got %s", resp.Error.Code)
				}
			},
		},
		{
			name: "works with .json suffix",
			path: "/api/v1/links/1.json",
			linkByID: &data.IRCLink{
				ID:        1,
				Timestamp: now,
				User:      "testuser",
				Title:     "Test Link",
				URL:       "https://example.com",
				Clicks:    10,
			},
			expectedStatus: http.StatusOK,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APILinkResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID != 1 {
					t.Errorf("expected ID 1, got %d", resp.ID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockAPIStore{linkByID: tt.linkByID, err: tt.storeErr}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.RemoteAddr = "127.0.0.1:12345" // Localhost for auth bypass
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			w := httptest.NewRecorder()

			handler.APIv1LinksHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != tt.expectedType {
				t.Errorf("expected Content-Type %s, got %s", tt.expectedType, contentType)
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

func TestAPIv1_CreateLink(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		body           string
		linksByURL     []data.IRCLink
		insertedID     int
		storeErr       error
		acceptHeader   string
		expectedStatus int
		expectedType   string
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:           "valid link returns 201",
			body:           `{"url":"https://example.com/article","user":"testuser"}`,
			linksByURL:     []data.IRCLink{},
			insertedID:     42,
			expectedStatus: http.StatusCreated,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APILinkCreateResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID != 42 {
					t.Errorf("expected ID 42, got %d", resp.ID)
				}
				if resp.URL != "https://example.com/article" {
					t.Errorf("expected URL https://example.com/article, got %s", resp.URL)
				}
				if resp.User != "testuser" {
					t.Errorf("expected user testuser, got %s", resp.User)
				}
				if resp.IsDuplicate {
					t.Errorf("expected is_duplicate false, got true")
				}
				if len(resp.PreviousSubmissions) != 0 {
					t.Errorf("expected no previous submissions, got %d", len(resp.PreviousSubmissions))
				}
			},
		},
		{
			name:           "missing url returns 422",
			body:           `{"user":"testuser"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp ValidationErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "validation_error" {
					t.Errorf("expected code validation_error, got %s", resp.Error.Code)
				}
				if resp.Error.Details["url"] == "" {
					t.Errorf("expected url validation error")
				}
			},
		},
		{
			name:           "missing user returns 422",
			body:           `{"url":"https://example.com"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp ValidationErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "validation_error" {
					t.Errorf("expected code validation_error, got %s", resp.Error.Code)
				}
				if resp.Error.Details["user"] == "" {
					t.Errorf("expected user validation error")
				}
			},
		},
		{
			name:           "invalid url scheme returns 422",
			body:           `{"url":"javascript:alert(1)","user":"testuser"}`,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp ValidationErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "validation_error" {
					t.Errorf("expected code validation_error, got %s", resp.Error.Code)
				}
				if resp.Error.Details["url"] == "" {
					t.Errorf("expected url validation error")
				}
			},
		},
		{
			name: "duplicate link returns 201 with is_duplicate true",
			body: `{"url":"https://example.com/article","user":"newuser"}`,
			linksByURL: []data.IRCLink{
				{
					ID:        10,
					Timestamp: now.Add(-24 * time.Hour),
					User:      "olduser",
					Title:     "Old Title",
					URL:       "https://example.com/article",
				},
			},
			insertedID:     42,
			expectedStatus: http.StatusCreated,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APILinkCreateResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID != 42 {
					t.Errorf("expected ID 42, got %d", resp.ID)
				}
				if !resp.IsDuplicate {
					t.Errorf("expected is_duplicate true, got false")
				}
				if len(resp.PreviousSubmissions) != 1 {
					t.Fatalf("expected 1 previous submission, got %d", len(resp.PreviousSubmissions))
				}
				prev := resp.PreviousSubmissions[0]
				if prev.ID != 10 {
					t.Errorf("expected previous ID 10, got %d", prev.ID)
				}
				if prev.User != "olduser" {
					t.Errorf("expected previous user olduser, got %s", prev.User)
				}
				if prev.Title != "Old Title" {
					t.Errorf("expected previous title Old Title, got %s", prev.Title)
				}
			},
		},
		{
			name:           "invalid JSON returns 400",
			body:           `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "invalid_request" {
					t.Errorf("expected code invalid_request, got %s", resp.Error.Code)
				}
			},
		},
		{
			name:           "plain text response for Accept: text/plain",
			body:           `{"url":"https://example.com/article","user":"testuser"}`,
			linksByURL:     []data.IRCLink{},
			insertedID:     42,
			acceptHeader:   "text/plain",
			expectedStatus: http.StatusCreated,
			expectedType:   "text/plain",
			checkBody: func(t *testing.T, body []byte) {
				expected := "Created link 42: https://example.com/article"
				if string(body) != expected {
					t.Errorf("expected %q, got %q", expected, string(body))
				}
			},
		},
		{
			name: "plain text response for duplicate",
			body: `{"url":"https://example.com/article","user":"testuser"}`,
			linksByURL: []data.IRCLink{
				{
					ID:        10,
					Timestamp: now.Add(-24 * time.Hour),
					User:      "olduser",
					Title:     "Old Title",
					URL:       "https://example.com/article",
				},
			},
			insertedID:     42,
			acceptHeader:   "text/plain",
			expectedStatus: http.StatusCreated,
			expectedType:   "text/plain",
			checkBody: func(t *testing.T, body []byte) {
				expected := "Created link 42: https://example.com/article (duplicate of link 10 by olduser)"
				if string(body) != expected {
					t.Errorf("expected %q, got %q", expected, string(body))
				}
			},
		},
		{
			name:           "http url is valid",
			body:           `{"url":"http://example.com/article","user":"testuser"}`,
			linksByURL:     []data.IRCLink{},
			insertedID:     43,
			expectedStatus: http.StatusCreated,
			expectedType:   "application/json",
			checkBody: func(t *testing.T, body []byte) {
				var resp APILinkCreateResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.ID != 43 {
					t.Errorf("expected ID 43, got %d", resp.ID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockAPIStore{
				linksByURL:     tt.linksByURL,
				insertedLinkID: tt.insertedID,
				err:            tt.storeErr,
			}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/links", strings.NewReader(tt.body))
			req.RemoteAddr = "127.0.0.1:12345" // Localhost for auth bypass
			req.Header.Set("Content-Type", "application/json")
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			w := httptest.NewRecorder()

			handler.APIv1LinksHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != tt.expectedType {
				t.Errorf("expected Content-Type %s, got %s", tt.expectedType, contentType)
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

func TestAPIv1_CreateLink_StoreError(t *testing.T) {
	store := &mockAPIStore{
		linksByURL: []data.IRCLink{},
		insertLinkFn: func(user, title, url, contentType string) (int, error) {
			return 0, context.DeadlineExceeded
		},
	}
	handler := &Handler{
		Store:  store,
		Config: &config.Config{},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/links", strings.NewReader(`{"url":"https://example.com","user":"testuser"}`))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Content-Type", "application/json")
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

func TestAPIv1_DeleteLink(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		path           string
		linkByID       *data.IRCLink
		linkByIDFn     func(id int) (*data.IRCLink, error)
		deleteLinkFn   func(id int) error
		remoteAddr     string
		apiKey         string
		adminSecret    string
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name: "authorized delete returns 204",
			path: "/api/v1/links/1",
			linkByID: &data.IRCLink{
				ID:        1,
				Timestamp: now,
				User:      "testuser",
				Title:     "Test Link",
				URL:       "https://example.com",
			},
			remoteAddr:     "127.0.0.1:12345",
			expectedStatus: http.StatusNoContent,
			checkBody: func(t *testing.T, body []byte) {
				if len(body) != 0 {
					t.Errorf("expected empty body, got %q", string(body))
				}
			},
		},
		{
			name:           "unauthorized (no header) returns 403",
			path:           "/api/v1/links/1",
			linkByID:       &data.IRCLink{ID: 1},
			remoteAddr:     "192.168.1.100:12345",
			adminSecret:    "secret123",
			expectedStatus: http.StatusForbidden,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "forbidden" {
					t.Errorf("expected code forbidden, got %s", resp.Error.Code)
				}
			},
		},
		{
			name:           "unauthorized (wrong key) returns 403",
			path:           "/api/v1/links/1",
			linkByID:       &data.IRCLink{ID: 1},
			remoteAddr:     "192.168.1.100:12345",
			apiKey:         "wrongkey",
			adminSecret:    "secret123",
			expectedStatus: http.StatusForbidden,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "forbidden" {
					t.Errorf("expected code forbidden, got %s", resp.Error.Code)
				}
			},
		},
		{
			name:           "delete non-existent link returns 404",
			path:           "/api/v1/links/999",
			linkByID:       nil,
			remoteAddr:     "127.0.0.1:12345",
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "not_found" {
					t.Errorf("expected code not_found, got %s", resp.Error.Code)
				}
			},
		},
		{
			name: "localhost always authorized (no key needed)",
			path: "/api/v1/links/1",
			linkByID: &data.IRCLink{
				ID:        1,
				Timestamp: now,
				User:      "testuser",
				Title:     "Test Link",
				URL:       "https://example.com",
			},
			remoteAddr:     "127.0.0.1:12345",
			adminSecret:    "secret123",
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "valid API key authorizes delete",
			path: "/api/v1/links/1",
			linkByID: &data.IRCLink{
				ID:        1,
				Timestamp: now,
				User:      "testuser",
				Title:     "Test Link",
				URL:       "https://example.com",
			},
			remoteAddr:     "192.168.1.100:12345",
			apiKey:         "secret123",
			adminSecret:    "secret123",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid ID returns 400",
			path:           "/api/v1/links/abc",
			remoteAddr:     "127.0.0.1:12345",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "invalid_id" {
					t.Errorf("expected code invalid_id, got %s", resp.Error.Code)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockAPIStore{
				linkByID:     tt.linkByID,
				linkByIDFn:   tt.linkByIDFn,
				deleteLinkFn: tt.deleteLinkFn,
			}
			handler := &Handler{
				Store: store,
				Config: &config.Config{
					AdminSecret: tt.adminSecret,
				},
			}

			req := httptest.NewRequest(http.MethodDelete, tt.path, nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			w := httptest.NewRecorder()

			handler.APIv1LinksHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

func TestAPIv1_DeleteLink_StoreError(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		linkByIDFn     func(id int) (*data.IRCLink, error)
		deleteLinkFn   func(id int) error
		expectedStatus int
		expectedCode   string
	}{
		{
			name: "GetIRCLinkByID error returns 500",
			linkByIDFn: func(id int) (*data.IRCLink, error) {
				return nil, context.DeadlineExceeded
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "internal_error",
		},
		{
			name: "DeleteIRCLink error returns 500",
			linkByIDFn: func(id int) (*data.IRCLink, error) {
				return &data.IRCLink{
					ID:        1,
					Timestamp: now,
					User:      "testuser",
					Title:     "Test Link",
					URL:       "https://example.com",
				}, nil
			},
			deleteLinkFn: func(id int) error {
				return context.DeadlineExceeded
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockAPIStore{
				linkByIDFn:   tt.linkByIDFn,
				deleteLinkFn: tt.deleteLinkFn,
			}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/links/1", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()

			handler.APIv1LinksHandler(w, req)

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
