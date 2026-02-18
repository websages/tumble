package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tumble/internal/config"
	"tumble/internal/data"
)

// mockKittenStore is a mock implementation of data.Store for testing kitten handlers.
type mockKittenStore struct {
	data.Store
	getTodayImageByLinkFn    func(ctx context.Context, link string) (*data.Image, error)
	insertImageFn            func(ctx context.Context, image *data.Image) (int, error)
	deleteTodayImageByLinkFn func(ctx context.Context, link string) error
}

func (m *mockKittenStore) GetTodayImageByLink(ctx context.Context, link string) (*data.Image, error) {
	if m.getTodayImageByLinkFn != nil {
		return m.getTodayImageByLinkFn(ctx, link)
	}
	return nil, nil
}

func (m *mockKittenStore) InsertImage(ctx context.Context, image *data.Image) (int, error) {
	if m.insertImageFn != nil {
		return m.insertImageFn(ctx, image)
	}
	return 1, nil
}

func (m *mockKittenStore) DeleteTodayImageByLink(ctx context.Context, link string) error {
	if m.deleteTodayImageByLinkFn != nil {
		return m.deleteTodayImageByLinkFn(ctx, link)
	}
	return nil
}

func TestAPIv1_GetKittenDaily(t *testing.T) {
	testDate := time.Now().Format("2006-01-02")

	tests := []struct {
		name           string
		method         string
		getTodayFn     func(ctx context.Context, link string) (*data.Image, error)
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:   "returns kitten when exists",
			method: http.MethodGet,
			getTodayFn: func(ctx context.Context, link string) (*data.Image, error) {
				return &data.Image{
					ID:        1,
					URL:       "https://cataas.com/cat/abc123",
					Timestamp: time.Now(),
				}, nil
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIKittenResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.URL != "https://cataas.com/cat/abc123" {
					t.Errorf("expected URL 'https://cataas.com/cat/abc123', got %s", resp.URL)
				}
				if resp.Date != testDate {
					t.Errorf("expected date %s, got %s", testDate, resp.Date)
				}
				// GET should not have fetched field set to true
				if resp.Fetched {
					t.Error("expected fetched to be false for GET")
				}
			},
		},
		{
			name:   "returns 404 when no kitten",
			method: http.MethodGet,
			getTodayFn: func(ctx context.Context, link string) (*data.Image, error) {
				return nil, nil
			},
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "not_found" {
					t.Errorf("expected code 'not_found', got %s", resp.Error.Code)
				}
			},
		},
		{
			name:   "returns 500 on store error",
			method: http.MethodGet,
			getTodayFn: func(ctx context.Context, link string) (*data.Image, error) {
				return nil, errors.New("database error")
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
			store := &mockKittenStore{
				getTodayImageByLinkFn: tt.getTodayFn,
			}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			req := httptest.NewRequest(tt.method, "/api/v1/kittens/daily", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()

			handler.APIv1KittensDailyHandler(w, req)

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

func TestAPIv1_PutKittenDaily(t *testing.T) {
	testDate := time.Now().Format("2006-01-02")

	tests := []struct {
		name           string
		method         string
		remoteAddr     string
		apiKey         string
		adminSecret    string
		getTodayFn     func(ctx context.Context, link string) (*data.Image, error)
		insertImageFn  func(ctx context.Context, image *data.Image) (int, error)
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:        "returns existing kitten with fetched=false",
			method:      http.MethodPut,
			remoteAddr:  "127.0.0.1:12345",
			adminSecret: "secret123",
			getTodayFn: func(ctx context.Context, link string) (*data.Image, error) {
				return &data.Image{
					ID:        1,
					URL:       "https://cataas.com/cat/existing",
					Timestamp: time.Now(),
				}, nil
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIKittenResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.URL != "https://cataas.com/cat/existing" {
					t.Errorf("expected URL 'https://cataas.com/cat/existing', got %s", resp.URL)
				}
				if resp.Date != testDate {
					t.Errorf("expected date %s, got %s", testDate, resp.Date)
				}
				if resp.Fetched {
					t.Error("expected fetched to be false when kitten already exists")
				}
			},
		},
		{
			name:        "requires authorization - no key returns 403",
			method:      http.MethodPut,
			remoteAddr:  "192.168.1.100:12345",
			adminSecret: "secret123",
			getTodayFn: func(ctx context.Context, link string) (*data.Image, error) {
				return nil, nil
			},
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
			name:        "valid API key authorizes request",
			method:      http.MethodPut,
			remoteAddr:  "192.168.1.100:12345",
			apiKey:      "secret123",
			adminSecret: "secret123",
			getTodayFn: func(ctx context.Context, link string) (*data.Image, error) {
				return &data.Image{
					ID:        1,
					URL:       "https://cataas.com/cat/abc",
					Timestamp: time.Now(),
				}, nil
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIKittenResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.URL == "" {
					t.Error("expected URL to be set")
				}
			},
		},
		{
			name:        "wrong API key returns 403",
			method:      http.MethodPut,
			remoteAddr:  "192.168.1.100:12345",
			apiKey:      "wrongkey",
			adminSecret: "secret123",
			getTodayFn: func(ctx context.Context, link string) (*data.Image, error) {
				return nil, nil
			},
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
			name:       "localhost is authorized without key",
			method:     http.MethodPut,
			remoteAddr: "127.0.0.1:12345",
			getTodayFn: func(ctx context.Context, link string) (*data.Image, error) {
				return &data.Image{
					ID:        1,
					URL:       "https://cataas.com/cat/local",
					Timestamp: time.Now(),
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockKittenStore{
				getTodayImageByLinkFn: tt.getTodayFn,
				insertImageFn:         tt.insertImageFn,
			}
			handler := &Handler{
				Store: store,
				Config: &config.Config{
					AdminSecret: tt.adminSecret,
				},
			}

			req := httptest.NewRequest(tt.method, "/api/v1/kittens/daily", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			w := httptest.NewRecorder()

			handler.APIv1KittensDailyHandler(w, req)

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

func TestAPIv1_DeleteKittenDaily(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		remoteAddr     string
		apiKey         string
		adminSecret    string
		getTodayFn     func(ctx context.Context, link string) (*data.Image, error)
		deleteTodayFn  func(ctx context.Context, link string) error
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name:        "deletes kitten returns 204",
			method:      http.MethodDelete,
			remoteAddr:  "127.0.0.1:12345",
			adminSecret: "secret123",
			getTodayFn: func(ctx context.Context, link string) (*data.Image, error) {
				return &data.Image{
					ID:        1,
					URL:       "https://cataas.com/cat/todelete",
					Timestamp: time.Now(),
				}, nil
			},
			deleteTodayFn: func(ctx context.Context, link string) error {
				return nil
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:        "returns 404 when no kitten to delete",
			method:      http.MethodDelete,
			remoteAddr:  "127.0.0.1:12345",
			adminSecret: "secret123",
			getTodayFn: func(ctx context.Context, link string) (*data.Image, error) {
				return nil, nil
			},
			expectedStatus: http.StatusNotFound,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIErrorResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Error.Code != "not_found" {
					t.Errorf("expected code 'not_found', got %s", resp.Error.Code)
				}
			},
		},
		{
			name:        "requires authorization - no key returns 403",
			method:      http.MethodDelete,
			remoteAddr:  "192.168.1.100:12345",
			adminSecret: "secret123",
			getTodayFn: func(ctx context.Context, link string) (*data.Image, error) {
				return &data.Image{ID: 1}, nil
			},
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
			name:        "valid API key authorizes delete",
			method:      http.MethodDelete,
			remoteAddr:  "192.168.1.100:12345",
			apiKey:      "secret123",
			adminSecret: "secret123",
			getTodayFn: func(ctx context.Context, link string) (*data.Image, error) {
				return &data.Image{
					ID:        1,
					URL:       "https://cataas.com/cat/authed",
					Timestamp: time.Now(),
				}, nil
			},
			deleteTodayFn: func(ctx context.Context, link string) error {
				return nil
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:       "localhost is authorized without key",
			method:     http.MethodDelete,
			remoteAddr: "127.0.0.1:12345",
			getTodayFn: func(ctx context.Context, link string) (*data.Image, error) {
				return &data.Image{ID: 1}, nil
			},
			deleteTodayFn: func(ctx context.Context, link string) error {
				return nil
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:       "returns 500 on delete error",
			method:     http.MethodDelete,
			remoteAddr: "127.0.0.1:12345",
			getTodayFn: func(ctx context.Context, link string) (*data.Image, error) {
				return &data.Image{ID: 1}, nil
			},
			deleteTodayFn: func(ctx context.Context, link string) error {
				return errors.New("database error")
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
			store := &mockKittenStore{
				getTodayImageByLinkFn:    tt.getTodayFn,
				deleteTodayImageByLinkFn: tt.deleteTodayFn,
			}
			handler := &Handler{
				Store: store,
				Config: &config.Config{
					AdminSecret: tt.adminSecret,
				},
			}

			req := httptest.NewRequest(tt.method, "/api/v1/kittens/daily", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.apiKey != "" {
				req.Header.Set("X-API-Key", tt.apiKey)
			}
			w := httptest.NewRecorder()

			handler.APIv1KittensDailyHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			// 204 No Content should not have Content-Type header
			if tt.expectedStatus != http.StatusNoContent {
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", contentType)
				}
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.Bytes())
			}
		})
	}
}

func TestAPIv1_KittensDailyHandler_MethodNotAllowed(t *testing.T) {
	tests := []struct {
		name   string
		method string
	}{
		{"POST not allowed", http.MethodPost},
		{"PATCH not allowed", http.MethodPatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockKittenStore{}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			req := httptest.NewRequest(tt.method, "/api/v1/kittens/daily", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()

			handler.APIv1KittensDailyHandler(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
			}

			var resp APIErrorResponse
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			if resp.Error.Code != "method_not_allowed" {
				t.Errorf("expected code 'method_not_allowed', got %s", resp.Error.Code)
			}
		})
	}
}
