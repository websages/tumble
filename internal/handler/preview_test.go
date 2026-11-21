package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"tumble/internal/config"
	"tumble/internal/data"
)

func TestIsPreviewExpired(t *testing.T) {
	tests := []struct {
		name          string
		data          string
		age           time.Duration
		linkAge       time.Duration // age of the link itself; 0 means zero-value timestamp
		zeroTimestamp bool          // if true, pass zero time
		expected      bool
	}{
		{
			name:     "fresh normal preview (1 day old)",
			data:     `{"title":"Example","description":"A page"}`,
			age:      24 * time.Hour,
			expected: false,
		},
		{
			name:     "stale normal preview (11 days old)",
			data:     `{"title":"Example","description":"A page"}`,
			age:      11 * 24 * time.Hour,
			expected: true,
		},
		{
			name:     "fresh error on recent link (1d error, 5d link) - expired (24h TTL)",
			data:     `{"error":"Video Unavailable","status":"404"}`,
			age:      24 * time.Hour,
			linkAge:  5 * 24 * time.Hour,
			expected: true,
		},
		{
			name:     "fresh error on recent link (12h error, 5d link) - not expired",
			data:     `{"error":"Video Unavailable","status":"404"}`,
			age:      12 * time.Hour,
			linkAge:  5 * 24 * time.Hour,
			expected: false,
		},
		{
			name:     "error on old link (30d error, 90d link) - not expired (60d TTL)",
			data:     `{"error":"Video Unavailable","status":"404"}`,
			age:      30 * 24 * time.Hour,
			linkAge:  90 * 24 * time.Hour,
			expected: false,
		},
		{
			name:     "stale error on old link (61d error, 90d link) - expired",
			data:     `{"error":"Video Unavailable","status":"404"}`,
			age:      61 * 24 * time.Hour,
			linkAge:  90 * 24 * time.Hour,
			expected: true,
		},
		{
			name:          "error with zero linkTimestamp uses old TTL - not expired at 30d",
			data:          `{"error":"Video Unavailable","status":"404"}`,
			age:           30 * 24 * time.Hour,
			zeroTimestamp: true,
			expected:      false,
		},
		{
			name:          "error with zero linkTimestamp uses old TTL - expired at 61d",
			data:          `{"error":"Video Unavailable","status":"404"}`,
			age:           61 * 24 * time.Hour,
			zeroTimestamp: true,
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cached := &data.LinkPreview{
				URL:       "https://example.com",
				Data:      []byte(tt.data),
				UpdatedAt: time.Now().Add(-tt.age),
			}
			var linkTimestamp time.Time
			if !tt.zeroTimestamp && tt.linkAge > 0 {
				linkTimestamp = time.Now().Add(-tt.linkAge)
			}
			result := isPreviewExpired(cached, linkTimestamp)
			if result != tt.expected {
				t.Errorf("isPreviewExpired() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsPreviewExpiredTransientError(t *testing.T) {
	recent := time.Now().Add(-2 * 24 * time.Hour)
	tests := []struct {
		name     string
		data     string
		age      time.Duration
		expected bool
	}{
		{"503 on recent link expires after 1h", `{"error":"HTTP Error","status":"503"}`, 2 * time.Hour, true},
		{"429 on recent link fresh at 30m", `{"error":"HTTP Error","status":"429"}`, 30 * time.Minute, false},
		{"404 on recent link keeps 24h TTL", `{"error":"HTTP Error","status":"404"}`, 2 * time.Hour, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cached := &data.LinkPreview{Data: []byte(tt.data), UpdatedAt: time.Now().Add(-tt.age)}
			if got := isPreviewExpired(cached, recent); got != tt.expected {
				t.Errorf("isPreviewExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestScrapeWithFallbacks(t *testing.T) {
	const page = `<html><head><title>Hello</title></head></html>`
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantTitle  string
		wantErr    string
		wantProbes int
	}{
		{
			name: "browser UA blocked with 503, plain UA succeeds",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.UserAgent(), "Chrome") {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				w.Header().Set("Content-Type", "text/html")
				io.WriteString(w, page)
			},
			wantTitle:  "Hello",
			wantProbes: 2,
		},
		{
			name: "404 is definitive and not retried",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr:    "status 404",
			wantProbes: 1,
		},
		{
			name: "all blocked reports first status",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(r.UserAgent(), "Chrome") {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				w.WriteHeader(http.StatusForbidden)
			},
			wantErr:    "status 503",
			wantProbes: 3,
		},
		{
			name: "PDF is described without parsing",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/pdf")
				io.WriteString(w, "%PDF-1.7 binary")
			},
			wantTitle:  "doc.pdf",
			wantProbes: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			probes := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				probes++
				tt.handler(w, r)
			}))
			defer srv.Close()

			h := &Handler{Config: &config.Config{RequestTimeout: 5 * time.Second}}
			meta, err := h.scrapeWithFallbacks(srv.URL + "/doc.pdf")
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			} else if meta["title"] != tt.wantTitle {
				t.Errorf("title = %q, want %q", meta["title"], tt.wantTitle)
			}
			if probes != tt.wantProbes {
				t.Errorf("probes = %d, want %d", probes, tt.wantProbes)
			}
		})
	}
}

func TestCanonicalTweetURL(t *testing.T) {
	tests := map[string]string{
		"https://x.com/a/status/123/photo/1":        "https://x.com/a/status/123",
		"https://twitter.com/a/status/123/video/2/": "https://twitter.com/a/status/123",
		"https://x.com/a/status/123":                "https://x.com/a/status/123",
	}
	for in, want := range tests {
		if got := canonicalTweetURL(in); got != want {
			t.Errorf("canonicalTweetURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRawSnapshotURL(t *testing.T) {
	in := "http://web.archive.org/web/20261106053822/https://www.surfer.com/news/x"
	want := "http://web.archive.org/web/20261106053822id_/https://www.surfer.com/news/x"
	if got := rawSnapshotURL(in); got != want {
		t.Errorf("rawSnapshotURL() = %q, want %q", got, want)
	}
}
