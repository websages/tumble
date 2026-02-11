package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"tumble/internal/config"
)

// mockRedirectStore is a mock implementation for testing redirect handler.
type mockRedirectStore struct {
	mockAPIStore
	linkURL         string
	linkURLErr      error
	linkURLFn       func(id int) (string, error)
	incrementCalled atomic.Int32
	incrementFn     func(id int) error
}

func (m *mockRedirectStore) GetIRCLinkURL(ctx context.Context, id int) (string, error) {
	if m.linkURLFn != nil {
		return m.linkURLFn(id)
	}
	if m.linkURLErr != nil {
		return "", m.linkURLErr
	}
	return m.linkURL, nil
}

func (m *mockRedirectStore) IncrementClicks(ctx context.Context, id int) error {
	m.incrementCalled.Add(1)
	if m.incrementFn != nil {
		return m.incrementFn(id)
	}
	return nil
}

func TestAPIv1_RedirectHandler(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		linkURL          string
		linkURLErr       error
		linkURLFn        func(id int) (string, error)
		expectedStatus   int
		expectedLocation string
		checkBody        func(t *testing.T, body string)
	}{
		{
			name:             "valid ID redirects with 302",
			path:             "/go/123",
			linkURL:          "https://example.com/article",
			expectedStatus:   http.StatusFound,
			expectedLocation: "https://example.com/article",
		},
		{
			name:             "valid ID with http scheme redirects",
			path:             "/go/456",
			linkURL:          "http://example.com/page",
			expectedStatus:   http.StatusFound,
			expectedLocation: "http://example.com/page",
		},
		{
			name:           "invalid ID returns 400",
			path:           "/go/abc",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				if body != "Invalid ID\n" {
					t.Errorf("expected 'Invalid ID\\n', got %q", body)
				}
			},
		},
		{
			name:           "empty ID returns 400",
			path:           "/go/",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				if body != "Invalid ID\n" {
					t.Errorf("expected 'Invalid ID\\n', got %q", body)
				}
			},
		},
		{
			name:           "negative ID returns 400",
			path:           "/go/-5",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				if body != "Invalid ID\n" {
					t.Errorf("expected 'Invalid ID\\n', got %q", body)
				}
			},
		},
		{
			name:           "non-existent link returns 404",
			path:           "/go/999",
			linkURLErr:     errors.New("link not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "store returns not found error returns 404",
			path: "/go/888",
			linkURLFn: func(id int) (string, error) {
				return "", errors.New("record not found")
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "javascript scheme is blocked",
			path:           "/go/123",
			linkURL:        "javascript:alert(1)",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				if body != "Invalid redirect URL\n" {
					t.Errorf("expected 'Invalid redirect URL\\n', got %q", body)
				}
			},
		},
		{
			name:           "data scheme is blocked",
			path:           "/go/123",
			linkURL:        "data:text/html,<script>alert(1)</script>",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				if body != "Invalid redirect URL\n" {
					t.Errorf("expected 'Invalid redirect URL\\n', got %q", body)
				}
			},
		},
		{
			name:           "file scheme is blocked",
			path:           "/go/123",
			linkURL:        "file:///etc/passwd",
			expectedStatus: http.StatusBadRequest,
			checkBody: func(t *testing.T, body string) {
				if body != "Invalid redirect URL\n" {
					t.Errorf("expected 'Invalid redirect URL\\n', got %q", body)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockRedirectStore{
				linkURL:    tt.linkURL,
				linkURLErr: tt.linkURLErr,
				linkURLFn:  tt.linkURLFn,
			}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			handler.APIv1RedirectHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedLocation != "" {
				location := w.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("expected Location %q, got %q", tt.expectedLocation, location)
				}
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.String())
			}
		})
	}
}

func TestAPIv1_RedirectHandler_ClickTracking(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		sigQueryParam   string
		clickSigningKey string
		linkURL         string
		expectIncrement bool
		expectedStatus  int
	}{
		{
			name:            "valid signature increments clicks",
			path:            "/go/123",
			clickSigningKey: "testsecret",
			linkURL:         "https://example.com",
			expectIncrement: true,
			expectedStatus:  http.StatusFound,
		},
		{
			name:            "invalid signature does not increment",
			path:            "/go/123",
			sigQueryParam:   "invalidsig",
			clickSigningKey: "testsecret",
			linkURL:         "https://example.com",
			expectIncrement: false,
			expectedStatus:  http.StatusFound,
		},
		{
			name:            "missing signature does not increment",
			path:            "/go/123",
			sigQueryParam:   "",
			clickSigningKey: "testsecret",
			linkURL:         "https://example.com",
			expectIncrement: false,
			expectedStatus:  http.StatusFound,
		},
		{
			name:            "no signing key configured does not increment",
			path:            "/go/123",
			clickSigningKey: "",
			linkURL:         "https://example.com",
			expectIncrement: false,
			expectedStatus:  http.StatusFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockRedirectStore{
				linkURL: tt.linkURL,
			}
			handler := &Handler{
				Store: store,
				Config: &config.Config{
					ClickSigningKey: tt.clickSigningKey,
				},
			}

			// Generate valid signature if needed
			path := tt.path
			if tt.expectIncrement && tt.clickSigningKey != "" {
				// Generate a valid signature for this test
				sig := GenerateClickSignature(123, tt.clickSigningKey)
				path = tt.path + "?sig=" + sig
			} else if tt.sigQueryParam != "" {
				path = tt.path + "?sig=" + tt.sigQueryParam
			}

			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			handler.APIv1RedirectHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Give async goroutine time to execute
			time.Sleep(10 * time.Millisecond)

			incrementCount := store.incrementCalled.Load()
			if tt.expectIncrement && incrementCount == 0 {
				t.Error("expected IncrementClicks to be called, but it wasn't")
			}
			if !tt.expectIncrement && incrementCount > 0 {
				t.Error("expected IncrementClicks NOT to be called, but it was")
			}
		})
	}
}

func TestAPIv1_RedirectHandler_MethodNotAllowed(t *testing.T) {
	store := &mockRedirectStore{
		linkURL: "https://example.com",
	}
	handler := &Handler{
		Store:  store,
		Config: &config.Config{},
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/go/123", nil)
			w := httptest.NewRecorder()

			handler.APIv1RedirectHandler(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d for %s, got %d", http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

func TestAPIv1_RedirectHandler_HeadMethod(t *testing.T) {
	store := &mockRedirectStore{
		linkURL: "https://example.com",
	}
	handler := &Handler{
		Store:  store,
		Config: &config.Config{},
	}

	req := httptest.NewRequest(http.MethodHead, "/go/123", nil)
	w := httptest.NewRecorder()

	handler.APIv1RedirectHandler(w, req)

	// HEAD should work like GET for redirects
	if w.Code != http.StatusFound {
		t.Errorf("expected status %d for HEAD, got %d", http.StatusFound, w.Code)
	}

	location := w.Header().Get("Location")
	if location != "https://example.com" {
		t.Errorf("expected Location https://example.com, got %q", location)
	}
}
