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

// mockRelatedStore is a mock implementation of data.Store for testing
// the related links endpoint.
type mockRelatedStore struct {
	data.Store
	linkByID     *data.IRCLink
	relatedLinks []data.RelatedLink
	tags         []data.Tag
}

func (m *mockRelatedStore) GetIRCLinkByID(ctx context.Context, id int) (*data.IRCLink, error) {
	return m.linkByID, nil
}

func (m *mockRelatedStore) FindRelatedLinks(ctx context.Context, title string, url string, excludeID int, limit int) ([]data.RelatedLink, error) {
	return m.relatedLinks, nil
}

func (m *mockRelatedStore) GetTagsByResource(ctx context.Context, resourceType string, resourceID int) ([]data.Tag, error) {
	return m.tags, nil
}

func TestAPIv1_GetRelatedLinks(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		path           string
		linkByID       *data.IRCLink
		relatedLinks   []data.RelatedLink
		tags           []data.Tag
		expectedStatus int
		checkBody      func(t *testing.T, body []byte)
	}{
		{
			name: "returns related links",
			path: "/api/v1/links/1/related",
			linkByID: &data.IRCLink{
				ID:        1,
				Timestamp: now,
				User:      "testuser",
				Title:     "Understanding Kubernetes Networking",
				URL:       "https://example.com/k8s",
				Clicks:    10,
			},
			relatedLinks: []data.RelatedLink{
				{
					ID:        2,
					Title:     "Kubernetes Networking Deep Dive",
					URL:       "https://example.com/k8s-deep",
					User:      "alice",
					Clicks:    20,
					Timestamp: now.Add(-48 * time.Hour),
				},
				{
					ID:        3,
					Title:     "Understanding Container Networking",
					URL:       "https://example.com/containers",
					User:      "bob",
					Clicks:    5,
					Timestamp: now.Add(-72 * time.Hour),
				},
			},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIRelatedResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if len(resp.Data) == 0 {
					t.Error("expected at least one related link")
				}
				// Verify structure of first result
				first := resp.Data[0]
				if first.ID == 0 {
					t.Error("expected non-zero ID")
				}
				if first.Title == "" {
					t.Error("expected non-empty title")
				}
				if first.URL == "" {
					t.Error("expected non-empty URL")
				}
				if first.Score <= 0 {
					t.Error("expected positive score")
				}
			},
		},
		{
			name:           "link not found",
			path:           "/api/v1/links/999/related",
			linkByID:       nil,
			relatedLinks:   nil,
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
			name: "no related links returns empty array",
			path: "/api/v1/links/1/related",
			linkByID: &data.IRCLink{
				ID:        1,
				Timestamp: now,
				User:      "testuser",
				Title:     "Unique Unrepeatable Topic XYZZY",
				URL:       "https://unique-domain-12345.com/page",
				Clicks:    0,
			},
			relatedLinks:   []data.RelatedLink{},
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body []byte) {
				var resp APIRelatedResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if resp.Data == nil {
					t.Error("expected empty array, got null")
				}
				if len(resp.Data) != 0 {
					t.Errorf("expected 0 related links, got %d", len(resp.Data))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockRelatedStore{
				linkByID:     tt.linkByID,
				relatedLinks: tt.relatedLinks,
				tags:         tt.tags,
			}
			handler := &Handler{
				Store:  store,
				Config: &config.Config{},
			}

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.RemoteAddr = "127.0.0.1:12345"
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
