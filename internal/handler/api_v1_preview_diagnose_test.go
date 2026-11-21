package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"tumble/internal/config"
	"tumble/internal/data"
)

type mockDiagnoseStore struct {
	data.Store
	cached  *data.LinkPreview
	deleted string
}

func (m *mockDiagnoseStore) GetLinkPreview(ctx context.Context, u string) (*data.LinkPreview, error) {
	return m.cached, nil
}

func (m *mockDiagnoseStore) DeleteLinkPreview(ctx context.Context, u string) error {
	m.deleted = u
	return nil
}

func TestAPIv1PreviewDiagnose(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.UserAgent(), "Chrome") {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		io.WriteString(w, `<html><head><title>Alive</title></head></html>`)
	}))
	defer srv.Close()

	newHandler := func(store *mockDiagnoseStore) *Handler {
		return &Handler{Store: store, Config: &config.Config{
			AdminSecret:    "secret",
			RequestTimeout: 5 * time.Second,
		}}
	}
	target := srv.URL + "/post"
	path := "/api/v1/preview/diagnose?url=" + url.QueryEscape(target)

	t.Run("requires API key", func(t *testing.T) {
		w := httptest.NewRecorder()
		newHandler(&mockDiagnoseStore{}).APIv1PreviewDiagnoseHandler(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", w.Code)
		}
	})

	t.Run("rejects non-http url", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/preview/diagnose?url=file:///etc/passwd", nil)
		req.Header.Set("X-API-Key", "secret")
		w := httptest.NewRecorder()
		newHandler(&mockDiagnoseStore{}).APIv1PreviewDiagnoseHandler(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", w.Code)
		}
	})

	t.Run("probes each user agent and refreshes preview", func(t *testing.T) {
		store := &mockDiagnoseStore{cached: &data.LinkPreview{
			Data:      []byte(`{"error":"HTTP Error","status":"503"}`),
			UpdatedAt: time.Now(),
		}}
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-API-Key", "secret")
		w := httptest.NewRecorder()
		newHandler(store).APIv1PreviewDiagnoseHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
		}
		var resp DiagnoseResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if len(resp.Attempts) != 2*len(scrapeUserAgents) {
			t.Errorf("attempts = %d, want %d", len(resp.Attempts), 2*len(scrapeUserAgents))
		}
		if resp.Attempts[0].UserAgent != "browser" || resp.Attempts[0].Status != 503 {
			t.Errorf("first attempt = %+v, want browser/503", resp.Attempts[0])
		}
		if resp.Attempts[2].UserAgent != "plain" || resp.Attempts[2].Status != 200 {
			t.Errorf("third attempt = %+v, want plain/200", resp.Attempts[2])
		}
		if resp.CachedEntry == nil {
			t.Error("expected cached_entry to be reported")
		}
		if store.deleted != target {
			t.Errorf("deleted = %q, want %q", store.deleted, target)
		}
		if resp.Preview["title"] != "Alive" {
			t.Errorf("preview = %v, want title Alive", resp.Preview)
		}
	})
}
