package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tumble/internal/config"
	"tumble/internal/data"
)

// mockCacheStore is a mock implementation of data.Store for testing cache handlers.
type mockCacheStore struct {
	data.Store
	deleteLinkPreviewFn    func(url string) error
	deleteAllLinkPreviewFn func() (int, error)
	err                    error
}

func (m *mockCacheStore) DeleteLinkPreview(ctx context.Context, url string) error {
	if m.deleteLinkPreviewFn != nil {
		return m.deleteLinkPreviewFn(url)
	}
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *mockCacheStore) DeleteAllLinkPreviews(ctx context.Context) (int, error) {
	if m.deleteAllLinkPreviewFn != nil {
		return m.deleteAllLinkPreviewFn()
	}
	if m.err != nil {
		return 0, m.err
	}
	return 0, nil
}

func TestAPIv1_ClearCacheSpecificURL(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		remoteAddr     string
		apiKey         string
		adminSecret    string
		deleteFn       func(url string) error
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:           "clears specific URL returns cleared URL",
			method:         http.MethodDelete,
			path:           "/api/v1/cache?url=https://example.com/article",
			remoteAddr:     "127.0.0.1:12345",
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APICacheClearResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Cleared != "https://example.com/article" {
					t.Errorf("expected cleared 'https://example.com/article', got %s", resp.Cleared)
				}
			},
		},
		{
			name:           "localhost is authorized without key",
			method:         http.MethodDelete,
			path:           "/api/v1/cache?url=https://example.com/test",
			remoteAddr:     "127.0.0.1:12345",
			adminSecret:    "secret123",
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APICacheClearResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Cleared != "https://example.com/test" {
					t.Errorf("expected cleared URL, got %s", resp.Cleared)
				}
			},
		},
		{
			name:           "valid API key authorizes request",
			method:         http.MethodDelete,
			path:           "/api/v1/cache?url=https://example.com/test",
			remoteAddr:     "192.168.1.100:12345",
			apiKey:         "secret123",
			adminSecret:    "secret123",
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APICacheClearResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Cleared != "https://example.com/test" {
					t.Errorf("expected cleared URL, got %s", resp.Cleared)
				}
			},
		},
		{
			name:           "unauthorized without key returns 403",
			method:         http.MethodDelete,
			path:           "/api/v1/cache?url=https://example.com/test",
			remoteAddr:     "192.168.1.100:12345",
			adminSecret:    "secret123",
			expectedStatus: http.StatusForbidden,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "forbidden" {
					t.Errorf("expected code 'forbidden', got %s", resp.Error.Code)
				}
			},
		},
		{
			name:           "wrong API key returns 403",
			method:         http.MethodDelete,
			path:           "/api/v1/cache?url=https://example.com/test",
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
					t.Errorf("expected code 'forbidden', got %s", resp.Error.Code)
				}
			},
		},
		{
			name:           "only DELETE method allowed",
			method:         http.MethodGet,
			path:           "/api/v1/cache?url=https://example.com/test",
			remoteAddr:     "127.0.0.1:12345",
			expectedStatus: http.StatusMethodNotAllowed,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "method_not_allowed" {
					t.Errorf("expected code 'method_not_allowed', got %s", resp.Error.Code)
				}
			},
		},
		{
			name:           "POST method not allowed",
			method:         http.MethodPost,
			path:           "/api/v1/cache?url=https://example.com/test",
			remoteAddr:     "127.0.0.1:12345",
			expectedStatus: http.StatusMethodNotAllowed,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "method_not_allowed" {
					t.Errorf("expected code 'method_not_allowed', got %s", resp.Error.Code)
				}
			},
		},
		{
			name:       "store error returns 500",
			method:     http.MethodDelete,
			path:       "/api/v1/cache?url=https://example.com/test",
			remoteAddr: "127.0.0.1:12345",
			deleteFn: func(url string) error {
				return context.DeadlineExceeded
			},
			expectedStatus: http.StatusInternalServerError,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "internal_error" {
					t.Errorf("expected code 'internal_error', got %s", resp.Error.Code)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockCacheStore{
				deleteLinkPreviewFn: tt.deleteFn,
			}
			handler := &Handler{
				Store: store,
				Config: &config.Config{
					AdminSecret: tt.adminSecret,
				},
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			w := httptest.NewRecorder()

			handler.APIv1CacheHandler(w, req)

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

func TestAPIv1_ClearAllCache(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		remoteAddr     string
		deleteAllFn    func() (int, error)
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:       "clear all cache returns 'all' with count",
			method:     http.MethodDelete,
			path:       "/api/v1/cache",
			remoteAddr: "127.0.0.1:12345",
			deleteAllFn: func() (int, error) {
				return 47, nil
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APICacheClearResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Cleared != "all" {
					t.Errorf("expected cleared 'all', got %s", resp.Cleared)
				}
				if resp.Count != 47 {
					t.Errorf("expected count 47, got %d", resp.Count)
				}
			},
		},
		{
			name:       "clear all cache with zero entries",
			method:     http.MethodDelete,
			path:       "/api/v1/cache",
			remoteAddr: "127.0.0.1:12345",
			deleteAllFn: func() (int, error) {
				return 0, nil
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APICacheClearResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Cleared != "all" {
					t.Errorf("expected cleared 'all', got %s", resp.Cleared)
				}
				if resp.Count != 0 {
					t.Errorf("expected count 0, got %d", resp.Count)
				}
			},
		},
		{
			name:       "store error returns 500",
			method:     http.MethodDelete,
			path:       "/api/v1/cache",
			remoteAddr: "127.0.0.1:12345",
			deleteAllFn: func() (int, error) {
				return 0, context.DeadlineExceeded
			},
			expectedStatus: http.StatusInternalServerError,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "internal_error" {
					t.Errorf("expected code 'internal_error', got %s", resp.Error.Code)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockCacheStore{
				deleteAllLinkPreviewFn: tt.deleteAllFn,
			}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.RemoteAddr = tt.remoteAddr
			w := httptest.NewRecorder()

			handler.APIv1CacheHandler(w, req)

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
