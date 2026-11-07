package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSiteIconHandler(t *testing.T) {
	builtin := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	tests := []struct {
		name         string
		iconURL      string
		path         string
		wantStatus   int
		wantLocation string
	}{
		{"no override serves built-in", "", "/favicon.ico", http.StatusTeapot, ""},
		{"override redirects favicon", "https://example.com/i.png", "/favicon.ico", http.StatusFound, "https://example.com/i.png"},
		{"override redirects touch icon", "/themes/corkboard/favicon.png", "/apple-touch-icon.png", http.StatusFound, "/themes/corkboard/favicon.png"},
		{"override of the same path does not loop", "/favicon.ico", "/favicon.ico", http.StatusTeapot, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			siteIconHandler(tt.iconURL, builtin).ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if got := w.Header().Get("Location"); got != tt.wantLocation {
				t.Errorf("Location = %q, want %q", got, tt.wantLocation)
			}
		})
	}
}
